package main

import (
	"os"
	"path/filepath"
)

// ArkTS Profile.saveByUri and TailscaleProfile persist the per-profile directory.
// Prefer that source of truth; retain flat files for older native clients.
func profileConfigPath(home, id string) string {
	current := filepath.Join(home, "profiles", id, "config.yaml")
	if _, err := os.Stat(current); err == nil || !os.IsNotExist(err) {
		return current
	}
	legacy := filepath.Join(home, "profiles", id+".yaml")
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return current
}
