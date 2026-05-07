# closde-sync

`closde-sync` is a Go + Node preload wrapper around Claude Code with optional Xray-based proxying and a lightweight sync server.

## What it does

- Builds a cross-platform `closde` binary into `build/closde` or `build/closde.exe`.
- Runs in **client mode** by default and forwards all received flags to the downstream Node-based Claude CLI.
- Runs in **server mode** when the executable filename contains `server`.
- Injects a preload bundle into the Claude CLI process.
- Optionally starts an in-process Xray proxy from `CLOSDE_PROXY` instead of using an external Xray binary.

## Usage

1. Download the `closde` binary and `preload.js` for your platform from [Release](https://github.com/Jel1ySpot/closde-sync/releases/latest).
2. Create `.closed` folder in home directory.
3. Move `preload.js` into `.closde`.
4. Create `.env` file in `.closde`, edit it according to [this instructions](#client-environment) in the README.
5. Move the binary to directory like `~/.loacl/bin` or `/usr/local/bin`, and use it.

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

The client also expects a Claude Code installation managed under `~/.closde/claude/<version>`. If `CLOSDE_CLAUDE_VERSION` is unset, closde automatically resolves and downloads the latest Claude Code version from npm.

## Build

Install dependencies:

```bash
npm install
```

Build everything:

```bash
npm run build
```

Useful scripts:

```bash
npm run build         # build preload + binary
npm run build:preload # build dist/preload.js
npm run build:cmd     # build build/closde(.exe)
npm run check         # TypeScript check + go test ./...
```

## Client

### Client environment

`closde` loads `~/.closde/.env` automatically. If the file does not exist, it is created and the first run exits so you can fill in the required values.

```env
CLOSDE_CLAUDE_VERSION=<installed-claude-version> # optional: set only when pinning
```

Support environment flags:

| Name | Required | Default | Describe |
| --- | --- | --- | --- |
| `CLOSDE_CLAUDE_VERSION` | N | / | Claude Code version to use. If omitted, the latest npm release is used. |
| `CLOSDE_PROXY` | N | / | The proxy URI provided to Xray to drive claude. |
| `CLOSDE_SERVER_URL` | N | / | Closde server URL. |
| `CLOSDE_AUTH_TOKEN` | N | / | The token used to connect to the Cloude server. |
| `HTTPS_PROXY` | N | / | The HTTPS proxy URL provided to claude. |
| `HTTP_PROXY` | N | / | The HTTP proxy URL provided to claude. |
| `CLOSDE_PLATFORM` | N | / | Platform name injected into `process.platform` |
| `CLOSDE_ARCH` | N | / | Arch information injected into `process.arch` |
| `CLOSDE_NODE_VERSION` | N | / | NodeJS version injected into `process.version`. |
| `CLOSDE_CLAUDE_SETTINGS` | N | `$HOME/.claude.json` | `.claude.json` file path. |
| `CLOSDE_CLAUDE_CREDENTIALS` | N | `$HOME/.claude/.credentials.json` | `.credentials.json` file path. |

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

## Verification

Run the standard validation steps:

```bash
npm run check
npm run build
```
