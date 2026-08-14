# tether

An SSH connection manager for Windows. Save your SSH connections once,
then connect or switch between them by name — tether wraps your existing
`ssh` client, it doesn't reimplement the protocol.

Two ways to use it, sharing the same saved connections
(`%APPDATA%\tether\connections.json`):

- **`tether-gui.exe`** — a desktop app: a sidebar of saved connections and
  a tabbed, embedded terminal for each open session (Termius-style).
- **`tether.exe`** — a scriptable command-line tool.

## Install

Download `tether.exe` and `tether-gui.exe` from the
[Releases](https://github.com/BeardedTech0o/tether/releases) page, or
build them yourself (see [Development](#development)).

Requires the OpenSSH client (`ssh`) to be on your `PATH` — it's bundled
with Windows 10 and later.

## GUI

Launch `tether-gui.exe`. Click **+** in the sidebar to save a connection,
then click a saved connection to open it as a tab with a live terminal.
Each tab runs its own `ssh` session; close a tab (✕) to end that session.
Open multiple tabs to switch between active sessions.

## CLI usage

```
tether add <name> --host <host> --user <user> [--port 22] [--identity path]
tether ls
tether rm <name>
tether connect <name>
tether switch
```

Running `tether` with no arguments is the same as `tether switch`.

### Save a connection

```
> tether add prod --host 203.0.113.10 --user deploy --identity ~/.ssh/id_ed25519
saved connection "prod" (deploy@203.0.113.10:22)
```

### List saved connections

```
> tether ls
NAME     TARGET                    LAST USED
prod     deploy@203.0.113.10:22   2026-08-14 21:03
staging  deploy@203.0.113.11:22   never
```

### Connect / switch

```
> tether connect prod
```

or pick interactively, most-recently-used first:

```
> tether switch
Saved connections (most recently used first):
  1) prod                 deploy@203.0.113.10:22
  2) staging               deploy@203.0.113.11:22
Select a connection (number or name), or q to quit:
```

### Delete a connection

```
> tether rm staging
deleted connection "staging"
```

## How it works

Connections are stored as JSON in your user config directory
(`%APPDATA%\tether\connections.json` on Windows). The CLI shells out to
the system `ssh` binary directly; the GUI spawns `ssh` attached to a
Windows pseudo-console (ConPTY) per tab and streams its output into an
embedded terminal (xterm.js). Either way, authentication (keys, agent,
password prompts, known hosts) works exactly as it does with plain `ssh`.
No credentials are stored by tether itself.

## Development

CLI (repo root, `github.com/BeardedTech0o/tether`):

```sh
go vet ./...
go test ./...
go build -o tether.exe .                            # native build
GOOS=windows GOARCH=amd64 go build -o tether.exe .   # cross-compile for Windows
```

GUI (`cmd/tether-gui`, a nested Go module — needs the
[Wails CLI](https://wails.io) and Node.js):

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd cmd/tether-gui
go vet ./...
wails build -platform windows/amd64   # -> build/bin/tether-gui.exe
```

The ConPTY-backed terminal only works on Windows; `go build`/`go vet`
still succeed on other platforms for development, but `OpenSession` will
return an error since there's no pseudo-console backend for that OS.
