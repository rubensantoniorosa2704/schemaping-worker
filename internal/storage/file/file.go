package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

func snapshotPath(monitorName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("storage/file: resolve home dir: %w", err)
	}
	safe := unsafeChars.ReplaceAllString(monitorName, "_")
	return filepath.Join(home, ".schemaping", "snapshots", safe+".json"), nil
}

// Save writes the snapshot to disk, overwriting any previous snapshot for the same monitor.
func Save(snap types.Snapshot) error {
	path, err := snapshotPath(snap.MonitorName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("storage/file: create dirs: %w", err)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("storage/file: marshal snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("storage/file: write file: %w", err)
	}

	return nil
}

// Load reads the latest snapshot for the given monitor from disk.
// Returns os.ErrNotExist if no snapshot has been saved yet.
func Load(monitorName string) (types.Snapshot, error) {
	path, err := snapshotPath(monitorName)
	if err != nil {
		return types.Snapshot{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return types.Snapshot{}, err // preserves os.ErrNotExist for callers to check
	}

	var snap types.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return types.Snapshot{}, fmt.Errorf("storage/file: unmarshal snapshot: %w", err)
	}

	return snap, nil
}
