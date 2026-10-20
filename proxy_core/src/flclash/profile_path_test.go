package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileConfigPathMatchesArkTSPersistence(t *testing.T) {
	home := t.TempDir()
	id := "selected-profile"
	current := filepath.Join(home, "profiles", id, "config.yaml")
	legacy := filepath.Join(home, "profiles", id+".yaml")
	if err := os.MkdirAll(filepath.Dir(current), 0700); err != nil {
		t.Fatal(err)
	}
	if got := profileConfigPath(home, id); got != current {
		t.Fatalf("missing profile path: %s", got)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := profileConfigPath(home, id); got != legacy {
		t.Fatalf("legacy compatibility: %s", got)
	}
	if err := os.WriteFile(current, []byte("current"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := profileConfigPath(home, id); got != current {
		t.Fatalf("must prefer current saved profile over stale legacy copy: %s", got)
	}
	data, err := os.ReadFile(profileConfigPath(home, id))
	if err != nil || string(data) != "current" {
		t.Fatalf("selected profile content = %q, %v", data, err)
	}
}
