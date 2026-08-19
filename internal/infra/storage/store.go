// Package storage implements the domain.Store port using the local filesystem.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

// Store implements domain.Store by persisting snapshots as JSON files under
// ~/.schemaping/snapshots/.
type Store struct{}

// New returns a Store ready to use.
func New() *Store {
	return &Store{}
}

func snapshotPath(monitorName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("storage: resolve home dir: %w", err)
	}
	safe := unsafeChars.ReplaceAllString(monitorName, "_")
	return filepath.Join(home, ".schemaping", "snapshots", safe+".json"), nil
}

// Save writes the snapshot to disk, overwriting any previous snapshot for the same monitor.
func (s *Store) Save(snap domain.Snapshot) error {
	path, err := snapshotPath(snap.MonitorName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("storage: create dirs: %w", err)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: marshal snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("storage: write file: %w", err)
	}

	return nil
}

// Load reads the latest snapshot for the given monitor from disk.
// Returns os.ErrNotExist if no snapshot has been saved yet.
func (s *Store) Load(monitorName string) (domain.Snapshot, error) {
	path, err := snapshotPath(monitorName)
	if err != nil {
		return domain.Snapshot{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Snapshot{}, err // preserves os.ErrNotExist for callers
	}

	var snap domain.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return domain.Snapshot{}, fmt.Errorf("storage: unmarshal snapshot: %w", err)
	}

	return snap, nil
}
