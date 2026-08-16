//go:build windows

package main

import (
	"os"
	"strings"
	"sync"

	"github.com/UserExistsError/conpty"

	"github.com/BeardedTech0o/tether/internal/sshexec"
	"github.com/BeardedTech0o/tether/internal/store"
)

// conptyPTY adapts *conpty.ConPty to the pty interface. Close may be called
// both by an explicit CloseSession request and by the read-pump's own
// cleanup once the process exits, so it's made idempotent with closeOnce —
// calling conpty.Close twice, or closing while a Read is still in flight,
// is not guaranteed to be safe otherwise.
type conptyPTY struct {
	cpty      *conpty.ConPty
	closeOnce sync.Once
	closeErr  error
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
	p.closeOnce.Do(func() {
		// conpty.Close only closes handles/pipes; despite its doc comment it
		// does not actually call TerminateProcess, so the spawned ssh
		// process (and anything it's still waiting on, e.g. a remote tmux
		// session) can keep running and leave a pending Read() that never
		// unblocks. Kill it explicitly by PID first.
		if proc, err := os.FindProcess(p.cpty.Pid()); err == nil {
			_ = proc.Kill()
		}
		p.closeErr = p.cpty.Close()
	})
	return p.closeErr
}
