// Package sshexec builds ssh command-line arguments for a saved connection.
// Shared by the CLI (which execs the system ssh directly) and the GUI
// (which spawns ssh attached to a pseudo-terminal).
package sshexec

import (
	"fmt"
	"os"
	"strings"

	"github.com/BeardedTech0o/tether/internal/store"
)

// Validate rejects Host/User values that could never be legitimate
// hostnames or usernames, as defense-in-depth alongside proper argument
// quoting at the exec/spawn site: a Host or User containing whitespace or a
// leading "-" is either malformed input or an attempt to smuggle extra
// flags into the ssh invocation (e.g. "-oProxyCommand=..."), so it's
// rejected before it can ever be saved to connections.json.
func Validate(c store.Connection) error {
	for field, v := range map[string]string{"host": c.Host, "user": c.User} {
		if strings.ContainsAny(v, " \t\r\n") {
			return fmt.Errorf("%s must not contain whitespace", field)
		}
		if strings.HasPrefix(v, "-") {
			return fmt.Errorf("%s must not start with \"-\"", field)
		}
	}
	return nil
}

// Args builds the argument list for invoking the system ssh client for c.
func Args(c store.Connection) []string {
	args := []string{"-p", fmt.Sprint(c.Port)}
	if c.IdentityFile != "" {
		args = append(args, "-i", ExpandHome(c.IdentityFile))
	}
	args = append(args, fmt.Sprintf("%s@%s", c.User, c.Host))
	return args
}

// ExpandHome expands a leading "~" in path to the current user's home directory.
func ExpandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}
