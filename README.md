# tether

A small command-line SSH connection manager for Windows. Save your SSH
connections once, then connect or switch between them by name — tether
wraps your existing `ssh` client, it doesn't reimplement the protocol.

## Install

Download `tether.exe` from the [Releases](https://github.com/BeardedTech0o/tether/releases)
page, or build it yourself:

```sh
git clone https://github.com/BeardedTech0o/tether.git
cd tether
GOOS=windows GOARCH=amd64 go build -o tether.exe .
```

Requires the OpenSSH client (`ssh`) to be on your `PATH` — it's bundled
with Windows 10 and later.

## Usage

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
(`%APPDATA%\tether\connections.json` on Windows). `connect`/`switch` shell
out to the system `ssh` binary with the saved host, port, user, and
identity file, so authentication (keys, agent, password prompts, known
hosts) works exactly as it does with plain `ssh`. No credentials are
stored by tether itself.

## Development

```sh
go vet ./...
go test ./...
go build -o tether.exe .          # native build
GOOS=windows GOARCH=amd64 go build -o tether.exe .   # cross-compile for Windows
```
