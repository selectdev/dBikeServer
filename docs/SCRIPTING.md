# dBike Script Engine

_For general architecture and package documentation see [README.md](README.md)._  

Scripts are [Tengo](https://github.com/d5/tengo) files placed in this directory. Each file is compiled once at startup and executed on demand. The filename determines which IPC topic it handles — `ping.tengo` runs whenever a frame arrives with `topic: "ping"`. Scripts run synchronously within the BLE handler goroutine. Keep them fast. Use `throttle()` to drop excess invocations.

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
- [Built-ins: Signal processing](#built-ins-signal-processing)
- [Built-ins: Database](#built-ins-database)
- [Built-ins: GPIO](#built-ins-gpio)
- [Built-ins: OpenAI](#built-ins-openai)
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

x := 10
name := "hello"


x = 20


m := {a: 1, b: 2}   
m := {a: 1, b: 2,}  


val := get_state("key") || 0   


s := string(42)      
n := int("42")       
f := float("3.14")   


for i, v in [1, 2, 3] {
    log(i, v)
}
for k, v in {a: 1, b: 2} {
    log(k, v)
}


double := func(x) { return x * 2 }
log(double(5))  
```

Full Tengo language reference: https:

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
text   := import("text")    
times  := import("times")   
math   := import("math")    
rand   := import("rand")    
json   := import("json")    
```

Full stdlib reference: https:

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

### `format_duration(ms) → string`

Format a millisecond duration as a compact human-readable string. Hours are omitted when zero; leading zeros are added to minutes and seconds when a higher unit is present.

| Input | Output |
|---|---|
| `45000` | `"45s"` |
| `90000` | `"1m30s"` |
| `3723000` | `"1h02m03s"` |

```tengo
elapsed := time_since_ms(get_state("ride.start") || now_ms())
notify("ride_time", {display: format_duration(elapsed)})
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

```

### `json_decode(str) → any`

Parse a JSON string back into a Tengo map or array.

```tengo
data := json_decode('{"speed": 28.5, "unit": "km/h"}')
log(data.speed)   
log(data.unit)    
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

pct := map_range(payload.raw, 0, 1023, 0, 100)


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

### Sensor helpers

| Function | Signature | Description |
|---|---|---|
| `dead_band(val, threshold)` | `→ float` | Returns `0.0` if `abs(val) < threshold`, otherwise returns `val`. Filters noise around zero. |
| `haversine(lat1, lon1, lat2, lon2)` | `→ float` | Great-circle distance in **metres** between two GPS coordinates (decimal degrees). |

```tengo

clean_rpm := dead_band(float(payload.rpm), 3.0)


dist_m := haversine(51.5074, -0.1278, 51.5080, -0.1270)
```

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

safe := omit(payload, "auth_token", "secret")
notify("forwarded", merge(safe, {ts: now_ms(), source: "script"}))


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

display := pad_left(string(payload.rpm), 4, "0")  


parts := split(payload.row, ",")
speed := float(trim(parts[2]))


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

rpm   := is_int(payload.rpm)   ? payload.rpm   : 0
speed := is_float(payload.spd) ? payload.spd   : 0.0
label := is_string(payload.id) ? payload.id    : "unknown"


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

raw   := hex_decode(payload.frame)     
again := hex_encode(raw)               


enc := base64_encode("hello, world")
dec := string(base64_decode(enc))      
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

if throttle("speed.notify", 500) {
    notify("speed", {value: payload.speed})
}
```

### `debounce(key, delay_ms) → bool`

Returns `true` only when the gap between calls is **at least** `delay_ms` milliseconds — i.e. the signal has been quiet. Returns `false` on the very first call and whenever calls arrive faster than `delay_ms`. Uses the state store internally (`__debounce.<key>`).

This is the complement of `throttle`: throttle fires on the leading edge, debounce fires on the trailing edge (after silence).

```tengo

gpio_input(23)
gpio_detect(23, "fall")
if gpio_edge(23) {
    if debounce("btn.23", 50) {
        notify("button", {pin: 23})
    }
}
```

---

## Built-ins: Signal processing

These functions are stateful — they store internal state under a namespaced key so each sensor or control loop gets its own history. Keys are prefixed with `__` and are not persisted to the database.

### `ewma(key, val, alpha) → float`

Exponential weighted moving average. On the first call the raw `val` is returned as-is; subsequent calls blend the new sample with the running average using `alpha * val + (1 - alpha) * prev`. A smaller `alpha` gives a smoother but slower-responding signal.

| `alpha` | Character |
|---|---|
| `0.05` | Very smooth — slow to respond |
| `0.1` | Smooth — good for cadence / speed |
| `0.3` | Moderate |
| `0.5` | Fast — follows signal closely |

```tengo
smooth_rpm := ewma("cadence.smooth", float(payload.rpm), 0.1)
notify("cadence", {rpm: round(smooth_rpm)})
```

### `pid_update(key, setpoint, measured, kp, ki, kd) → float`

Discrete PID controller. Returns the control output for one time step. `key` is used to store the integrator and previous error between calls — use a unique key per control loop. Gains (`kp`, `ki`, `kd`) must be tuned for your plant and call rate.

```tengo

target  := 25.0
current := is_float(payload.speed) ? payload.speed : 0.0
duty    := pid_update("speed.pid", target, current, 0.8, 0.05, 0.1)
duty     = clamp(duty, 0, 100)
gpio_pwm_duty(18, int(duty), 100)
```

---

## Built-ins: Database

A persistent key-value store backed by [BadgerDB](https:

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

odo  := db_get("bike.odometer")   
cfg  := db_get("ride.config")     
miss := db_get("nonexistent")     

all_keys  := db_keys()          
bike_keys := db_keys("bike.")   
```

### Time-series log

#### `db_log(topic, data)`

Append a log entry for `topic`. Entries are stored with a nanosecond-precision timestamp key, so they are always ordered and never overwrite each other. `data` can be any JSON-serializable value.

#### `db_logs(topic[, limit]) → array`

Retrieve the most recent log entries for `topic`, newest first. `limit` defaults to `100`. Each element is the raw `data` value that was passed to `db_log`.

```tengo

db_log("cadence", {rpm: payload.rpm, ts: now_ms()})


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

config_set("device.name", "My Bike")
config_set("device.wheel_mm", "2096")


name     := config_get("device.name")       
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

gpio_input(23)
gpio_pull_up(23)
gpio_detect(23, "fall")


if gpio_edge(23) {
    log("button pressed on GPIO 23")
    notify("button", {pin: 23, ts: now_ms()})
}
```

> **Note:** `gpio_edge` only checks whether an edge occurred — it does not block. Call it from a script that is invoked regularly (e.g. via a timed IPC topic) to react to hardware events.

---

## Built-ins: OpenAI

These functions call the [OpenAI Chat Completions API](https:

**Prerequisite:** set the `OPENAI_API_KEY` environment variable before starting the server. The SDK picks it up automatically.

```sh
export OPENAI_API_KEY=sk-...
./dbikeserver
```

Each message in the `messages` array is a map with two required string keys:

| Key | Values |
|---|---|
| `role` | `"system"`, `"user"`, or `"assistant"` |
| `content` | The message text |

### `openai_chat(model, messages) → string`

Send a chat completion request and return the reply text. Errors are propagated as script errors and logged.

```tengo
reply := openai_chat("gpt-4o-mini", [
    {role: "system", content: "You are a cycling coach. Be concise."},
    {role: "user",   content: "My cadence is " + string(payload.rpm) + " RPM. Is that good?"}
])
notify("coach", {advice: reply})
```

### `openai_chat_ex(model, messages) → map`

Like `openai_chat` but returns a map with the full response, including token usage:

| Key | Type | Description |
|---|---|---|
| `content` | string | Reply text |
| `finish_reason` | string | Why generation stopped (`"stop"`, `"length"`, etc.) |
| `prompt_tokens` | int | Tokens consumed by the prompt |
| `completion_tokens` | int | Tokens in the reply |
| `total_tokens` | int | Sum of prompt + completion tokens |

```tengo
res := openai_chat_ex("gpt-4o", [
    {role: "user", content: "Summarise this ride: " + json_encode(payload)}
])
log("tokens used:", res.total_tokens)
db_log("ai.summary", {text: res.content, tokens: res.total_tokens, ts: now_ms()})
notify("summary", {text: res.content})
```

### Example: AI-powered anomaly detection

```tengo

rpm   := is_int(payload.rpm)   ? payload.rpm   : 0
speed := is_float(payload.spd) ? payload.spd   : 0.0


if rpm > 150 || speed > 60.0 {
    context := sprintf("RPM: %d, Speed: %.1f km/h", rpm, speed)
    reply := openai_chat("gpt-4o-mini", [
        {role: "system", content: "You are a bike sensor analyst. Reply in one sentence."},
        {role: "user",   content: "Is this reading abnormal? " + context}
    ])
    notify("anomaly_alert", {reading: context, analysis: reply})
}
```

---

## Examples

### Ping handler

```tengo

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

rpm := is_int(payload.rpm) ? payload.rpm : 0


samples := get_state("cadence.samples") || []
samples  = append(samples, rpm)
if len(samples) > 20 {
    samples = slice_array(samples, len(samples) - 20)
}
set_state("cadence.samples", samples)


db_log("cadence", {rpm: rpm, ts: now_ms()})


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

speed_kmh := is_float(payload.speed) ? payload.speed : float(payload.speed || 0)


last_ts  := get_state("odo.last_ts") || now_ms()
elapsed_s := float(time_since_ms(last_ts)) / 1000.0
set_state("odo.last_ts", now_ms())

km_delta := speed_kmh * (elapsed_s / 3600.0)


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
