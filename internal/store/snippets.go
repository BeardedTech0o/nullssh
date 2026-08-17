package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Snippet is a short command a user can insert into a terminal session
// without submitting it, grouped under a Category for display (e.g.
// "tmux"). ID is assigned by the caller (the GUI) when a snippet is
// created, since this package doesn't depend on a UUID library.
type Snippet struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Label    string `json:"label"`
	Command  string `json:"command"`
}

// SnippetStore holds the loaded snippet list and knows how to persist
// itself.
type SnippetStore struct {
	path     string
	Snippets []Snippet
}

func snippetsFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "snippets.json"), nil
}

// defaultSnippets seeds a fresh install with the built-in tmux snippets, as
// regular (editable, deletable) data rather than something hardcoded in the
// frontend.
func defaultSnippets() []Snippet {
	return []Snippet{
		{ID: "tmux-new", Category: "tmux", Label: "New session", Command: "tmux new -s mysession"},
		{ID: "tmux-ls", Category: "tmux", Label: "List sessions", Command: "tmux ls"},
		{ID: "tmux-attach", Category: "tmux", Label: "Attach to session", Command: "tmux attach -t mysession"},
		{ID: "tmux-attach-detach-others", Category: "tmux", Label: "Attach, detaching others", Command: "tmux attach -d -t mysession"},
		{ID: "tmux-rename", Category: "tmux", Label: "Rename session", Command: "tmux rename-session -t mysession newname"},
		{ID: "tmux-kill", Category: "tmux", Label: "Kill session", Command: "tmux kill-session -t mysession"},
		{ID: "tmux-kill-others", Category: "tmux", Label: "Kill all other sessions", Command: "tmux kill-session -a"},
	}
}

// LoadSnippets reads the snippet store from disk. On first run (no file
// yet) it seeds and persists the default tmux snippets.
func LoadSnippets() (*SnippetStore, error) {
	path, err := snippetsFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	s := &SnippetStore{path: path}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		s.Snippets = defaultSnippets()
		if err := s.Save(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	if len(data) == 0 {
		s.Snippets = defaultSnippets()
		return s, nil
	}

	if err := json.Unmarshal(data, &s.Snippets); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return s, nil
}

// Save writes the snippet store back to disk, creating the config
// directory if needed.
func (s *SnippetStore) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(s.Snippets, "", "  ")
	if err != nil {
		return fmt.Errorf("encode snippets: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", s.path, err)
	}
	return nil
}

// Add appends a new snippet.
func (s *SnippetStore) Add(sn Snippet) {
	s.Snippets = append(s.Snippets, sn)
}

// Update replaces the snippet with the given ID.
func (s *SnippetStore) Update(id string, sn Snippet) error {
	for i, existing := range s.Snippets {
		if existing.ID == id {
			sn.ID = id
			s.Snippets[i] = sn
			return nil
		}
	}
	return fmt.Errorf("no snippet with id %q", id)
}

// Delete removes the snippet with the given ID.
func (s *SnippetStore) Delete(id string) error {
	for i, sn := range s.Snippets {
		if sn.ID == id {
			s.Snippets = append(s.Snippets[:i], s.Snippets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("no snippet with id %q", id)
}
