# Tether

**A fast, no-nonsense SSH connection manager for Windows.**

Save your SSH connections once, then open, switch between, and manage
them without retyping hosts, users, or key paths. Tether wraps your
existing OpenSSH client rather than reimplementing the SSH protocol, so
authentication, keys, and known-hosts behave exactly as they do from a
plain terminal — Tether just makes getting there faster.

It ships two ways to work, sharing the same saved connections:

- **Desktop app** — a Termius-style GUI: a sidebar of saved connections
  and a tabbed, embedded terminal for every open session.
- **Command line** — a scriptable CLI for saving, listing, and connecting
  from a terminal or a script.

## Features

- **Save connections once** — name, host, port, user, and an optional
  identity file (private key).
- **Embedded terminal sessions** — open a saved connection as a tab with
  a real, interactive terminal; open several at once and switch between
  them by clicking tabs.
- **Edit and delete** connections in place, including renaming.
- **Browse for identity files** with a native file picker that opens
  straight into `~/.ssh`.
- **tmux snippets** — insert common `tmux` commands into the active
  session without submitting them, so you can adjust a session name
  before pressing Enter.
- **Copy/paste that works** — Ctrl+V, Shift+Insert, or right-click.
- **No credential storage** — Tether never touches your passwords or
  keys directly; it hands connection details to the real `ssh.exe` and
  lets your normal key/agent setup handle authentication.
- **Shared state** — the GUI and CLI read and write the same saved
  connections, so either one stays in sync with the other.

## Install

Download the latest release from the
[Releases](https://github.com/BeardedTech0o/tether/releases) page:

| File | What it is |
|---|---|
| **`Tether-Setup.exe`** | The installer. Runs a normal Windows install wizard, adds Start Menu and Desktop shortcuts, and registers an uninstaller in "Add or remove programs". Use this unless you have a reason not to. |
| **`tether-portable.exe`** | The GUI as a single portable exe — no install step, just run it. |
| **`tether-cli.exe`** | The portable command-line tool. |

Tether requires the OpenSSH client (`ssh`) to be on your `PATH`, which is
bundled with Windows 10 and later by default.

## Using the GUI

Launch Tether (from the Start Menu if you used the installer, or by
running `tether-portable.exe`).

1. Click **+** in the sidebar to save a connection: name, host, user,
   port, and optionally an identity file (use **Browse…** to pick a key
   from `~/.ssh`).
2. Click a saved connection to open it as a tab with a live terminal.
3. Open more connections to add more tabs — click between tabs to
   switch which session you're looking at.
4. Close a tab (**✕**) to end that session.
5. Hover a saved connection to reveal **✎ Edit** (change any field, or
   rename it) and **✕ Delete**.

**tmux snippets** — the panel at the bottom of the sidebar inserts a
common `tmux` command into the active terminal without submitting it, so
you can edit it (e.g. swap out `mysession` for a real name) before
pressing Enter.

**Paste** into a terminal with **Ctrl+V**, **Shift+Insert**, or
right-click.

## Using the CLI

```
tether-cli add <name> --host <host> --user <user> [--port 22] [--identity path]
tether-cli ls
tether-cli connect <name>
tether-cli switch
tether-cli rm <name>
```

Running `tether-cli` with no arguments is the same as `tether-cli switch`.

**Save a connection:**
```
> tether-cli add prod --host 203.0.113.10 --user deploy --identity ~/.ssh/id_ed25519
saved connection "prod" (deploy@203.0.113.10:22)
```

**List saved connections:**
```
> tether-cli ls
NAME     TARGET                    LAST USED
prod     deploy@203.0.113.10:22   2026-08-14 21:03
staging  deploy@203.0.113.11:22   never
```

**Connect directly, or pick interactively** (most-recently-used first):
```
> tether-cli connect prod

> tether-cli switch
Saved connections (most recently used first):
  1) prod                 deploy@203.0.113.10:22
  2) staging               deploy@203.0.113.11:22
Select a connection (number or name), or q to quit:
```

**Delete a connection:**
```
> tether-cli rm staging
deleted connection "staging"
```

## SSH keys and passwordless login

Tether stores **no credentials** — only a connection's host, port, user,
and (optionally) a path to a private key file. Whether a session needs a
password comes down to your normal OpenSSH setup, exactly as if you ran
`ssh` yourself:

- **Identity file set on the connection** — Tether passes it to `ssh` as
  `-i <path>`, so that key is used for that connection.
- **Identity file left blank** — `ssh` falls back to its own defaults:
  keys already loaded in `ssh-agent`/Pageant, then the standard files in
  `~/.ssh` (`id_ed25519`, `id_rsa`, etc.).

Either way, if the matching public key is authorized on the remote host,
you'll connect without a password prompt. If the key has a passphrase
and isn't already loaded in an agent, `ssh` will prompt for it once per
session — in the GUI this shows up right in the embedded terminal, since
it's a real interactive `ssh` process underneath.

Connections are stored as JSON in `%APPDATA%\tether\connections.json`,
shared by both the GUI and CLI.

---

## How it's built

Tether is two Go programs sharing a common core (`internal/store` for
saved connections, `internal/sshexec` for building `ssh` arguments):

- **CLI** (repo root) — a small stdlib-only Go program that execs the
  system `ssh` binary directly, with the terminal's own stdio.
- **GUI** (`cmd/tether-gui`) — a [Wails](https://wails.io) app: a Go
  backend and an HTML/CSS/JS (xterm.js) frontend. Each open tab spawns
  `ssh` attached to a Windows pseudo-console (ConPTY) and streams its
  output into an embedded terminal.

Building from source:

```sh
# CLI
go vet ./... && go test ./...
GOOS=windows GOARCH=amd64 go build -o tether-cli.exe .

# GUI (needs the Wails CLI, Node.js, and NSIS for the installer)
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd cmd/tether-gui
wails build -platform windows/amd64         # -> build/bin/tether-portable.exe
wails build -platform windows/amd64 -nsis   # also builds the installer
```

The ConPTY-backed terminal only works on Windows; `go build`/`go vet`
still succeed on other platforms for development, but opening a GUI
session will return an error since there's no pseudo-console backend for
that OS.
