package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRawStore_Save_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	store := NewRawStore()
	ts := time.Date(2026, 8, 19, 8, 45, 0, 0, time.UTC)
	data := []byte(`{"id": 1, "name": "test"}`)

	err := store.Save("payments-api", ts, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDir := filepath.Join(dir, ".schemaping", "raw", "payments-api")
	expectedFile := filepath.Join(expectedDir, "payments-api-2026-08-19T08-45-00Z.json")

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	if string(content) != string(data) {
		t.Errorf("content mismatch:\ngot:  %s\nwant: %s", content, data)
	}
}

func TestRawStore_Save_OverwritesPreviousFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	store := NewRawStore()

	// First save
	ts1 := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	err := store.Save("my-api", ts1, []byte(`{"v": 1}`))
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Second save — should remove the first file
	ts2 := time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)
	err = store.Save("my-api", ts2, []byte(`{"v": 2}`))
	if err != nil {
		t.Fatalf("second save: %v", err)
	}

	rawDir := filepath.Join(dir, ".schemaping", "raw", "my-api")
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	if entries[0].Name() != "my-api-2026-08-19T09-00-00Z.json" {
		t.Errorf("unexpected filename: %s", entries[0].Name())
	}

	content, _ := os.ReadFile(filepath.Join(rawDir, entries[0].Name()))
	if string(content) != `{"v": 2}` {
		t.Errorf("content mismatch: got %s", content)
	}
}

func TestRawStore_Save_YAMLExtension(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	store := NewRawStore()
	ts := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	data := []byte("openapi: 3.0.0\ninfo:\n  title: Test\n")

	err := store.Save("my-spec", ts, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDir := filepath.Join(dir, ".schemaping", "raw", "my-spec")
	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	if entries[0].Name() != "my-spec-2026-08-19T10-00-00Z.yaml" {
		t.Errorf("expected .yaml extension, got: %s", entries[0].Name())
	}
}

func TestRawStore_Save_SanitizesMonitorName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	store := NewRawStore()
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	err := store.Save("weird/name with spaces!", ts, []byte("ok"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDir := filepath.Join(dir, ".schemaping", "raw", "weird_name_with_spaces_")
	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		t.Fatalf("expected sanitized dir to exist: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
}
