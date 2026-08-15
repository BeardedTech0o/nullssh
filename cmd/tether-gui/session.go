package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/BeardedTech0o/tether/internal/store"
)

// pty abstracts a pseudo-terminal running "ssh" for a saved connection.
// The real implementation (pty_windows.go) uses ConPTY; pty_other.go is a
// stub for non-Windows dev builds.
type pty interface {
	// Read blocks until output is available and returns it, or returns an
	// error (including io.EOF-like behavior) once the process has exited.
	Read(buf []byte) (int, error)
	Write(data []byte) (int, error)
	Resize(cols, rows int) error
	Close() error
}

// startPTY spawns "ssh <sshexec.Args(c)>" attached to a new pseudo-terminal.
// Implemented per-OS: see pty_windows.go (build tag windows) and
// pty_other.go (build tag !windows).

type sessionManager struct {
	ctx context.Context

	mu       sync.Mutex
	sessions map[string]pty
}

func newSessionManager(ctx context.Context) *sessionManager {
	return &sessionManager{
		ctx:      ctx,
		sessions: make(map[string]pty),
	}
}

func (m *sessionManager) open(c store.Connection) (string, error) {
	p, err := startPTY(c)
	if err != nil {
		return "", fmt.Errorf("start session: %w", err)
	}

	id := uuid.NewString()

	m.mu.Lock()
	m.sessions[id] = p
	m.mu.Unlock()

	go m.pump(id, p)

	return id, nil
}

// pump reads PTY output and forwards it to the frontend as events until the
// process exits or the PTY is closed. It always cleans up and notifies the
// frontend exactly once, even if reading panics — a panic on any goroutine
// in a Wails app would otherwise crash the whole window, not just this
// session, since the backend and the window share one OS process.
func (m *sessionManager) pump(id string, p pty) {
	defer func() {
		recover()

		m.mu.Lock()
		delete(m.sessions, id)
		m.mu.Unlock()

		_ = p.Close()
		runtime.EventsEmit(m.ctx, "session:"+id+":closed")
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := p.Read(buf)
		if n > 0 {
			runtime.EventsEmit(m.ctx, "session:"+id+":data", string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

func (m *sessionManager) get(id string) (pty, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("no such session %q", id)
	}
	return p, nil
}

func (m *sessionManager) write(id string, data string) error {
	p, err := m.get(id)
	if err != nil {
		return err
	}
	_, err = p.Write([]byte(data))
	return err
}

func (m *sessionManager) resize(id string, cols, rows int) error {
	p, err := m.get(id)
	if err != nil {
		return err
	}
	return p.Resize(cols, rows)
}

func (m *sessionManager) close(id string) error {
	p, err := m.get(id)
	if err != nil {
		return err
	}
	return p.Close()
}

func (m *sessionManager) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.sessions {
		_ = p.Close()
	}
}
