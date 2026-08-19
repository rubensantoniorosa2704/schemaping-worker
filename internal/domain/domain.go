// Package domain contains the core types and port interfaces for SchemaPing.
// No infrastructure dependencies are allowed in this package.
package domain

import "time"

// WebhookConfig holds the configuration for a single notification webhook.
// The URL and other sensitive fields support ${ENV_VAR} expansion.
type WebhookConfig struct {
	Type   string `yaml:"type"`    // "discord" or "telegram"
	URL    string `yaml:"url"`     // webhook URL (Discord) or bot API URL (Telegram)
	ChatID string `yaml:"chat_id"` // required for Telegram
}

// MonitorType identifies what kind of content the monitor is tracking.
type MonitorType string

const (
	// MonitorTypeHTTP is the default type: monitors a JSON API endpoint and
	// compares response structure over time.
	MonitorTypeHTTP MonitorType = "http"
	// MonitorTypeOpenAPI monitors an OpenAPI 3.0 specification file and
	// detects structural drift in paths, schemas, and parameters.
	MonitorTypeOpenAPI MonitorType = "openapi"
)

// Monitor represents a configured endpoint to be monitored.
type Monitor struct {
	Name           string            `yaml:"name"`
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	Interval       time.Duration     `yaml:"interval"`
	Timeout        time.Duration     `yaml:"timeout"`
	ExpectedStatus int               `yaml:"expected_status"`
	Headers        map[string]string `yaml:"headers"`
	// Type determines the comparison strategy: "http" (default) or "openapi".
	Type MonitorType `yaml:"type"`
	// Webhooks overrides the global webhook list for this monitor.
	// If nil, global webhooks are used. If empty slice, notifications are silenced.
	Webhooks []WebhookConfig `yaml:"webhooks"`
	// Retries is the number of additional attempts after the first transient failure.
	// nil means "not configured" and config.Load will apply the default (3).
	// A pointer to 0 means retries are explicitly disabled.
	Retries *int `yaml:"retries"`
	// RetryBackoff is the base interval for exponential backoff between attempts.
	// Zero value means "not configured" and config.Load will apply the default (2s).
	// The actual wait before attempt n is: RetryBackoff * 2^(n-1), capped at 30s.
	RetryBackoff time.Duration `yaml:"retry_backoff"`
	// Raw enables saving the raw response body to disk after each successful check.
	// Files are stored under ~/.schemaping/raw/<monitor-name>/ and overwritten on each interval.
	// Defaults to false (opt-in).
	Raw bool `yaml:"raw"`
}

// Snapshot represents a captured response from a monitor at a point in time.
type Snapshot struct {
	MonitorName string         `json:"monitor_name"`
	CapturedAt  time.Time      `json:"captured_at"`
	StatusCode  int            `json:"status_code"`
	Body        map[string]any `json:"body"`
	Error       string         `json:"error,omitempty"`
	// TransportErr holds the raw transport error (timeout, connection refused, etc.)
	// used internally by the retry logic. Not persisted to disk.
	TransportErr error `json:"-"`
	// RawBody holds the raw response bytes as received from the network.
	// Not persisted in the schema snapshot — used by the raw storage layer
	// when the monitor has raw: true enabled.
	RawBody []byte `json:"-"`
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
