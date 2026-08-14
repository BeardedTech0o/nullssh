//go:build windows

package main

import (
	"strings"

	"github.com/UserExistsError/conpty"

	"github.com/BeardedTech0o/tether/internal/sshexec"
	"github.com/BeardedTech0o/tether/internal/store"
)

// conptyPTY adapts *conpty.ConPty to the pty interface.
type conptyPTY struct {
	cpty *conpty.ConPty
}

func startPTY(c store.Connection) (pty, error) {
	args := append([]string{"ssh"}, sshexec.Args(c)...)
	commandLine := strings.Join(args, " ")

	cpty, err := conpty.Start(commandLine)
	if err != nil {
		return nil, err
	}
	return &conptyPTY{cpty: cpty}, nil
}

func (p *conptyPTY) Read(buf []byte) (int, error) {
	return p.cpty.Read(buf)
}

func (p *conptyPTY) Write(data []byte) (int, error) {
	return p.cpty.Write(data)
}

func (p *conptyPTY) Resize(cols, rows int) error {
	return p.cpty.Resize(cols, rows)
}

func (p *conptyPTY) Close() error {
	return p.cpty.Close()
}
