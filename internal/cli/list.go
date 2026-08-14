package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func runList(args []string) int {
	s, err := loadStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}

	conns := s.List()
	if len(conns) == 0 {
		fmt.Println("no saved connections. Add one with: tether add <name> --host <host> --user <user>")
		return 0
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTARGET\tLAST USED")
	for _, c := range conns {
		lastUsed := "never"
		if !c.LastUsed.IsZero() {
			lastUsed = c.LastUsed.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s@%s:%d\t%s\n", c.Name, c.User, c.Host, c.Port, lastUsed)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}
	return 0
}
