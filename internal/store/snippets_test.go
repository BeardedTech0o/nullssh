package store

import "testing"

func TestLoadSnippetsSeedsDefaultsOnFirstRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	s, err := LoadSnippets()
	if err != nil {
		t.Fatalf("LoadSnippets: %v", err)
	}
	if len(s.Snippets) == 0 {
		t.Fatal("LoadSnippets() on first run returned no snippets, want seeded tmux defaults")
	}
	for _, sn := range s.Snippets {
		if sn.Category != "tmux" {
			t.Fatalf("seeded snippet %+v has category %q, want \"tmux\"", sn, sn.Category)
		}
	}

	// A second load should read back the persisted file, not reseed.
	s2, err := LoadSnippets()
	if err != nil {
		t.Fatalf("LoadSnippets (second load): %v", err)
	}
	if len(s2.Snippets) != len(s.Snippets) {
		t.Fatalf("LoadSnippets (second load) returned %d snippets, want %d", len(s2.Snippets), len(s.Snippets))
	}
}

func TestSnippetAddUpdateDelete(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	s, err := LoadSnippets()
	if err != nil {
		t.Fatalf("LoadSnippets: %v", err)
	}
	seeded := len(s.Snippets)

	s.Add(Snippet{ID: "custom-1", Category: "Deploy", Label: "Restart service", Command: "sudo systemctl restart myapp"})
	if len(s.Snippets) != seeded+1 {
		t.Fatalf("after Add, len(Snippets) = %d, want %d", len(s.Snippets), seeded+1)
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := LoadSnippets()
	if err != nil {
		t.Fatalf("LoadSnippets (after add): %v", err)
	}
	if len(reloaded.Snippets) != seeded+1 {
		t.Fatalf("LoadSnippets (after add) returned %d snippets, want %d", len(reloaded.Snippets), seeded+1)
	}

	if err := reloaded.Update("custom-1", Snippet{Category: "Deploy", Label: "Restart service (renamed)", Command: "sudo systemctl restart myapp"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	found := false
	for _, sn := range reloaded.Snippets {
		if sn.ID == "custom-1" {
			found = true
			if sn.Label != "Restart service (renamed)" {
				t.Fatalf("Update() did not change Label, got %+v", sn)
			}
		}
	}
	if !found {
		t.Fatal("Update() lost the snippet's ID")
	}

	if err := reloaded.Delete("custom-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(reloaded.Snippets) != seeded {
		t.Fatalf("after Delete, len(Snippets) = %d, want %d", len(reloaded.Snippets), seeded)
	}

	if err := reloaded.Delete("custom-1"); err == nil {
		t.Fatal("Delete() of already-removed snippet succeeded, want error")
	}
	if err := reloaded.Update("nope", Snippet{}); err == nil {
		t.Fatal("Update() of missing snippet succeeded, want error")
	}
}
