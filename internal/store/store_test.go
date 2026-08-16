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

func TestMasterConfigRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	mc, err := LoadMasterConfig()
	if err != nil {
		t.Fatalf("LoadMasterConfig (before setup): %v", err)
	}
	if mc != nil {
		t.Fatalf("LoadMasterConfig (before setup) = %+v, want nil", mc)
	}

	want := &MasterConfig{Salt: "c2FsdA==", Check: "Y2hlY2s="}
	if err := SaveMasterConfig(want); err != nil {
		t.Fatalf("SaveMasterConfig: %v", err)
	}

	got, err := LoadMasterConfig()
	if err != nil {
		t.Fatalf("LoadMasterConfig (after setup): %v", err)
	}
	if got == nil || got.Salt != want.Salt || got.Check != want.Check {
		t.Fatalf("LoadMasterConfig() = %+v, want %+v", got, want)
	}
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

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(Connection{Name: "prod", Host: "1.2.3.4", User: "root", Port: 22}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Touch("prod"); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	lastUsed, _ := s.Get("prod")

	if err := s.Update("prod", Connection{Name: "prod-renamed", Host: "5.6.7.8", User: "deploy", Port: 2222}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if _, ok := s.Get("prod"); ok {
		t.Fatal("Get() found connection under old name after rename")
	}
	c, ok := s.Get("prod-renamed")
	if !ok {
		t.Fatal("Get() did not find connection under new name after rename")
	}
	if c.Host != "5.6.7.8" || c.User != "deploy" || c.Port != 2222 {
		t.Fatalf("updated connection mismatch: %+v", c)
	}
	if !c.LastUsed.Equal(lastUsed.LastUsed) {
		t.Fatalf("Update() did not preserve LastUsed: got %v, want %v", c.LastUsed, lastUsed.LastUsed)
	}
}

func TestUpdateRenameToExistingNameFails(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(Connection{Name: "prod", Host: "1.2.3.4", User: "root"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(Connection{Name: "staging", Host: "5.6.7.8", User: "deploy"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Update("prod", Connection{Name: "staging", Host: "1.2.3.4", User: "root"}); err == nil {
		t.Fatal("Update() renaming onto an existing name succeeded, want error")
	}
}

func TestUpdateMissingFails(t *testing.T) {
	s := newTestStore(t)
	if err := s.Update("nope", Connection{Name: "nope", Host: "h", User: "u"}); err == nil {
		t.Fatal("Update() of missing connection succeeded, want error")
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
