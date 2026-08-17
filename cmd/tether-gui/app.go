package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/BeardedTech0o/tether/cmd/tether-gui/vault"
	"github.com/BeardedTech0o/tether/internal/store"
)

// App is bound to the frontend and exposes connection management, the
// master-password vault (below), and the session manager (see session.go).
type App struct {
	ctx      context.Context
	sessions *sessionManager

	// vaultKey is the key derived from the master password, held only in
	// memory for the lifetime of the running app; it is never persisted.
	// nil means the vault is locked (no master password created/entered
	// yet this run).
	vaultMu  sync.RWMutex
	vaultKey []byte
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

// --- Master password vault ---

// HasMasterPassword reports whether a master password has already been set
// up on this machine.
func (a *App) HasMasterPassword() (bool, error) {
	mc, err := store.LoadMasterConfig()
	if err != nil {
		return false, fmt.Errorf("load master config: %w", err)
	}
	return mc != nil, nil
}

// CreateMasterPassword sets up the master password for the first time. It
// fails if one has already been created.
func (a *App) CreateMasterPassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("master password must be at least 8 characters")
	}

	existing, err := store.LoadMasterConfig()
	if err != nil {
		return fmt.Errorf("load master config: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("a master password has already been set")
	}

	salt, err := vault.NewSalt()
	if err != nil {
		return err
	}
	key, err := vault.DeriveKey(password, salt)
	if err != nil {
		return err
	}
	check, err := vault.EncryptString(key, vault.CheckPlaintext)
	if err != nil {
		return fmt.Errorf("encrypt check value: %w", err)
	}

	mc := &store.MasterConfig{
		Salt:  base64.StdEncoding.EncodeToString(salt),
		Check: check,
	}
	if err := store.SaveMasterConfig(mc); err != nil {
		return fmt.Errorf("save master config: %w", err)
	}

	a.vaultMu.Lock()
	a.vaultKey = key
	a.vaultMu.Unlock()
	return nil
}

// UnlockMasterPassword verifies the given password against the stored
// master config and, if correct, derives and holds the vault key for the
// rest of this run.
func (a *App) UnlockMasterPassword(password string) error {
	mc, err := store.LoadMasterConfig()
	if err != nil {
		return fmt.Errorf("load master config: %w", err)
	}
	if mc == nil {
		return fmt.Errorf("no master password has been set up")
	}

	salt, err := base64.StdEncoding.DecodeString(mc.Salt)
	if err != nil {
		return fmt.Errorf("decode salt: %w", err)
	}
	key, err := vault.DeriveKey(password, salt)
	if err != nil {
		return err
	}

	got, err := vault.DecryptString(key, mc.Check)
	if err != nil || got != vault.CheckPlaintext {
		return fmt.Errorf("incorrect master password")
	}

	a.vaultMu.Lock()
	a.vaultKey = key
	a.vaultMu.Unlock()
	return nil
}

func (a *App) getVaultKey() ([]byte, error) {
	a.vaultMu.RLock()
	defer a.vaultMu.RUnlock()
	if a.vaultKey == nil {
		return nil, fmt.Errorf("vault is locked")
	}
	return a.vaultKey, nil
}

func (a *App) encryptPassword(plaintext string) (string, error) {
	key, err := a.getVaultKey()
	if err != nil {
		return "", err
	}
	return vault.EncryptString(key, plaintext)
}

func (a *App) decryptPassword(ciphertext string) (string, error) {
	key, err := a.getVaultKey()
	if err != nil {
		return "", err
	}
	return vault.DecryptString(key, ciphertext)
}

// --- Connections ---

// ConnectionView is what ListConnections sends to the frontend: everything
// about a saved connection except the encrypted password itself, which the
// frontend never needs to see.
type ConnectionView struct {
	Name         string    `json:"name"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	User         string    `json:"user"`
	IdentityFile string    `json:"identity_file,omitempty"`
	LastUsed     time.Time `json:"last_used,omitempty"`
	HasPassword  bool      `json:"hasPassword"`
}

// ListConnections returns all saved connections, most recently used first.
func (a *App) ListConnections() ([]ConnectionView, error) {
	s, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("load connections: %w", err)
	}

	conns := s.List()
	views := make([]ConnectionView, len(conns))
	for i, c := range conns {
		views[i] = ConnectionView{
			Name:         c.Name,
			Host:         c.Host,
			Port:         c.Port,
			User:         c.User,
			IdentityFile: c.IdentityFile,
			LastUsed:     c.LastUsed,
			HasPassword:  c.Password != "",
		}
	}
	return views, nil
}

// AddConnectionInput mirrors store.Connection but omits fields the frontend
// shouldn't set directly (LastUsed), and carries a plaintext Password that
// gets encrypted before it's ever written to disk.
type AddConnectionInput struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	IdentityFile string `json:"identityFile"`
	// Password is plaintext from the form. Empty means "no change" on
	// Update (see ClearPassword to explicitly remove a saved password),
	// or "no saved password" on Add.
	Password string `json:"password"`
	// ClearPassword, on Update only, removes a previously saved password
	// even though Password is left blank.
	ClearPassword bool `json:"clearPassword"`
}

func (in AddConnectionInput) toConnection() (store.Connection, error) {
	if in.Name == "" || in.Host == "" || in.User == "" {
		return store.Connection{}, fmt.Errorf("name, host, and user are required")
	}
	if in.Port == 0 {
		in.Port = 22
	}
	return store.Connection{
		Name:         in.Name,
		Host:         in.Host,
		Port:         in.Port,
		User:         in.User,
		IdentityFile: in.IdentityFile,
	}, nil
}

// AddConnection saves a new connection.
func (a *App) AddConnection(in AddConnectionInput) error {
	c, err := in.toConnection()
	if err != nil {
		return err
	}

	if in.Password != "" {
		enc, err := a.encryptPassword(in.Password)
		if err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
		c.Password = enc
	}

	s, err := store.Load()
	if err != nil {
		return fmt.Errorf("load connections: %w", err)
	}

	if err := s.Add(c); err != nil {
		return err
	}

	return s.Save()
}

// UpdateConnection replaces the connection named oldName with in, which may
// rename it. Leaving Password blank keeps the existing saved password (if
// any); set ClearPassword to remove it instead.
func (a *App) UpdateConnection(oldName string, in AddConnectionInput) error {
	c, err := in.toConnection()
	if err != nil {
		return err
	}

	s, err := store.Load()
	if err != nil {
		return fmt.Errorf("load connections: %w", err)
	}

	switch {
	case in.ClearPassword:
		c.Password = ""
	case in.Password != "":
		enc, err := a.encryptPassword(in.Password)
		if err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
		c.Password = enc
	default:
		if existing, ok := s.Get(oldName); ok {
			c.Password = existing.Password
		}
	}

	if err := s.Update(oldName, c); err != nil {
		return err
	}

	return s.Save()
}

// BrowseIdentityFile opens a native file picker starting in the user's
// .ssh directory (falling back to their home directory if .ssh doesn't
// exist) so they can pick a private key file. Returns "" with no error if
// the user cancels the dialog.
func (a *App) BrowseIdentityFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	dir := filepath.Join(home, ".ssh")
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		dir = home
	}

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "Select SSH identity file",
		DefaultDirectory: dir,
		ShowHiddenFiles:  true,
	})
	if err != nil {
		return "", fmt.Errorf("open file dialog: %w", err)
	}
	return path, nil
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

// --- Snippets ---

// ListSnippets returns all saved snippets. The frontend groups them by
// Category for display.
func (a *App) ListSnippets() ([]store.Snippet, error) {
	s, err := store.LoadSnippets()
	if err != nil {
		return nil, fmt.Errorf("load snippets: %w", err)
	}
	return s.Snippets, nil
}

// SnippetInput is what the frontend submits when creating or editing a
// snippet.
type SnippetInput struct {
	Category string `json:"category"`
	Label    string `json:"label"`
	Command  string `json:"command"`
}

func (in SnippetInput) validate() error {
	if in.Category == "" || in.Label == "" || in.Command == "" {
		return fmt.Errorf("category, label, and command are required")
	}
	return nil
}

// AddSnippet saves a new snippet and returns it (with its assigned ID).
func (a *App) AddSnippet(in SnippetInput) (store.Snippet, error) {
	if err := in.validate(); err != nil {
		return store.Snippet{}, err
	}

	s, err := store.LoadSnippets()
	if err != nil {
		return store.Snippet{}, fmt.Errorf("load snippets: %w", err)
	}

	sn := store.Snippet{
		ID:       uuid.NewString(),
		Category: in.Category,
		Label:    in.Label,
		Command:  in.Command,
	}
	s.Add(sn)

	if err := s.Save(); err != nil {
		return store.Snippet{}, fmt.Errorf("save snippets: %w", err)
	}
	return sn, nil
}

// UpdateSnippet replaces the snippet with the given ID.
func (a *App) UpdateSnippet(id string, in SnippetInput) error {
	if err := in.validate(); err != nil {
		return err
	}

	s, err := store.LoadSnippets()
	if err != nil {
		return fmt.Errorf("load snippets: %w", err)
	}

	if err := s.Update(id, store.Snippet{
		Category: in.Category,
		Label:    in.Label,
		Command:  in.Command,
	}); err != nil {
		return err
	}

	return s.Save()
}

// DeleteSnippet removes a saved snippet by ID.
func (a *App) DeleteSnippet(id string) error {
	s, err := store.LoadSnippets()
	if err != nil {
		return fmt.Errorf("load snippets: %w", err)
	}
	if err := s.Delete(id); err != nil {
		return err
	}
	return s.Save()
}

// OpenSession starts a new terminal session for the named connection and
// returns its session ID. The frontend should subscribe to
// "session:<id>:data" and "session:<id>:closed" events before/after calling
// this. If the connection has a saved password, it's decrypted here and
// handed to the session manager to auto-fill ssh's password prompt; the
// plaintext never crosses back into the frontend.
func (a *App) OpenSession(name string) (id string, err error) {
	defer recoverToError(&err)

	s, err := store.Load()
	if err != nil {
		return "", fmt.Errorf("load connections: %w", err)
	}
	c, ok := s.Get(name)
	if !ok {
		return "", fmt.Errorf("no connection named %q", name)
	}

	var password string
	if c.Password != "" {
		if p, err := a.decryptPassword(c.Password); err == nil {
			password = p
		}
	}

	_ = s.Touch(name)
	_ = s.Save()

	return a.sessions.open(c, password)
}

// WriteToSession sends input (keystrokes) to an open session's PTY.
func (a *App) WriteToSession(id string, data string) (err error) {
	defer recoverToError(&err)
	return a.sessions.write(id, data)
}

// ResizeSession resizes an open session's PTY to match the frontend terminal.
func (a *App) ResizeSession(id string, cols int, rows int) (err error) {
	defer recoverToError(&err)
	return a.sessions.resize(id, cols, rows)
}

// CloseSession terminates an open session and its ssh process.
func (a *App) CloseSession(id string) (err error) {
	defer recoverToError(&err)
	return a.sessions.close(id)
}

// recoverToError turns a panic in the deferring function into an error
// instead of letting it escape. A Wails app is a single OS process shared
// with the window itself, so an unrecovered panic on any goroutine —
// including one triggered by a frontend-bound method call — would crash the
// entire app, not just the feature that panicked.
func recoverToError(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("internal error: %v", r)
	}
}
