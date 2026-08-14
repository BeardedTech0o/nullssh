package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{path: filepath.Join(t.TempDir(), "connections.json")}
}

func TestAddAndList(t *testing.T) {
	s := newTestStore(t)

	if err := s.Add(Connection{Name: "prod", Host: "1.2.3.4", User: "root", Port: 22}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(Connection{Name: "staging", Host: "5.6.7.8", User: "deploy", Port: 22}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := s.List()
	if len(got) != 2 {
		t.Fatalf("List() returned %d connections, want 2", len(got))
	}
}

func TestAddDuplicateNameFails(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(Connection{Name: "prod", Host: "1.2.3.4", User: "root"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(Connection{Name: "prod", Host: "9.9.9.9", User: "root"}); err == nil {
		t.Fatal("Add() with duplicate name succeeded, want error")
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(Connection{Name: "prod", Host: "1.2.3.4", User: "root"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Delete("prod"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := s.Get("prod"); ok {
		t.Fatal("Get() found connection after Delete")
	}
	if err := s.Delete("prod"); err == nil {
		t.Fatal("Delete() of missing connection succeeded, want error")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	s := &Store{path: path}
	if err := s.Add(Connection{Name: "prod", Host: "1.2.3.4", User: "root", Port: 2222, IdentityFile: "~/.ssh/id_ed25519"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s2 := &Store{path: path}
	if err := json.Unmarshal(raw, &s2.Connections); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	c, ok := s2.Get("prod")
	if !ok {
		t.Fatal("Get() did not find connection after round trip")
	}
	if c.Host != "1.2.3.4" || c.Port != 2222 || c.User != "root" || c.IdentityFile != "~/.ssh/id_ed25519" {
		t.Fatalf("round-tripped connection mismatch: %+v", c)
	}
}

func TestTouchUpdatesOrdering(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(Connection{Name: "a", Host: "h", User: "u"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(Connection{Name: "b", Host: "h", User: "u"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Touch("b"); err != nil {
		t.Fatalf("Touch: %v", err)
	}

	got := s.List()
	if got[0].Name != "b" {
		t.Fatalf("List()[0].Name = %q, want %q (most recently used first)", got[0].Name, "b")
	}
}
