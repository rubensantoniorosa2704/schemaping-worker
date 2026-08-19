package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RawStore implements domain.RawStore by persisting raw response bodies as files
// under ~/.schemaping/raw/<monitor-name>/. Only one file is kept per monitor —
// the previous file is removed before writing the new one.
type RawStore struct{}

// NewRawStore returns a RawStore ready to use.
func NewRawStore() *RawStore {
	return &RawStore{}
}

// Save writes the raw response body to disk, replacing any previous file for the monitor.
// The filename includes the capture timestamp for identification:
//
//	<monitor-name>-2026-08-19T08-45-00Z.json
func (s *RawStore) Save(monitorName string, capturedAt time.Time, data []byte) error {
	dir, err := rawDir(monitorName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("raw: create dirs: %w", err)
	}

	// Remove all existing files in the directory (single-file guarantee).
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("raw: read dir: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}

	ts := capturedAt.UTC().Format("2006-01-02T15-04-05Z")
	safeName := unsafeChars.ReplaceAllString(monitorName, "_")
	filename := fmt.Sprintf("%s-%s.json", safeName, ts)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("raw: write file: %w", err)
	}

	return nil
}

func rawDir(monitorName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("raw: resolve home dir: %w", err)
	}
	safe := unsafeChars.ReplaceAllString(monitorName, "_")
	return filepath.Join(home, ".schemaping", "raw", safe), nil
}
