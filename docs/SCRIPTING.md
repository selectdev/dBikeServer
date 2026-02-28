# dBike Script Engine

Scripts are [Tengo](https://github.com/d5/tengo) files placed in this directory. Each file is compiled once at startup and executed on demand. The filename determines which IPC topic it handles — `ping.tengo` runs whenever a frame arrives with `topic: "ping"`.

Scripts run synchronously within the BLE handler goroutine. Keep them fast. Use `throttle()` to drop excess invocations.

---

## Table of contents

- [Script lifecycle](#script-lifecycle)
- [Tengo language basics](#tengo-language-basics)
- [Script context](#script-context)
- [Standard library](#standard-library)
- [Built-ins: BLE](#built-ins-ble)
- [Built-ins: Logging & time](#built-ins-logging--time)
- [Built-ins: Formatting & JSON](#built-ins-formatting--json)
- [Built-ins: Math](#built-ins-math)
- [Built-ins: Arrays](#built-ins-arrays)
- [Built-ins: Maps](#built-ins-maps)
- [Built-ins: Strings](#built-ins-strings)
- [Built-ins: Type predicates](#built-ins-type-predicates)
- [Built-ins: Encoding](#built-ins-encoding)
- [Built-ins: State](#built-ins-state)
- [Built-ins: Database](#built-ins-database)
- [Built-ins: GPIO](#built-ins-gpio)
- [Examples](#examples)

---

## Script lifecycle

1. On startup, every `*.tengo` file in `scripts/` is compiled into bytecode.
2. When a BLE frame arrives for topic `X`, the compiled bytecode for `X.tengo` is cloned and run with the frame's payload injected.
3. If no script exists for a topic, Go's built-in `switch` in `handler.go` handles it as a fallback.
4. Scripts share a persistent in-memory + on-disk state store. State written in one invocation is readable in the next.

> **Note:** If a script panics or returns an error, the error is logged and execution continues. The BLE connection is not dropped.

---

## Tengo language basics

Tengo is a statically-typed, dynamically-dispatched scripting language with Go-like syntax. Key differences from Go:

```tengo
// Declaration — use :=
x := 10
name := "hello"

// Assignment — use =
x = 20

// No trailing commas in maps or arrays
m := {a: 1, b: 2}   // ok
m := {a: 1, b: 2,}  // PARSE ERROR

// Undefined / default values
val := get_state("key") || 0   // use || for defaults

// String conversion
s := string(42)      // "42"
n := int("42")       // 42
f := float("3.14")   // 3.14

// Built-in iteration
for i, v in [1, 2, 3] {
    log(i, v)
}
for k, v in {a: 1, b: 2} {
    log(k, v)
}

// Closures (no captures from outer scope in Tengo — use state or pass args)
double := func(x) { return x * 2 }
log(double(5))  // 10
```

Full Tengo language reference: https://github.com/d5/tengo/blob/master/docs/tutorial.md

---

## Script context

Every script has two pre-set variables injected before execution:

| Variable  | Type   | Description                                     |
|-----------|--------|-------------------------------------------------|
| `topic`   | string | The IPC topic that triggered this script        |
| `payload` | map    | The decoded JSON payload from the incoming frame |

`payload` fields depend on what the client sent. Always guard against missing or wrong-typed fields:

```tengo
rpm := is_int(payload.rpm) ? payload.rpm : 0
label := is_string(payload.label) ? payload.label : "unknown"
```

---

## Standard library

All Tengo stdlib modules are available via `import`. These are the most useful ones:

```tengo
text   := import("text")    // regexp, string ops
times  := import("times")   // time parsing and formatting
math   := import("math")    // (prefer the built-in math functions instead)
rand   := import("rand")    // (prefer rand_int / rand_float instead)
json   := import("json")    // (prefer json_encode / json_decode instead)
```

Full stdlib reference: https://github.com/d5/tengo/blob/master/docs/stdlib.md

---

## Built-ins: BLE

### `notify(topic, payload)`

Send a BLE notification to the connected central. `payload` must be a map.

```tengo
notify("pong", {sequence: payload.sequence, ts: now_ms()})
notify("error", {reason: "value out of range", value: payload.rpm})
```

---

## Built-ins: Logging & time

### `log(arg, ...)`

Print values to the debug log, space-separated. Accepts any number of arguments of any type.

```tengo
log("received rpm:", payload.rpm)
log("state:", get_state("counter"), "ts:", now_ms())
```

### `now_ms() → int`

Current Unix timestamp in milliseconds.

```tengo
start := now_ms()
// ... do work ...
log("elapsed:", now_ms() - start, "ms")
```

### `time_since_ms(ts) → int`

Milliseconds elapsed since the Unix-ms timestamp `ts`. Equivalent to `now_ms() - ts`.

```tengo
last := get_state("last_seen") || 0
if time_since_ms(last) > 5000 {
    log("no data for 5+ seconds")
}
set_state("last_seen", now_ms())
```

---

## Built-ins: Formatting & JSON

### `sprintf(format, ...) → string`

Go-style string formatting. Supports `%d`, `%f`, `%s`, `%v`, `%02d`, `%.2f`, etc.

```tengo
label := sprintf("RPM: %d  Speed: %.1f km/h", payload.rpm, payload.speed)
hex   := sprintf("%08x", payload.frame_id)
```

### `json_encode(val) → string`

Serialize any Tengo value to a JSON string. Maps, arrays, strings, numbers, and booleans are all supported.

```tengo
raw := json_encode({ts: now_ms(), rpm: payload.rpm, flags: [1, 2, 3]})
// raw == '{"flags":[1,2,3],"rpm":95,"ts":1700000000000}'
```

### `json_decode(str) → any`

Parse a JSON string back into a Tengo map or array.

```tengo
data := json_decode('{"speed": 28.5, "unit": "km/h"}')
log(data.speed)   // 28.5
log(data.unit)    // "km/h"
```

---

## Built-ins: Math

### Basic arithmetic

| Function | Signature | Notes |
|---|---|---|
| `abs(x)` | `→ number` | Absolute value. Preserves int/float type. |
| `min(a, b)` | `→ number` | Smaller of two numbers |
| `max(a, b)` | `→ number` | Larger of two numbers |
| `sign(x)` | `→ int` | Returns `-1`, `0`, or `1` |
| `clamp(val, lo, hi)` | `→ number` | Clamp `val` to the range `[lo, hi]` |
| `round(x)` | `→ int` | Round to nearest integer |
| `floor(x)` | `→ int` | Round down |
| `ceil(x)` | `→ int` | Round up |

### Interpolation & remapping

| Function | Signature | Notes |
|---|---|---|
| `lerp(a, b, t)` | `→ float` | Linear interpolation. `t` is not clamped. |
| `map_range(val, in_min, in_max, out_min, out_max)` | `→ float` | Remap `val` from one range to another |

```tengo
// Map sensor 0-1023 to 0-100%
pct := map_range(payload.raw, 0, 1023, 0, 100)

// Smooth a noisy signal with exponential smoothing
prev  := get_state("smooth") || float(payload.value)
smooth := lerp(prev, float(payload.value), 0.1)
set_state("smooth", smooth)
```

### Trigonometry & advanced

| Function | Signature | Notes |
|---|---|---|
| `sqrt(x)` | `→ float` | Square root |
| `pow(base, exp)` | `→ float` | Exponentiation |
| `sin(x)` | `→ float` | Sine (radians) |
| `cos(x)` | `→ float` | Cosine (radians) |
| `tan(x)` | `→ float` | Tangent (radians) |
| `atan2(y, x)` | `→ float` | Four-quadrant arctangent |
| `hypot(a, b)` | `→ float` | `sqrt(a² + b²)` |
| `is_nan(x)` | `→ bool` | True if `x` is NaN |
| `is_inf(x)` | `→ bool` | True if `x` is ±Infinity |

### Random numbers

| Function | Signature | Notes |
|---|---|---|
| `rand_int(min, max)` | `→ int` | Random integer in `[min, max]` inclusive |
| `rand_float()` | `→ float` | Random float in `[0.0, 1.0)` |

### Constants

| Name | Value |
|---|---|
| `PI` | `3.141592653589793` |
| `E` | `2.718281828459045` |

---

## Built-ins: Arrays

All array functions return new arrays — the originals are not modified.

| Function | Signature | Description |
|---|---|---|
| `sum(arr)` | `→ number` | Sum of all elements |
| `avg(arr)` | `→ float` | Arithmetic mean |
| `min_of(arr)` | `→ number` | Smallest element |
| `max_of(arr)` | `→ number` | Largest element |
| `sort_array(arr)` | `→ array` | Sorted copy. Numeric arrays sort numerically; mixed arrays sort lexicographically. |
| `unique(arr)` | `→ array` | Remove duplicates, preserving order |
| `flatten(arr)` | `→ array` | Flatten one level of nesting |
| `zip(a, b)` | `→ array` | Pair elements: `[[a[0],b[0]], [a[1],b[1]], …]` |
| `slice_array(arr, start)` | `→ array` | Elements from `start` to end |
| `slice_array(arr, start, end)` | `→ array` | Elements from `start` to `end` (exclusive) |
| `array_contains(arr, val)` | `→ bool` | True if `val` is in `arr` |
| `reverse(arr)` | `→ array` | Reversed copy |

```tengo
// Maintain a sliding window of the last 10 readings
samples := get_state("samples") || []
samples  = append(samples, payload.value)
if len(samples) > 10 {
    samples = slice_array(samples, len(samples) - 10)
}
set_state("samples", samples)
notify("stats", {
    count: len(samples),
    avg:   avg(samples),
    min:   min_of(samples),
    max:   max_of(samples)
})
```

---

## Built-ins: Maps

| Function | Signature | Description |
|---|---|---|
| `keys(m)` | `→ array` | All keys as a string array |
| `values(m)` | `→ array` | All values as an array |
| `has_key(m, key)` | `→ bool` | True if `key` exists in `m` |
| `merge(m1, m2)` | `→ map` | Shallow merge. Keys in `m2` overwrite `m1`. |
| `pick(m, key, ...)` | `→ map` | New map containing only the specified keys |
| `omit(m, key, ...)` | `→ map` | New map with the specified keys removed |
| `map_to_pairs(m)` | `→ array` | Convert to `[[key, val], …]` |
| `pairs_to_map(arr)` | `→ map` | Convert `[[key, val], …]` back to a map |

```tengo
// Forward only safe fields
safe := omit(payload, "auth_token", "secret")
notify("forwarded", merge(safe, {ts: now_ms(), source: "script"}))

// Check for optional fields before using them
if has_key(payload, "metadata") {
    log("metadata keys:", keys(payload.metadata))
}
```

---

## Built-ins: Strings

| Function | Signature | Description |
|---|---|---|
| `split(str, sep)` | `→ array` | Split `str` on `sep` |
| `join(arr, sep)` | `→ string` | Join an array with `sep` |
| `trim(str)` | `→ string` | Remove leading and trailing whitespace |
| `to_upper(str)` | `→ string` | Uppercase |
| `to_lower(str)` | `→ string` | Lowercase |
| `contains(str, sub)` | `→ bool` | True if `str` contains `sub` |
| `starts_with(str, prefix)` | `→ bool` | True if `str` starts with `prefix` |
| `ends_with(str, suffix)` | `→ bool` | True if `str` ends with `suffix` |
| `replace(str, old, new)` | `→ string` | Replace the first occurrence of `old` with `new` |
| `replace_all(str, old, new)` | `→ string` | Replace all occurrences |
| `repeat(str, n)` | `→ string` | Concatenate `str` `n` times |
| `pad_left(str, width, pad)` | `→ string` | Left-pad `str` to `width` using `pad` character |
| `pad_right(str, width, pad)` | `→ string` | Right-pad `str` to `width` using `pad` character |

```tengo
// Zero-padded display value
display := pad_left(string(payload.rpm), 4, "0")  // "0095"

// Parse a CSV row
parts := split(payload.row, ",")
speed := float(trim(parts[2]))

// Build a structured topic name
t := join(["sensor", payload.device_id, "speed"], ".")
notify(t, {value: speed})
```

---

## Built-ins: Type predicates

Use these to guard against unexpected payload shapes before operating on values.

| Function | Returns `true` when… |
|---|---|
| `is_int(x)` | `x` is an integer |
| `is_float(x)` | `x` is a float |
| `is_string(x)` | `x` is a string |
| `is_bool(x)` | `x` is a boolean |
| `is_array(x)` | `x` is an array |
| `is_map(x)` | `x` is a map |
| `is_bytes(x)` | `x` is a byte slice |
| `is_undefined(x)` | `x` is undefined (missing field, uninitialised variable) |

```tengo
// Safely coerce payload fields
rpm   := is_int(payload.rpm)   ? payload.rpm   : 0
speed := is_float(payload.spd) ? payload.spd   : 0.0
label := is_string(payload.id) ? payload.id    : "unknown"

// Guard against missing nested fields
if !is_map(payload.meta) {
    notify("error", {reason: "missing meta field"})
    return
}
```

---

## Built-ins: Encoding

| Function | Signature | Description |
|---|---|---|
| `hex_encode(val)` | `→ string` | Encode bytes or a string to lowercase hex |
| `hex_decode(str)` | `→ bytes` | Decode a hex string to bytes |
| `base64_encode(val)` | `→ string` | Encode bytes or a string to standard base64 |
| `base64_decode(str)` | `→ bytes` | Decode a base64 string to bytes |

```tengo
// Decode a binary frame sent as hex
raw   := hex_decode(payload.frame)     // bytes
again := hex_encode(raw)               // back to hex string

// Round-trip through base64
enc := base64_encode("hello, world")
dec := string(base64_decode(enc))      // "hello, world"
```

---

## Built-ins: State

State is an in-memory key-value store shared across **all script invocations**. Values are **persisted to BadgerDB** and restored automatically on restart. Keys prefixed with `__` are reserved for internal use and are not persisted.

Because all topics share the same store, use a namespacing convention for your keys (e.g. `"cadence.count"`, `"speed.last_ts"`).

### `set_state(key, val)`

Write `val` under `key`. Accepts any Tengo value (int, float, string, bool, array, map). The value is also written to the database immediately.

### `get_state(key) → val`

Read the value stored under `key`. Returns `undefined` if the key does not exist — use `|| default` to handle this.

```tengo
count := get_state("ping.count") || 0
count += 1
set_state("ping.count", count)
```

### `del_state(key)`

Delete a key from both the in-memory store and the database.

### `throttle(key, delay_ms) → bool`

Rate-limiter. Returns `true` the first time it is called for `key`, then returns `false` until `delay_ms` milliseconds have elapsed. Uses the state store internally (`__throttle.<key>`).

```tengo
// Emit at most once every 500ms regardless of how often data arrives
if throttle("speed.notify", 500) {
    notify("speed", {value: payload.speed})
}
```

---

## Built-ins: Database

A persistent key-value store backed by [BadgerDB](https://github.com/dgraph-io/badger). Unlike `set_state`, the database is designed for **larger or structured data** that you want to query and log over time. It is organised into three namespaces:

| Namespace | Functions | Purpose |
|---|---|---|
| `kv:` | `db_get`, `db_set`, `db_del`, `db_keys` | General-purpose key-value storage |
| `log:` | `db_log`, `db_logs` | Append-only time-series event log |
| `config:` | `config_get`, `config_set`, `config_del` | Device-level string settings |

The database path defaults to `./data` and can be changed with the `-db` flag when starting the binary.

### Key-value store

#### `db_set(key, val)`

Persist any JSON-serializable value (map, array, number, string, bool). The value is stored under the `kv:` namespace — you do not include the prefix in the key.

#### `db_get(key) → val`

Retrieve a previously stored value. Returns `undefined` if the key does not exist.

#### `db_del(key)`

Delete a key from the store.

#### `db_keys([prefix]) → array`

Return all stored keys. Pass an optional prefix string to filter results. Keys are returned without the internal `kv:` prefix.

```tengo
db_set("bike.odometer", 12345.6)
db_set("bike.firmware", "v1.2.3")
db_set("ride.config", {unit: "km", display: "cadence"})

odo  := db_get("bike.odometer")   // 12345.6
cfg  := db_get("ride.config")     // {unit: "km", display: "cadence"}
miss := db_get("nonexistent")     // undefined

all_keys  := db_keys()          // ["bike.firmware", "bike.odometer", "ride.config"]
bike_keys := db_keys("bike.")   // ["bike.firmware", "bike.odometer"]
```

### Time-series log

#### `db_log(topic, data)`

Append a log entry for `topic`. Entries are stored with a nanosecond-precision timestamp key, so they are always ordered and never overwrite each other. `data` can be any JSON-serializable value.

#### `db_logs(topic[, limit]) → array`

Retrieve the most recent log entries for `topic`, newest first. `limit` defaults to `100`. Each element is the raw `data` value that was passed to `db_log`.

```tengo
// Log every incoming data point
db_log("cadence", {rpm: payload.rpm, ts: now_ms()})

// Read the last 5 cadence entries
entries := db_logs("cadence", 5)
for i, e in entries {
    log(i, "rpm:", e.rpm, "at", e.ts)
}
```

### Config

Config is stored as plain strings. It is intended for device-level settings that should survive restarts and be readable from any script.

#### `config_set(key, val)`
#### `config_get(key) → string`
#### `config_del(key)`

```tengo
// Write once (e.g. during a "setup" topic handler)
config_set("device.name", "My Bike")
config_set("device.wheel_mm", "2096")

// Read anywhere
name     := config_get("device.name")       // "My Bike"
wheel_mm := int(config_get("device.wheel_mm") || "2096")
```

---

## Built-ins: GPIO

GPIO access is available on Linux platforms with `/dev/gpiomem` (e.g. Raspberry Pi). On unsupported platforms all `gpio_*` calls return an error — scripts continue running, so guard with `throttle` or a config flag if you need to run the same script on both Pi and a dev machine.

Pin numbers are **BCM GPIO numbers** (the numbering used by the Broadcom SoC, not the physical header pin numbers).

### Pin direction

#### `gpio_input(pin)`

Configure `pin` as a digital input. Call this before `gpio_read`.

#### `gpio_output(pin)`

Configure `pin` as a digital output. Call this before `gpio_high`, `gpio_low`, or `gpio_toggle`.

### Digital output

#### `gpio_high(pin)`

Drive `pin` high (3.3 V).

#### `gpio_low(pin)`

Drive `pin` low (0 V).

#### `gpio_toggle(pin)`

Flip `pin` between high and low.

### Digital input

#### `gpio_read(pin) → int`

Read the current level of `pin`. Returns `1` if high, `0` if low.

```tengo
gpio_input(17)
level := gpio_read(17)
if level == 1 {
    log("button pressed")
}
```

### Pull resistors

| Function | Description |
|---|---|
| `gpio_pull_up(pin)` | Enable the internal pull-up resistor on `pin` |
| `gpio_pull_down(pin)` | Enable the internal pull-down resistor on `pin` |
| `gpio_pull_off(pin)` | Disable the internal pull resistor (floating) |

```tengo
// Input with pull-up — reads 0 when button connects pin to GND
gpio_input(17)
gpio_pull_up(17)
```

### PWM

The Raspberry Pi hardware PWM is available on GPIO 12, 13, 18, and 19. Software PWM is not currently supported through these builtins.

#### `gpio_pwm(pin)`

Switch `pin` into PWM mode. Must be called before `gpio_pwm_freq` and `gpio_pwm_duty`.

#### `gpio_pwm_freq(pin, freq_hz)`

Set the PWM clock frequency for `pin` in Hz. This sets the overall clock, not the period — the effective frequency seen on the pin depends on the duty cycle settings passed to `gpio_pwm_duty`.

#### `gpio_pwm_duty(pin, duty, cycle)`

Set the duty cycle. `duty` is the number of clock ticks the signal is high per `cycle` ticks total. For example, `gpio_pwm_duty(18, 64, 256)` is a 25% duty cycle.

```tengo
// 50% duty cycle on GPIO 18 at 1 kHz
gpio_pwm(18)
gpio_pwm_freq(18, 1000)
gpio_pwm_duty(18, 512, 1024)
```

### Edge detection

Edge detection lets a script poll for pin state changes without busy-waiting on `gpio_read`.

#### `gpio_detect(pin, edge)`

Enable edge detection on `pin`. `edge` must be one of:

| Value | Triggers on |
|---|---|
| `"rise"` | Low → High transition |
| `"fall"` | High → Low transition |
| `"any"` | Either transition |

#### `gpio_edge(pin) → bool`

Returns `true` if an edge has been detected on `pin` since the last call to `gpio_edge` for that pin. The flag is cleared on read.

#### `gpio_stop_detect(pin)`

Disable edge detection on `pin`.

```tengo
// Detect a button press (falling edge, button pulls pin to GND)
gpio_input(23)
gpio_pull_up(23)
gpio_detect(23, "fall")

// In a periodic script invocation:
if gpio_edge(23) {
    log("button pressed on GPIO 23")
    notify("button", {pin: 23, ts: now_ms()})
}
```

> **Note:** `gpio_edge` only checks whether an edge occurred — it does not block. Call it from a script that is invoked regularly (e.g. via a timed IPC topic) to react to hardware events.

---

## Examples

### Ping handler

```tengo
// scripts/ping.tengo
count := get_state("ping.count") || 0
count += 1
set_state("ping.count", count)

notify("pong", {
    sequence: payload.sequence,
    count:    count,
    ts:       now_ms()
})
```

### Cadence monitor with rolling average and rate limiting

```tengo
// scripts/cadence.tengo
rpm := is_int(payload.rpm) ? payload.rpm : 0

// Keep a rolling window of the last 20 samples
samples := get_state("cadence.samples") || []
samples  = append(samples, rpm)
if len(samples) > 20 {
    samples = slice_array(samples, len(samples) - 20)
}
set_state("cadence.samples", samples)

// Log every reading to the database
db_log("cadence", {rpm: rpm, ts: now_ms()})

// Notify the central at most once per 250ms
if throttle("cadence.notify", 250) {
    notify("cadence_stats", {
        rpm:    rpm,
        avg:    round(avg(samples)),
        max:    max_of(samples),
        window: len(samples)
    })
}
```

### Odometer that persists across restarts

```tengo
// scripts/speed.tengo
speed_kmh := is_float(payload.speed) ? payload.speed : float(payload.speed || 0)

// Accumulate distance: speed (km/h) * interval (s) = km
last_ts  := get_state("odo.last_ts") || now_ms()
elapsed_s := float(time_since_ms(last_ts)) / 1000.0
set_state("odo.last_ts", now_ms())

km_delta := speed_kmh * (elapsed_s / 3600.0)

// db_get/db_set for data that must survive restarts
odo := db_get("bike.odometer") || 0.0
odo += km_delta
db_set("bike.odometer", odo)

if throttle("speed.notify", 1000) {
    notify("speed", {
        speed_kmh: speed_kmh,
        odometer:  sprintf("%.2f", odo)
    })
}
```
