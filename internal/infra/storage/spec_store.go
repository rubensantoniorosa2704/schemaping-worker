package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

// SpecStore implements domain.Store for OpenAPI monitors by persisting the raw
// spec bytes to disk. It follows the same interface as Store (SnapshotStore) but
// stores the raw response body instead of parsed JSON.
type SpecStore struct{}

// NewSpecStore returns a SpecStore ready to use.
func NewSpecStore() *SpecStore {
	return &SpecStore{}
}

// Save writes the raw spec body to disk, overwriting any previous snapshot for the monitor.
func (s *SpecStore) Save(snap domain.Snapshot) error {
	path, err := specPath(snap.MonitorName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("specstore: create dirs: %w", err)
	}

	if len(snap.RawBody) == 0 {
		return nil
	}

	if err := os.WriteFile(path, snap.RawBody, 0600); err != nil {
		return fmt.Errorf("specstore: write file: %w", err)
	}

	return nil
}

// Load reads the raw spec body for the given monitor from disk and returns a Snapshot
// with RawBody populated. Returns os.ErrNotExist if no snapshot has been saved yet.
func (s *SpecStore) Load(monitorName string) (domain.Snapshot, error) {
	path, err := specPath(monitorName)
	if err != nil {
		return domain.Snapshot{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Snapshot{}, err // preserves os.ErrNotExist
	}

	info, err := os.Stat(path)
	if err != nil {
		return domain.Snapshot{}, err
	}

	return domain.Snapshot{
		MonitorName: monitorName,
		CapturedAt:  info.ModTime().UTC(),
		StatusCode:  200,
		RawBody:     data,
	}, nil
}

func specPath(monitorName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("specstore: resolve home dir: %w", err)
	}
	safe := unsafeChars.ReplaceAllString(monitorName, "_")
	return filepath.Join(home, ".schemaping", "snapshots", safe+".spec"), nil
}
