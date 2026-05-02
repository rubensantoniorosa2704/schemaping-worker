package types

import "time"

// WebhookConfig holds the configuration for a single notification webhook.
// The URL and other sensitive fields support ${ENV_VAR} expansion.
type WebhookConfig struct {
	Type   string `yaml:"type"`    // "discord" or "telegram"
	URL    string `yaml:"url"`     // webhook URL (Discord) or bot API URL (Telegram)
	ChatID string `yaml:"chat_id"` // required for Telegram
}

// Monitor represents a configured endpoint to be monitored.
type Monitor struct {
	Name           string            `yaml:"name"`
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	Interval       time.Duration     `yaml:"interval"`
	Timeout        time.Duration     `yaml:"timeout"`
	ExpectedStatus int               `yaml:"expected_status"`
	Headers        map[string]string `yaml:"headers"`
	// Webhooks overrides the global webhook list for this monitor.
	// If nil, global webhooks are used. If empty slice, notifications are silenced.
	Webhooks []WebhookConfig `yaml:"webhooks"`
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
