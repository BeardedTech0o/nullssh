// Package cli implements the tether subcommands.
package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/BeardedTech0o/tether/internal/sshexec"
	"github.com/BeardedTech0o/tether/internal/store"
)

const usage = `tether — a simple SSH connection manager

Usage:
  tether add <name> --host <host> --user <user> [--port 22] [--identity path]
  tether ls
  tether rm <name>
  tether connect <name>
  tether switch
  tether help

With no arguments, tether behaves like "tether switch".
`

// Run dispatches to the requested subcommand and returns the process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		return runSwitch()
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "add":
		return runAdd(rest)
	case "ls", "list":
		return runList(rest)
	case "rm", "remove", "delete":
		return runRemove(rest)
	case "connect":
		return runConnect(rest)
	case "switch":
		return runSwitch()
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "tether: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}

// runSSH execs the system ssh binary against c, with stdio inherited from the
// current process, and returns the exit code ssh terminated with.
func runSSH(c store.Connection) int {
	cmd := exec.Command("ssh", sshexec.Args(c)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "tether: failed to run ssh: %v\n", err)
		return 1
	}
	return 0
}

func loadStore() (*store.Store, error) {
	s, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("load connections: %w", err)
	}
	return s, nil
}
