package cli

import (
	"fmt"
	"os"
)

func runRemove(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "tether rm: expected exactly one argument: the connection name")
		return 2
	}
	name := args[0]

	s, err := loadStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}

	if err := s.Delete(name); err != nil {
		fmt.Fprintln(os.Stderr, "tether rm:", err)
		return 1
	}

	if err := s.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}

	fmt.Printf("deleted connection %q\n", name)
	return 0
}
