package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/storage/sqlite"
)

// TestDaemonConfigPrefixFallback tests that daemon reads auto-commit/auto-push
// config from both sync.* (documented) and daemon.* (legacy) prefixes.
// This is a regression test for the bug where users set sync.auto_commit=true
// but daemon only read from daemon.auto_commit.
func TestDaemonConfigPrefixFallback(t *testing.T) {
	tests := []struct {
		name           string
		configKey      string
		configValue    string
		expectEnabled  bool
		description    string
	}{
		{
			name:          "sync.auto_commit enabled",
			configKey:     "sync.auto_commit",
			configValue:   "true",
			expectEnabled: true,
			description:   "daemon should read sync.auto_commit (documented prefix)",
		},
		{
			name:          "daemon.auto_commit enabled",
			configKey:     "daemon.auto_commit",
			configValue:   "true",
			expectEnabled: true,
			description:   "daemon should read daemon.auto_commit (legacy prefix)",
		},
		{
			name:          "sync.auto_push enabled",
			configKey:     "sync.auto_push",
			configValue:   "true",
			expectEnabled: true,
			description:   "daemon should read sync.auto_push (documented prefix)",
		},
		{
			name:          "daemon.auto_push enabled",
			configKey:     "daemon.auto_push",
			configValue:   "true",
			expectEnabled: true,
			description:   "daemon should read daemon.auto_push (legacy prefix)",
		},
		{
			name:          "sync.auto_commit disabled",
			configKey:     "sync.auto_commit",
			configValue:   "false",
			expectEnabled: false,
			description:   "daemon should respect sync.auto_commit=false",
		},
		{
			name:          "no config set",
			configKey:     "",
			configValue:   "",
			expectEnabled: false,
			description:   "daemon should default to false when no config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with .beads structure
			tmpDir := t.TempDir()
			beadsDir := filepath.Join(tmpDir, ".beads")
			if err := os.MkdirAll(beadsDir, 0755); err != nil {
				t.Fatalf("Failed to create beads dir: %v", err)
			}

			testDBPath := filepath.Join(beadsDir, "beads.db")
			ctx := context.Background()

			// Create database and set config
			store, err := sqlite.New(ctx, testDBPath)
			if err != nil {
				t.Fatalf("Failed to create store: %v", err)
			}

			if tt.configKey != "" {
				if err := store.SetConfig(ctx, tt.configKey, tt.configValue); err != nil {
					t.Fatalf("Failed to set config %s=%s: %v", tt.configKey, tt.configValue, err)
				}
			}
			store.Close()

			// Test the helper function
			result := readDaemonAutoConfigFromDB(testDBPath, "auto_commit")

			// Determine expected result based on config key
			var expected bool
			if tt.configKey == "sync.auto_commit" || tt.configKey == "daemon.auto_commit" {
				expected = tt.expectEnabled
			}

			if result != expected {
				t.Errorf("readDaemonAutoConfigFromDB(%q, 'auto_commit') = %v, want %v (%s)",
					testDBPath, result, expected, tt.description)
			}
		})
	}
}

// TestDaemonConfigSyncPrefixPriority tests that sync.* prefix takes priority
// over daemon.* prefix when both are set (sync.* is the documented approach).
func TestDaemonConfigSyncPrefixPriority(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	testDBPath := filepath.Join(beadsDir, "beads.db")
	ctx := context.Background()

	store, err := sqlite.New(ctx, testDBPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Set both prefixes with different values - sync.* should win
	if err := store.SetConfig(ctx, "daemon.auto_commit", "false"); err != nil {
		t.Fatalf("Failed to set daemon.auto_commit: %v", err)
	}
	if err := store.SetConfig(ctx, "sync.auto_commit", "true"); err != nil {
		t.Fatalf("Failed to set sync.auto_commit: %v", err)
	}
	store.Close()

	// sync.* should take priority
	result := readDaemonAutoConfigFromDB(testDBPath, "auto_commit")
	if !result {
		t.Errorf("readDaemonAutoConfigFromDB() = false, want true (sync.* should take priority over daemon.*)")
	}
}

// TestDaemonConfigAutoPushFallback tests auto_push config reading
func TestDaemonConfigAutoPushFallback(t *testing.T) {
	tests := []struct {
		name          string
		configKey     string
		configValue   string
		expectEnabled bool
	}{
		{
			name:          "sync.auto_push enabled",
			configKey:     "sync.auto_push",
			configValue:   "true",
			expectEnabled: true,
		},
		{
			name:          "daemon.auto_push enabled",
			configKey:     "daemon.auto_push",
			configValue:   "true",
			expectEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			beadsDir := filepath.Join(tmpDir, ".beads")
			if err := os.MkdirAll(beadsDir, 0755); err != nil {
				t.Fatalf("Failed to create beads dir: %v", err)
			}

			testDBPath := filepath.Join(beadsDir, "beads.db")
			ctx := context.Background()

			store, err := sqlite.New(ctx, testDBPath)
			if err != nil {
				t.Fatalf("Failed to create store: %v", err)
			}

			if err := store.SetConfig(ctx, tt.configKey, tt.configValue); err != nil {
				t.Fatalf("Failed to set config: %v", err)
			}
			store.Close()

			result := readDaemonAutoConfigFromDB(testDBPath, "auto_push")
			if result != tt.expectEnabled {
				t.Errorf("readDaemonAutoConfigFromDB(%q, 'auto_push') = %v, want %v",
					testDBPath, result, tt.expectEnabled)
			}
		})
	}
}
