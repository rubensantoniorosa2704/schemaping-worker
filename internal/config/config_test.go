package config_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/config"
)

// writeTemp writes content to a temp file and returns its path.
// The caller is responsible for removing it.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "schemaping-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// --- file-level errors ---

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "config: read file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// A tab character at the start of a line is illegal in YAML.
	path := writeTemp(t, "\t bad_key: value")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "config: parse yaml") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- monitor validation errors ---

func TestLoad_MissingMonitorName(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - url: https://example.com
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing monitor name, got nil")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLoad_MissingMonitorURL(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - name: my-api
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing monitor URL, got nil")
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Errorf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "my-api") {
		t.Errorf("error should mention the monitor name, got: %v", err)
	}
}

// --- monitor defaults ---

func TestLoad_MonitorDefaults(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - name: minimal
    url: https://example.com/api
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := cfg.Monitors[0]
	if m.Method != "GET" {
		t.Errorf("default Method: want GET, got %s", m.Method)
	}
	if m.ExpectedStatus != 200 {
		t.Errorf("default ExpectedStatus: want 200, got %d", m.ExpectedStatus)
	}
	if m.Timeout != 10*time.Second {
		t.Errorf("default Timeout: want 10s, got %v", m.Timeout)
	}
	if m.Interval != time.Minute {
		t.Errorf("default Interval: want 1m, got %v", m.Interval)
	}
}

func TestLoad_MonitorExplicitValuesNotOverridden(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - name: explicit
    url: https://example.com/api
    method: POST
    expected_status: 201
    timeout: 5s
    interval: 30s
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := cfg.Monitors[0]
	if m.Method != "POST" {
		t.Errorf("Method: want POST, got %s", m.Method)
	}
	if m.ExpectedStatus != 201 {
		t.Errorf("ExpectedStatus: want 201, got %d", m.ExpectedStatus)
	}
	if m.Timeout != 5*time.Second {
		t.Errorf("Timeout: want 5s, got %v", m.Timeout)
	}
	if m.Interval != 30*time.Second {
		t.Errorf("Interval: want 30s, got %v", m.Interval)
	}
}

func TestLoad_MonitorHeaders(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - name: auth-api
    url: https://example.com/api
    headers:
      Authorization: Bearer token123
      X-Custom: value
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h := cfg.Monitors[0].Headers
	if h["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization header: got %q", h["Authorization"])
	}
	if h["X-Custom"] != "value" {
		t.Errorf("X-Custom header: got %q", h["X-Custom"])
	}
}

// --- multiple monitors ---

func TestLoad_MultipleMonitors(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - name: api-one
    url: https://one.example.com
  - name: api-two
    url: https://two.example.com
    method: POST
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Monitors) != 2 {
		t.Fatalf("expected 2 monitors, got %d", len(cfg.Monitors))
	}
	if cfg.Monitors[0].Name != "api-one" {
		t.Errorf("monitor[0].Name: got %s", cfg.Monitors[0].Name)
	}
	if cfg.Monitors[1].Method != "POST" {
		t.Errorf("monitor[1].Method: got %s", cfg.Monitors[1].Method)
	}
	// defaults still applied to monitor[0]
	if cfg.Monitors[0].Method != "GET" {
		t.Errorf("monitor[0] default Method: got %s", cfg.Monitors[0].Method)
	}
}

// --- env expansion ---

func TestLoad_EnvExpansionInMonitorURL(t *testing.T) {
	t.Setenv("API_HOST", "https://api.example.com")
	path := writeTemp(t, `
monitors:
  - name: env-api
    url: ${API_HOST}/v1/resource
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "https://api.example.com/v1/resource"
	if cfg.Monitors[0].URL != want {
		t.Errorf("URL env expansion: want %s, got %s", want, cfg.Monitors[0].URL)
	}
}

func TestLoad_EnvExpansionInHeaders(t *testing.T) {
	t.Setenv("API_TOKEN", "secret-token")
	path := writeTemp(t, `
monitors:
  - name: auth-api
    url: https://example.com
    headers:
      Authorization: Bearer ${API_TOKEN}
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := cfg.Monitors[0].Headers["Authorization"]
	if got != "Bearer secret-token" {
		t.Errorf("header env expansion: got %q", got)
	}
}

func TestLoad_UnsetEnvVarExpandsToEmpty(t *testing.T) {
	os.Unsetenv("UNSET_VAR")
	path := writeTemp(t, `
monitors:
  - name: test
    url: https://example.com/${UNSET_VAR}/path
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// os.ExpandEnv replaces unset vars with empty string
	if cfg.Monitors[0].URL != "https://example.com//path" {
		t.Errorf("unset var should expand to empty, got: %s", cfg.Monitors[0].URL)
	}
}

// --- webhooks ---

func TestLoad_NoWebhooks(t *testing.T) {
	path := writeTemp(t, `
monitors:
  - name: api
    url: https://example.com
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Webhooks) != 0 {
		t.Errorf("expected 0 webhooks, got %d", len(cfg.Webhooks))
	}
}

func TestLoad_WebhookEnvExpansion(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123/abc")
	t.Setenv("TELEGRAM_BOT_TOKEN", "mytoken123")
	t.Setenv("TELEGRAM_CHAT_ID", "456789")

	path := writeTemp(t, `
monitors:
  - name: api
    url: https://example.com

webhooks:
  - type: discord
    url: ${DISCORD_WEBHOOK_URL}
  - type: telegram
    url: https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage
    chat_id: ${TELEGRAM_CHAT_ID}
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Webhooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(cfg.Webhooks))
	}

	d := cfg.Webhooks[0]
	if d.Type != "discord" {
		t.Errorf("webhook[0].Type: want discord, got %s", d.Type)
	}
	if d.URL != "https://discord.com/api/webhooks/123/abc" {
		t.Errorf("webhook[0].URL not expanded: %s", d.URL)
	}

	tg := cfg.Webhooks[1]
	if tg.Type != "telegram" {
		t.Errorf("webhook[1].Type: want telegram, got %s", tg.Type)
	}
	if tg.URL != "https://api.telegram.org/botmytoken123/sendMessage" {
		t.Errorf("webhook[1].URL not expanded: %s", tg.URL)
	}
	if tg.ChatID != "456789" {
		t.Errorf("webhook[1].ChatID not expanded: %s", tg.ChatID)
	}
}
