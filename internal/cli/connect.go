package cli

import (
	"fmt"
	"os"
)

func runConnect(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "tether connect: expected exactly one argument: the connection name")
		return 2
	}
	name := args[0]

	s, err := loadStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}

	c, ok := s.Get(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "tether connect: no connection named %q\n", name)
		return 1
	}

	// Record the connection attempt regardless of ssh's exit code — a failed
	// login still counts as "switching to" this connection.
	_ = s.Touch(name)
	_ = s.Save()

	return runSSH(c)
}
