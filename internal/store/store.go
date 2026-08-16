// Package store persists saved SSH connections as JSON in the user's config dir.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Connection struct {
	Name         string    `json:"name"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	User         string    `json:"user"`
	IdentityFile string    `json:"identity_file,omitempty"`
	LastUsed     time.Time `json:"last_used,omitempty"`

	// Password holds an opaque, already-encrypted value (produced by the
	// GUI's vault) when a password has been saved for this connection.
	// Empty means no saved password. This package never sees or handles
	// the plaintext; encryption and decryption happen entirely in the GUI.
	Password string `json:"password,omitempty"`
}

// MasterConfig persists the parameters needed to verify a master password
// and derive the key that encrypts/decrypts saved connection passwords.
// The master password itself is never stored.
type MasterConfig struct {
	// Salt is base64-encoded random bytes used to derive the vault key
	// from the master password via PBKDF2.
	Salt string `json:"salt"`
	// Check is an opaque value (AES-GCM ciphertext of a fixed known
	// plaintext) that a correctly-derived key must decrypt successfully.
	// It lets the GUI verify an entered master password without ever
	// storing the password or a directly-reversible hash of it.
	Check string `json:"check"`
}

// Store holds the loaded connection list and knows how to persist itself.
type Store struct {
	path        string
	Connections []Connection
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tether"), nil
}

func filePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "connections.json"), nil
}

func masterFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "master.json"), nil
}

// LoadMasterConfig reads the master password configuration, returning nil
// (with no error) if a master password has never been set up.
func LoadMasterConfig() (*MasterConfig, error) {
	path, err := masterFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var mc MasterConfig
	if err := json.Unmarshal(data, &mc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &mc, nil
}

// SaveMasterConfig writes the master password configuration to disk,
// creating the config directory if needed.
func SaveMasterConfig(mc *MasterConfig) error {
	path, err := masterFilePath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode master config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Load reads the connection store from disk, returning an empty store if it
// does not exist yet.
func Load() (*Store, error) {
	path, err := filePath()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	s := &Store{path: path}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if len(data) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(data, &s.Connections); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return s, nil
}

// Save writes the store back to disk, creating the config directory if needed.
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(s.Connections, "", "  ")
	if err != nil {
		return fmt.Errorf("encode connections: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", s.path, err)
	}
	return nil
}

// Add appends a new connection. It fails if a connection with the same name
// already exists.
func (s *Store) Add(c Connection) error {
	if _, ok := s.find(c.Name); ok {
		return fmt.Errorf("a connection named %q already exists", c.Name)
	}
	s.Connections = append(s.Connections, c)
	return nil
}

// Get returns the connection with the given name.
func (s *Store) Get(name string) (Connection, bool) {
	i, ok := s.find(name)
	if !ok {
		return Connection{}, false
	}
	return s.Connections[i], true
}

// Update replaces the connection named oldName with c, preserving its
// LastUsed timestamp. c.Name may differ from oldName to rename the
// connection, but fails if another connection already has that name.
func (s *Store) Update(oldName string, c Connection) error {
	i, ok := s.find(oldName)
	if !ok {
		return fmt.Errorf("no connection named %q", oldName)
	}
	if c.Name != oldName {
		if _, exists := s.find(c.Name); exists {
			return fmt.Errorf("a connection named %q already exists", c.Name)
		}
	}
	c.LastUsed = s.Connections[i].LastUsed
	s.Connections[i] = c
	return nil
}

// Delete removes the connection with the given name. It fails if no such
// connection exists.
func (s *Store) Delete(name string) error {
	i, ok := s.find(name)
	if !ok {
		return fmt.Errorf("no connection named %q", name)
	}
	s.Connections = append(s.Connections[:i], s.Connections[i+1:]...)
	return nil
}

// Touch updates a connection's LastUsed timestamp to now.
func (s *Store) Touch(name string) error {
	i, ok := s.find(name)
	if !ok {
		return fmt.Errorf("no connection named %q", name)
	}
	s.Connections[i].LastUsed = time.Now()
	return nil
}

// List returns all connections sorted by most-recently-used first, then by
// name for connections that have never been used.
func (s *Store) List() []Connection {
	out := make([]Connection, len(s.Connections))
	copy(out, s.Connections)
	sort.SliceStable(out, func(i, j int) bool {
		li, lj := out[i].LastUsed, out[j].LastUsed
		if li.IsZero() && lj.IsZero() {
			return out[i].Name < out[j].Name
		}
		if li.IsZero() != lj.IsZero() {
			return lj.IsZero() // non-zero (used) sorts before zero (never used)
		}
		return li.After(lj)
	})
	return out
}

func (s *Store) find(name string) (int, bool) {
	for i, c := range s.Connections {
		if c.Name == name {
			return i, true
		}
	}
	return 0, false
}
