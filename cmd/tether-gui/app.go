package main

import (
	"context"
	"fmt"

	"github.com/BeardedTech0o/tether/internal/store"
)

// App is bound to the frontend and exposes connection management plus the
// session manager (see session.go).
type App struct {
	ctx      context.Context
	sessions *sessionManager
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so we can
// call the runtime methods (e.g. emitting events to the frontend).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.sessions = newSessionManager(ctx)
}

// shutdown is called when the app is closing; make sure no ssh processes
// are left running.
func (a *App) shutdown(ctx context.Context) {
	if a.sessions != nil {
		a.sessions.closeAll()
	}
}

// ListConnections returns all saved connections, most recently used first.
func (a *App) ListConnections() ([]store.Connection, error) {
	s, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("load connections: %w", err)
	}
	return s.List(), nil
}

// AddConnectionInput mirrors store.Connection but omits fields the frontend
// shouldn't set directly (LastUsed).
type AddConnectionInput struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	IdentityFile string `json:"identityFile"`
}

// AddConnection saves a new connection.
func (a *App) AddConnection(in AddConnectionInput) error {
	if in.Name == "" || in.Host == "" || in.User == "" {
		return fmt.Errorf("name, host, and user are required")
	}
	if in.Port == 0 {
		in.Port = 22
	}

	s, err := store.Load()
	if err != nil {
		return fmt.Errorf("load connections: %w", err)
	}

	if err := s.Add(store.Connection{
		Name:         in.Name,
		Host:         in.Host,
		Port:         in.Port,
		User:         in.User,
		IdentityFile: in.IdentityFile,
	}); err != nil {
		return err
	}

	return s.Save()
}

// DeleteConnection removes a saved connection by name.
func (a *App) DeleteConnection(name string) error {
	s, err := store.Load()
	if err != nil {
		return fmt.Errorf("load connections: %w", err)
	}
	if err := s.Delete(name); err != nil {
		return err
	}
	return s.Save()
}

// OpenSession starts a new terminal session for the named connection and
// returns its session ID. The frontend should subscribe to
// "session:<id>:data" and "session:<id>:closed" events before/after calling
// this.
func (a *App) OpenSession(name string) (string, error) {
	s, err := store.Load()
	if err != nil {
		return "", fmt.Errorf("load connections: %w", err)
	}
	c, ok := s.Get(name)
	if !ok {
		return "", fmt.Errorf("no connection named %q", name)
	}

	_ = s.Touch(name)
	_ = s.Save()

	return a.sessions.open(c)
}

// WriteToSession sends input (keystrokes) to an open session's PTY.
func (a *App) WriteToSession(id string, data string) error {
	return a.sessions.write(id, data)
}

// ResizeSession resizes an open session's PTY to match the frontend terminal.
func (a *App) ResizeSession(id string, cols int, rows int) error {
	return a.sessions.resize(id, cols, rows)
}

// CloseSession terminates an open session and its ssh process.
func (a *App) CloseSession(id string) error {
	return a.sessions.close(id)
}
