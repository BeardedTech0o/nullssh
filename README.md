![nullssh banner](https://raw.githubusercontent.com/BeardedTech0o/tether/main/internal/nullssh-banner.png)

# nullssh

An SSH connection manager for Windows with custom snippets.

## The Problem

Every SSH session starts the same way: dig up the right host, remember
whether this one uses a key or a password, retype the user and port,
then re-run the same three commands once you're in (attach the tmux
session, cd into the deploy directory, tail the right log). None of
that is hard, it's just friction that repeats every single time you
connect to something.

nullssh exists to remove that friction. Save a connection once and
open it again with a click or one command; save a snippet once and
insert it into any session without retyping it. It wraps your existing
OpenSSH client rather than reimplementing the SSH protocol, so
authentication, keys, and known hosts behave exactly as they do from a
plain terminal — nullssh just remembers the details you'd otherwise
retype.

## What's Inside

nullssh ships two ways to work, sharing the same saved connections:

* **Desktop app** — a sidebar of saved connections and a tabbed,
  embedded terminal for every open session, with light and dark
  appearance built in.
* **Command line** — a scriptable CLI for saving, listing, and
  connecting from a terminal or a script.

Core features across both:

* Save connections once: name, host, port, user, and optionally an
  identity file or a saved password.
* Embedded terminal sessions in the GUI. Open a saved connection as a
  tab with a real, interactive terminal. Open several at once and
  switch between them by clicking tabs.
* Edit and delete connections in place, including renaming.
* Browse for identity files with a native file picker that opens
  straight into `~/.ssh`.
* Saved passwords, protected by a master password. See
  [Authentication](#authentication) below.
* Snippets. Insert a saved command into the active session without
  submitting it, so you can adjust it (for example a session name)
  before pressing Enter. Comes with a set of built-in tmux snippets,
  and you can add your own under any category.
* Copy and paste that actually works: Ctrl+V, Shift+Insert, or
  right-click.
* Shared state. The GUI and CLI read and write the same saved
  connections, so either one stays in sync with the other.

## Design Principles

* **Wrap, don't reimplement.** nullssh hands connection details to the
  real `ssh.exe` and lets your normal key, agent, or password setup
  take it from there, exactly as if you'd typed the command yourself.
  No custom SSH implementation to trust.
* **Nothing typed twice.** If a value is already saved (a host, a
  snippet, a session name), you should never have to retype it to use
  it again.
* **Credentials stay out of memory longer than they need to.** Saved
  passwords are encrypted at rest and only decrypted in the Go
  backend for the moment they're typed into the terminal; the master
  password itself is never stored anywhere.
* **The GUI and CLI are one tool, not two.** Both read and write the
  same connection store, so switching between them mid-workflow never
  loses state.

## Install

Download the latest release from the
[Releases](https://github.com/BeardedTech0o/tether/releases) page.

| File | What it is |
|---|---|
| `tether-setup.exe` | The installer. Runs a Windows install wizard, adds Start Menu and Desktop shortcuts, and registers an uninstaller in "Add or remove programs". |
| `tether-portable.exe` | The GUI as a single portable exe. No install step, just run it. |
| `tether-cli.exe` | The portable command line tool. |

nullssh requires the OpenSSH client (`ssh`) to be on your `PATH`, which
is bundled with Windows 10 and later by default.

## Using the GUI

Launch nullssh (from the Start Menu if you used the installer, or by
running `tether-portable.exe`).

The first time you launch it, you'll be asked to set a master password.
See [Authentication](#authentication) below for what this protects and
why it's required.

1. Click **+** in the sidebar to save a connection: name, host, user,
   port, and optionally an identity file (use **Browse…** to pick a key
   from `~/.ssh`) or a password.
2. Click a saved connection to open it as a tab with a live terminal.
3. Open more connections to add more tabs, then click between tabs to
   switch which session you're looking at.
4. Close a tab (**✕**) to end that session.
5. Hover a saved connection to reveal **✎ Edit** (change any field or
   rename it) and **✕ Delete**.

The Snippets dropdown on the right of the tab bar lists saved snippets
grouped under category headers, starting with a built-in tmux category.
Click one to insert it into the active terminal without submitting it,
so you can edit it (for example swap out `mysession` for a real name)
before pressing Enter. Use **+ Add snippet** to save your own, giving
it a category, a label, and the command itself; a new category name
creates its own header, and reusing an existing one adds to it. Hover a
saved snippet to reveal **✎ Edit**, which also lets you delete it.

Paste into a terminal with Ctrl+V, Shift+Insert, or right-click.

Click **⚙** in the sidebar to open Settings, where you can switch the
app's appearance between Light, Dark, and System (follows your OS
setting and updates live if it changes).

![Tether Connection Settings](https://raw.githubusercontent.com/BeardedTech0o/tether/main/internal/tetherpw.png)

## Using the CLI

```
tether-cli add <name> --host <host> --user <user> [--port 22] [--identity path]
tether-cli ls
tether-cli connect <name>
tether-cli switch
tether-cli rm <name>
```

Running `tether-cli` with no arguments is the same as `tether-cli switch`.

Save a connection:
```
> tether-cli add prod --host 203.0.113.10 --user deploy --identity ~/.ssh/id_ed25519
saved connection "prod" (deploy@203.0.113.10:22)
```

List saved connections:
```
> tether-cli ls
NAME     TARGET                    LAST USED
prod     deploy@203.0.113.10:22   2026-08-14 21:03
staging  deploy@203.0.113.11:22   never
```

Connect directly, or pick interactively (most recently used first):
```
> tether-cli connect prod

> tether-cli switch
Saved connections (most recently used first):
  1) prod                 deploy@203.0.113.10:22
  2) staging               deploy@203.0.113.11:22
Select a connection (number or name), or q to quit:
```

Delete a connection:
```
> tether-cli rm staging
deleted connection "staging"
```

The CLI does not currently prompt for or use saved passwords (see
below); it authenticates the same way plain `ssh` does.

## Authentication

nullssh never handles your credentials directly. It hands connection
details to the real `ssh.exe` and lets your normal key, agent, or
password setup take it from there, exactly as if you'd typed the `ssh`
command yourself.

**Identity files.** Set one on a connection and nullssh passes it to
`ssh` as `-i <path>`. Leave it blank and `ssh` falls back to its own
defaults: keys already loaded in `ssh-agent` or Pageant, then the
standard files in `~/.ssh` (`id_ed25519`, `id_rsa`, and so on). If the
matching public key is authorized on the remote host, you'll connect
without a password prompt. If the key has a passphrase and isn't
already loaded in an agent, `ssh` will prompt for it once per session;
in the GUI this shows up in the embedded terminal, since it's a real
interactive `ssh` process underneath.

**Saved passwords and the master password.** The GUI can also save a
password on a connection, for servers where key based login isn't set
up. Saved passwords are encrypted at rest with a key derived from a
master password you set on first launch, and that master password is
required every time you open nullssh afterward. It is never stored
anywhere, only used in memory to unlock the vault for that run, and it
cannot be recovered if you forget it: connections with saved passwords
would need their passwords re-entered. When you open a connection with
a saved password, nullssh waits for ssh's own password prompt in the
terminal and types the decrypted password in for you; the password
never passes back through the browser layer of the app. Connections
without a saved password behave exactly as described above under
identity files.

Connections are stored as JSON in `%APPDATA%\tether\connections.json`,
shared by both the GUI and CLI. The master password configuration lives
alongside it in `%APPDATA%\tether\master.json` (just a salt and a
verification value, never the password itself), and snippets in
`%APPDATA%\tether\snippets.json`.

## How it's built

nullssh is two Go programs sharing a common core (`internal/store` for
saved connections, `internal/sshexec` for building `ssh` arguments):

* CLI (repo root): a small stdlib only Go program that execs the system
  `ssh` binary directly, with the terminal's own stdio.
* GUI (`cmd/tether-gui`): a [Wails](https://wails.io) app, a Go backend
  and an HTML/CSS/JS frontend using xterm.js. Each open tab spawns
  `ssh` attached to a Windows pseudo console (ConPTY) and streams its
  output into an embedded terminal. Saved passwords are encrypted with
  AES-256-GCM using a key derived from the master password via PBKDF2
  (see `cmd/tether-gui/vault`).

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

The ConPTY-backed terminal only works on Windows. `go build` and
`go vet` still succeed on other platforms for development, but opening
a GUI session will return an error since there's no pseudo console
backend for that OS.

## Contributing

This started as a personal tool tuned to one person's workflow, but if
you spot a bug or have a suggestion, open an issue.

## Licence

Add your licence here.
