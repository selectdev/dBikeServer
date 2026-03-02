# dBikeServer Documentation

This directory holds reference material for the **dBikeServer** codebase.  The goal is to give newcomers an overview of each component, explain how they fit together, and collect hands‑on usage notes.  The centerpiece is this file; supplementary documents (like the detailed scripting guide) live alongside.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Configuration](#configuration)
4. [Packages & Directories](#packages--directories)
5. [Building and Running](#building-and-running)
6. [BLE Peripheral](#ble-peripheral)
7. [IPC Protocol](#ipc-protocol)
8. [Script Engine](#script-engine)
9. [Database](#database)
10. [GPIO Support](#gpio-support)
11. [Panel & Launcher](#panel--launcher)
12. [Utilities](#utilities)
13. [Scripting Reference](#scripting-reference)
14. [Extending the Codebase](#extending-the-codebase)

---

## Project Overview

dBikeServer is a lightweight Go-based BLE peripheral that exposes a simple IPC (inter‑process communication) protocol over Bluetooth Low Energy.  It is designed to run on Raspberry Pi‑class hardware and is accompanied by a terminal‑based “panel” and a macOS launcher for easy control.  The core features include:

- BLE service with bidirectional messaging
- Embedded scripting using Tengo for user‑defined handlers
- Persistent state stored in BadgerDB
- Optional GPIO access when running on hardware that provides it
- CLI panel for monitoring and controlling the service
- macOS launcher that opens the panel in Terminal/iTerm2

The majority of the code is organized into small, focused packages; high‑level logic lives under the `main` package and `handler.go`.


## Architecture

At startup the `main` package initializes the following subsystems in roughly this order:

1. **Database** – opens a Badger instance (`db` package) and starts a garbage‑collection goroutine.
2. **GPIO** – attempts to open `/dev/mem` via the `go-rpio` library; falls back gracefully on non‑Pi systems.
3. **Notify & Write characteristics** – created by the `ble` package; the former is used to send notifications to a connected central, the latter receives framed JSON packets.
4. **Script Engine** – the `script` package scans `scripts/` for `.tengo` files, compiles them, and exposes a handler that executes them on demand.  Built‑in helpers provide access to BLE, DB, GPIO, OpenAI, etc.
5. **BLE Manager** – a loop in `ble/manager.go` advertises the custom service and recovers from errors.
6. **Frame Handling** – incoming frames are parsed in `handler.go`; they are acknowledged and dispatched either to a script or to the hard‑coded switch (for topics like `ping`).

The panel and launcher are independent binaries that interact with the running service by inspecting process state and executing the server binary; they live in separate `main` packages under `panel` and `launcher`.

Communication between components is intentionally kept asynchronous and minimal; most of the complexity lives in the scripting layer and the BLE framing code.


## Configuration

All tunable parameters are defined in `config/constants.go`.  Values are either constants or read from the environment via `envOr`.

| Variable           | Environment Variable    | Description                            |
|--------------------|-------------------------|----------------------------------------|
| `DeviceName`       | `DBIKE_BLE_NAME`        | BLE advertising name                   |
| `ServiceUUID`      | –                       | 128‑bit UUID of the BLE service        |
| `WriteCharUUID`    | –                       | UUID of the write characteristic       |
| `NotifyCharUUID`   | –                       | UUID of the notify characteristic      |
| `ScriptsDir`       | `DBIKE_SCRIPTS_DIR`     | Directory where `.tengo` scripts live  |
| `DBPath`           | `DBIKE_DB_PATH`         | Path to Badger data directory          |
| `OpenAIAPIKey`     | `OPENAI_API_KEY`        | Key used by the OpenAI scripting helper|

Other constants control buffer sizes, advertising retry delays, etc.  See `config/constants.go` for details.


## Packages & Directories

The repository is arranged as follows:

```
cmd files (main.go, handler.go)     # entrypoints for server
ble/          # BLE service & framing logic
config/       # constants and environment helpers
db/           # thin BadgerDB wrapper with common helpers
gpio/         # wrapper around go-rpio for GPIO access
ipc/          # definitions for IPC packets/frames
script/       # script engine built on Tengo
    builtins/  # helpers exposed to scripts
panel/        # interactive terminal UI (bubbletea)
launcher/     # macOS helper for starting the panel
util/         # logging helpers
```

The `scripts/` directory lives at the project root and is meant to be populated by the user; sample scripts (e.g. `ping.tengo`) may be shipped alongside the binary.


## Building and Running

To build the server:

```sh
go build -o dbikeserver ./...
```

By default the BLE peripheral will advertise using the current working directory’s `config.DBPath` (`./data`).  Use `-db` flag or `DBIKE_DB_PATH` to override.

The panel binary is built with:

```sh
go build -o dbikeserver-panel ./panel
```

and the macOS launcher with:

```sh
go build -o dbikeserver-launcher ./launcher
```

A helper script `install.sh` wraps these steps and can also invoke `go get` or `go install` as appropriate; run `./install.sh build` for a quick start.


## BLE Peripheral

All BLE‑related code lives under the `ble` package.  `manager.go` is responsible for advertising the custom service and keeping it alive.  The service contains two characteristics:

* **Write characteristic:** Accepts writes (no response) from a central.  Data is buffered and split on newline characters by the `LineFramer` type.  Each line is interpreted as a JSON object and dispatched as an `ipc.Packet`.
* **Notify characteristic:** Clients may subscribe to receive notifications.  The `NotifyCharacteristic` maintains an internal buffered channel; when `Notify(topic,payload)` is called the payload is encoded as a JSON packet and queued.  A helper goroutine drains the channel and writes notifications at a controlled interval.

The BLE UUIDs are defined in `config/constants.go` and are part of the protocol.  The advertising loop in `RunBLEManager` recovers from transient failures and logs status via `util.Log`.


## IPC Protocol

IPC messages are simple JSON packets framed with newline separators.  The `ipc` package defines two types:

```go
// Packet is the JSON payload sent between client and peripheral.
type Packet struct {
    ID      string         `json:"id"`
    Topic   string         `json:"topic"`
    SentAt  string         `json:"sentAt"`
    Payload map[string]any `json:"payload"`
}

// Frame records metadata about a received line, including parse errors.
type Frame struct {
    Raw    string
    Bytes  int
    Packet *Packet
    Err    error
}
```

`handler.go` is the central dispatcher; it logs the incoming topic, sends an acknowledgement notification, and then either invokes a corresponding script or handles built‑in topics directly.


## Script Engine

The scripting subsystem is one of the most powerful features of dBikeServer.  It uses the [Tengo](https://github.com/d5/tengo) language to allow users to write dynamic handlers for arbitrary IPC topics.

Detailed information about the scripting language, built‑in functions, and examples lives in [SCRIPTING.md](./SCRIPTING.md).  In brief:

- Scripts are stored as `*.tengo` files in the configured scripts directory.
- Each file’s name (minus extension) corresponds to the topic it handles.
- Scripts execute synchronously on the BLE I/O goroutine; keep them short to avoid blocking the connection.
- A shared state map persists between invocations and is backed to the database.
- Built‑in helper functions expose logging, time helpers, GPIO control, database access, OpenAI integration, and more.

The `script.Engine` type compiles and caches scripts, provides a `HandleEvent` method to execute them, and manages the persistent state.  New engines can be constructed with `NewEngine` by passing references to the notify characteristic, database instance, GPIO controller, and script directory.


## Database

Persistence is provided by BadgerDB, wrapped by the `db` package to simplify common operations.  The `DB` type exposes:

```go
Get(key string) ([]byte, bool, error)
Set(key string, val []byte) error
Delete(key string) error
Scan(prefix string) ([][2][]byte, error)          // forward iteration
ScanKeys(prefix string) ([]string, error)
ScanReverse(prefix string, limit int) ([][2][]byte, error)
```

A background goroutine periodically runs value log garbage collection.  Scripts may interact with the database through built‑in helpers (`get_db`, `set_db`, `scan_db`, etc.).


## GPIO Support

GPIO handling is a thin wrapper around `github.com/stianeikeland/go-rpio/v4`.  The `gpio.GPIO` type provides methods for common operations (input/output, high/low, PWM, edge detection).  If the board does not support `/dev/mem`, `Open()` returns an error and the server continues without GPIO functionality.

Scripts access GPIO via helpers like `gpio_output(pin)`, `gpio_read(pin)`, and `gpio_detect(pin, edge)`.


## Panel & Launcher

The `panel` subdirectory contains a Bubble Tea–based command‑line user interface.  It shows system statistics, the status of the server process, and lets the user start/stop the service or trigger helper scripts (`install.sh build`, `install.sh upgrade`).  Running `dbikeserver-panel` without arguments displays the UI; the `--watch` flag enables automatic refreshing.

`launcher/main.go` is a macOS‑specific helper that finds the panel binary next to itself, ensures the server is running, and then opens a new Terminal or iTerm2 window/tab with the panel in fullscreen.


## Utilities

The `util` package currently only contains logging helpers.  `util.Log` and `util.Logf` print timestamped messages to STDOUT and optionally forward them to a debug writer when one is configured.  Centralizing logging makes it easy to redirect output or add additional sinks in the future.


## Scripting Reference

See [SCRIPTING.md](./SCRIPTING.md) for the full scripting guide.  It covers language basics, built‑ins, examples, and implementation notes.  This document is deliberately long – it is intended to be distributed with the binary and read by script authors.


## Extending the Codebase

When adding new features, please consider the following:

1. Keep packages small and focused.  Shared functionality belongs in the `util` package or a new helper package.
2. Update documentation in this directory whenever public APIs change.
3. For new scripting helpers add functions to the appropriate file under `script/builtins` and update the scripting doc with examples.
4. Run `go vet`, `go fmt`, and `go test` (where applicable) before committing.

Happy hacking!
