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

Download from the [Releases](https://github.com/BeardedTech0o/tether/releases)
page, or build them yourself (see [Development](#development)):

- **`tether-gui-amd64-installer.exe`** — the installable GUI: runs a normal
  Windows install wizard, adds Start Menu and Desktop shortcuts, and
  registers an uninstaller in "Add or remove programs".
- **`tether-gui.exe`** — the same GUI as a portable exe, no install step.
- **`tether.exe`** — the portable CLI.

Requires the OpenSSH client (`ssh`) to be on your `PATH` — it's bundled
with Windows 10 and later.

## GUI

Launch `tether-gui.exe`. Click **+** in the sidebar to save a connection,
then click a saved connection to open it as a tab with a live terminal.
Each tab runs its own `ssh` session; close a tab (✕) to end that session.
Open multiple tabs to switch between active sessions. Hover a saved
connection to reveal **✎ Edit** (change host/user/port/identity file, or
rename it) and **✕ Delete**.

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
embedded terminal (xterm.js). Either way, tether is just running the real
`ssh.exe` for you — it never handles your keys or passwords itself.

### SSH keys and passwordless login

tether stores **no credentials** — only the connection's host, port,
user, and (optionally) a path to a private key file. Whether a session
needs a password comes down to your normal OpenSSH setup, exactly as if
you ran `ssh` yourself from a terminal:

- **Identity file set on the connection** (the "Identity file" field in
  the GUI, or `--identity` on the CLI) — tether passes it to `ssh` as
  `-i <path>`, so that key is used for that connection.
- **Identity file left blank** — `ssh` falls back to its own defaults:
  keys already loaded in `ssh-agent`/Pageant, then the standard files in
  `~/.ssh` (`id_ed25519`, `id_rsa`, etc.).
- Either way, if the matching public key is authorized on the remote
  host, you'll connect without a password prompt. If the key has a
  passphrase and isn't already loaded in an agent, `ssh` will prompt for
  the passphrase once per session — in the GUI this shows up right in
  the embedded terminal, since it's a real interactive `ssh` process.

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
wails build -platform windows/amd64         # -> build/bin/tether-gui.exe
wails build -platform windows/amd64 -nsis   # also -> build/bin/tether-gui-amd64-installer.exe (needs NSIS's makensis on PATH)
```

The ConPTY-backed terminal only works on Windows; `go build`/`go vet`
still succeed on other platforms for development, but `OpenSession` will
return an error since there's no pseudo-console backend for that OS.
