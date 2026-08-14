package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/BeardedTech0o/tether/internal/store"
)

func runAdd(args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprintln(os.Stderr, "tether add: expected exactly one argument: the connection name")
		return 2
	}
	name := args[0]

	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	host := fs.String("host", "", "hostname or IP address to connect to (required)")
	user := fs.String("user", "", "SSH username (required)")
	port := fs.Int("port", 22, "SSH port")
	identity := fs.String("identity", "", "path to a private key file (optional)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "tether add: unexpected extra arguments:", fs.Args())
		return 2
	}

	if *host == "" || *user == "" {
		fmt.Fprintln(os.Stderr, "tether add: --host and --user are required")
		return 2
	}

	s, err := loadStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}

	c := store.Connection{
		Name:         name,
		Host:         *host,
		User:         *user,
		Port:         *port,
		IdentityFile: *identity,
	}

	if err := s.Add(c); err != nil {
		fmt.Fprintln(os.Stderr, "tether add:", err)
		return 1
	}

	if err := s.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}

	fmt.Printf("saved connection %q (%s@%s:%d)\n", name, *user, *host, *port)
	return 0
}
