//go:build !windows

package main

import (
	"fmt"

	"github.com/BeardedTech0o/tether/internal/store"
)

// startPTY is only implemented on Windows (ConPTY). This stub keeps the
// module building on other platforms for local development.
func startPTY(c store.Connection) (pty, error) {
	return nil, fmt.Errorf("tether-gui terminal sessions are only supported on Windows")
}
