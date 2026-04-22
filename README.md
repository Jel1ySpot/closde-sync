# closde-sync

`closde-sync` is a Go + Node preload wrapper around Claude Code with optional Xray-based proxying and a lightweight sync server.

## What it does

- Builds a cross-platform `closde` binary into `build/closde` or `build/closde.exe`.
- Runs in **client mode** by default and forwards all received flags to the downstream Node-based Claude CLI.
- Runs in **server mode** when the executable filename contains `server`.
- Injects a preload bundle into the Claude CLI process.
- Optionally starts an in-process Xray proxy from `CLOSDE_PROXY` instead of using an external Xray binary.
- Supports `DEBUG_MODE` for Go, preload, and Xray debug logging.

## Project layout

```text
.
├── build/            # built binaries
├── dist/             # built preload bundle
├── internal/         # client/runtime/build/xray internals
├── preload/          # preload TypeScript source
├── server/           # sync server implementation
├── main.go           # root binary entry
└── package.json      # JS build scripts
```

## Requirements

- Node.js
- npm
- Go (Development)

The client also expects a Claude Code installation managed under `~/.closde/claude/<version>` and requires `CLOSDE_CLAUDE_VERSION` to be set in `~/.closde/.env` or the process environment.

## Build

Install dependencies:

```bash
npm install
```

Build everything:

```bash
npm run build
```

This does two things:

- bundles `preload/main.ts` to `dist/preload.js`
- builds the Go binary to `build/closde` or `build/closde.exe`

Useful scripts:

```bash
npm run build         # build preload + binary
npm run build:preload # build dist/preload.js
npm run build:cmd     # build build/closde(.exe)
npm run check         # TypeScript check + go test ./...
```

## Client

Example:

```bash
./closde --help
./closde -p "hello"
```

### Client environment

`closde` loads `~/.closde/.env` automatically. If the file does not exist, it is created and the first run exits so you can fill in the required values.

```env
CLOSDE_CLAUDE_VERSION=<installed-claude-version>
```

### Preload resolution

At startup the client resolves the preload in this order:

1. `CLOSDE_PRELOAD_FILE`
2. `~/.closde/preload.js`

## Server

Rename the binary like `xxx-server` to enable server mode.

Examples:

```bash
cp ./build/closde ./build/closde-server
./build/closde-server --help
./build/closde-server serve --addr :8080
```

Server commands:

```bash
./build/closde-server serve
./build/closde-server create-token [name]
./build/closde-server get-config
```

### Server flags

`serve` supports:

```bash
--addr                 HTTP listen address
--config-file          server config file path
--watch-local-state    watch Claude local state files and broadcast updates
--claude-settings      Claude settings file path
--claude-credentials   Claude credentials file path
```

Defaults:

- listen address: `:8080`
- config file: `<repo>/config.json`
- Claude settings: `~/.claude.json`
- Claude credentials: `~/.claude/.credentials.json`

## Proxy / Xray

If `CLOSDE_PROXY` is set, the client starts an embedded Xray instance and points Node traffic to the local HTTP proxy.

Supported proxy URI styles are implemented under `internal/xray/`.

## Logging and `DEBUG_MODE`

Logging is controlled by the environment variable `DEBUG_MODE`.

Debug mode is enabled only when `DEBUG_MODE` is one of:

- `1`
- `true`
- `yes`
- `on`
- `debug`

Anything else, including `0`, an empty string, or an unset variable, is treated as non-debug mode.

Examples:

```bash
DEBUG_MODE=1 ./build/closde
DEBUG_MODE=0 ./build/closde
```

Behavior:

- **default / non-debug**
  - Go logger runs at `info`
  - preload keeps only important `info` logs
  - Xray log level is `none`
- **debug mode**
  - Go logger runs at `debug`
  - preload debug logs are enabled
  - Xray log level is `debug`

## Development notes

- The root binary entrypoint is `main.go`.
- Client orchestration lives in `internal/cli/`.
- Runtime environment and preload handling live in `internal/runtime/`.
- Embedded Xray config builders live in `internal/xray/`.
- Server implementation lives in `server/`.

## Verification

Run the standard validation steps:

```bash
npm run check
npm run build
```
