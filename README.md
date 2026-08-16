# tether

An SSH connection manager for Windows. Save your SSH connections once,
then connect or switch between them by name — tether wraps your existing
`ssh` client, it doesn't reimplement the protocol.

Two ways to use it, sharing the same saved connections
(`%APPDATA%\tether\connections.json`):

- **GUI** — a desktop app: a sidebar of saved connections and a tabbed,
  embedded terminal for each open session (Termius-style).
- **CLI** (`tether-cli.exe`) — a scriptable command-line tool.

## Install

Download from the [Releases](https://github.com/BeardedTech0o/tether/releases)
page, or build them yourself (see [Development](#development)):

- **`Tether-Setup.exe`** — the installer: runs a normal Windows install
  wizard, adds Start Menu and Desktop shortcuts, and registers an
  uninstaller in "Add or remove programs". Use this unless you have a
  reason not to.
- **`tether-portable.exe`** — the same GUI as a portable exe, no install
  step (just run it from wherever you put it).
- **`tether-cli.exe`** — the portable CLI.

Requires the OpenSSH client (`ssh`) to be on your `PATH` — it's bundled
with Windows 10 and later.

## GUI

Launch Tether (from the Start Menu if you used the installer, or by
running `tether-portable.exe`). Click **+** in the sidebar to save a connection,
then click a saved connection to open it as a tab with a live terminal.
Each tab runs its own `ssh` session; close a tab (✕) to end that session.
Open multiple tabs to switch between active sessions. Hover a saved
connection to reveal **✎ Edit** (change host/user/port/identity file, or
rename it) and **✕ Delete**.

The **tmux snippets** panel at the bottom of the sidebar inserts a
common `tmux` command into the active session's terminal without
submitting it — you can edit it (e.g. swap out `mysession`) and press
Enter yourself.

## CLI usage

```
tether-cli add <name> --host <host> --user <user> [--port 22] [--identity path]
tether-cli ls
tether-cli rm <name>
tether-cli connect <name>
tether-cli switch
```

Running `tether-cli` with no arguments is the same as `tether-cli switch`.

### Save a connection

```
> tether-cli add prod --host 203.0.113.10 --user deploy --identity ~/.ssh/id_ed25519
saved connection "prod" (deploy@203.0.113.10:22)
```

### List saved connections

```
> tether-cli ls
NAME     TARGET                    LAST USED
prod     deploy@203.0.113.10:22   2026-08-14 21:03
staging  deploy@203.0.113.11:22   never
```

### Connect / switch

```
> tether-cli connect prod
```

or pick interactively, most-recently-used first:

```
> tether-cli switch
Saved connections (most recently used first):
  1) prod                 deploy@203.0.113.10:22
  2) staging               deploy@203.0.113.11:22
Select a connection (number or name), or q to quit:
```

### Delete a connection

```
> tether-cli rm staging
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
go build -o tether-cli.exe .                            # native build
GOOS=windows GOARCH=amd64 go build -o tether-cli.exe .   # cross-compile for Windows
```

GUI (`cmd/tether-gui`, a nested Go module — needs the
[Wails CLI](https://wails.io) and Node.js):

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd cmd/tether-gui
go vet ./...
wails build -platform windows/amd64         # -> build/bin/tether-portable.exe
wails build -platform windows/amd64 -nsis   # also -> build/bin/tether-gui-amd64-installer.exe (renamed to Tether-Setup.exe in CI; needs NSIS's makensis on PATH)
```

The ConPTY-backed terminal only works on Windows; `go build`/`go vet`
still succeed on other platforms for development, but `OpenSession` will
return an error since there's no pseudo-console backend for that OS.
