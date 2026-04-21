package types

import "time"

// Monitor represents a configured endpoint to be monitored.
type Monitor struct {
	Name           string            `yaml:"name"`
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	Interval       time.Duration     `yaml:"interval"`
	Timeout        time.Duration     `yaml:"timeout"`
	ExpectedStatus int               `yaml:"expected_status"`
	Headers        map[string]string `yaml:"headers"`
}

// Snapshot represents a captured response from a monitor at a point in time.
type Snapshot struct {
	MonitorName string         `json:"monitor_name"`
	CapturedAt  time.Time      `json:"captured_at"`
	StatusCode  int            `json:"status_code"`
	Body        map[string]any `json:"body"`
	Error       string         `json:"error,omitempty"`
}

// ChangeKind describes the type of schema change detected.
type ChangeKind string

const (
	ChangeKindAdded              ChangeKind = "added"
	ChangeKindRemoved            ChangeKind = "removed"
	ChangeKindTypeChanged        ChangeKind = "type_changed"
	ChangeKindNullabilityChanged ChangeKind = "nullability_changed"
	ChangeKindStatusChanged      ChangeKind = "status_changed"
)

// DiffResult represents a single detected change between two snapshots.
type DiffResult struct {
	Kind   ChangeKind `json:"kind"`
	Path   string     `json:"path"`
	Before string     `json:"before"`
	After  string     `json:"after"`
}
