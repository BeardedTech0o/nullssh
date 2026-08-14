package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func runSwitch() int {
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

	fmt.Println("Saved connections (most recently used first):")
	for i, c := range conns {
		fmt.Printf("  %d) %-20s %s@%s:%d\n", i+1, c.Name, c.User, c.Host, c.Port)
	}
	fmt.Print("Select a connection (number or name), or q to quit: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "tether:", err)
		return 1
	}
	input := strings.TrimSpace(line)

	if input == "" || input == "q" || input == "quit" {
		return 0
	}

	var target string
	if n, err := strconv.Atoi(input); err == nil {
		if n < 1 || n > len(conns) {
			fmt.Fprintf(os.Stderr, "tether: %d is out of range\n", n)
			return 1
		}
		target = conns[n-1].Name
	} else {
		target = input
	}

	c, ok := s.Get(target)
	if !ok {
		fmt.Fprintf(os.Stderr, "tether: no connection named %q\n", target)
		return 1
	}

	_ = s.Touch(target)
	_ = s.Save()

	return runSSH(c)
}
