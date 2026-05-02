package notifier_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// startDiscordServer starts a fake Discord webhook server.
// handler receives the decoded payload and controls the response status.
func startDiscordServer(t *testing.T, status int, handler func(body map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("invalid JSON payload: %v", err)
		}
		if handler != nil {
			handler(payload)
		}
		w.WriteHeader(status)
	}))
}

func TestDiscord_Notify_Success(t *testing.T) {
	var received map[string]any
	srv := startDiscordServer(t, http.StatusNoContent, func(body map[string]any) {
		received = body
	})
	defer srv.Close()

	diffs := []types.DiffResult{
		{Kind: types.ChangeKindAdded, Path: "customer.phone", After: "string"},
		{Kind: types.ChangeKindRemoved, Path: "customer.document", Before: "string"},
		{Kind: types.ChangeKindTypeChanged, Path: "amount", Before: "string", After: "number"},
		{Kind: types.ChangeKindStatusChanged, Path: "status", Before: "200", After: "404"},
	}

	d := notifier.NewDiscord(srv.URL)
	if err := d.Notify("payments-api", diffs); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}

	// Validate top-level structure: must have "embeds" array
	embeds, ok := received["embeds"].([]any)
	if !ok || len(embeds) == 0 {
		t.Fatalf("payload missing 'embeds' array, got: %v", received)
	}

	embed := embeds[0].(map[string]any)

	// Title must mention the monitor name
	title, _ := embed["title"].(string)
	if !strings.Contains(title, "payments-api") {
		t.Errorf("embed title should contain monitor name, got: %s", title)
	}

	// Fields: one per diff
	fields, ok := embed["fields"].([]any)
	if !ok {
		t.Fatalf("embed missing 'fields', got: %v", embed)
	}
	if len(fields) != len(diffs) {
		t.Errorf("expected %d fields, got %d", len(diffs), len(fields))
	}

	// Color must be set
	if _, ok := embed["color"]; !ok {
		t.Error("embed missing 'color'")
	}
}

func TestDiscord_Notify_ServerError(t *testing.T) {
	srv := startDiscordServer(t, http.StatusInternalServerError, nil)
	defer srv.Close()

	d := notifier.NewDiscord(srv.URL)
	err := d.Notify("my-api", []types.DiffResult{
		{Kind: types.ChangeKindAdded, Path: "field", After: "string"},
	})
	if err == nil {
		t.Fatal("expected error for non-204 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status code, got: %v", err)
	}
}

func TestDiscord_Notify_ConnectionRefused(t *testing.T) {
	d := notifier.NewDiscord("http://127.0.0.1:1") // nothing listening
	err := d.Notify("my-api", []types.DiffResult{
		{Kind: types.ChangeKindAdded, Path: "field", After: "string"},
	})
	if err == nil {
		t.Fatal("expected error for connection refused, got nil")
	}
}

func TestDiscord_Notify_FieldFormats(t *testing.T) {
	cases := []struct {
		diff      types.DiffResult
		wantTitle string
		wantValue string
	}{
		{
			diff:      types.DiffResult{Kind: types.ChangeKindAdded, Path: "foo", After: "string"},
			wantTitle: "foo",
			wantValue: "after",
		},
		{
			diff:      types.DiffResult{Kind: types.ChangeKindRemoved, Path: "bar", Before: "number"},
			wantTitle: "bar",
			wantValue: "before",
		},
		{
			diff:      types.DiffResult{Kind: types.ChangeKindTypeChanged, Path: "baz", Before: "string", After: "number"},
			wantTitle: "baz",
			wantValue: "before",
		},
		{
			diff:      types.DiffResult{Kind: types.ChangeKindStatusChanged, Path: "status", Before: "200", After: "500"},
			wantTitle: "status",
			wantValue: "200",
		},
	}

	for _, tc := range cases {
		var received map[string]any
		srv := startDiscordServer(t, http.StatusNoContent, func(body map[string]any) {
			received = body
		})

		d := notifier.NewDiscord(srv.URL)
		if err := d.Notify("test", []types.DiffResult{tc.diff}); err != nil {
			t.Errorf("diff %v: unexpected error: %v", tc.diff.Kind, err)
			srv.Close()
			continue
		}

		embeds := received["embeds"].([]any)
		embed := embeds[0].(map[string]any)
		fields := embed["fields"].([]any)
		field := fields[0].(map[string]any)

		fieldName := field["name"].(string)
		fieldValue := field["value"].(string)

		if !strings.Contains(fieldName, tc.wantTitle) {
			t.Errorf("kind=%s: field name should contain %q, got %q", tc.diff.Kind, tc.wantTitle, fieldName)
		}
		if !strings.Contains(fieldValue, tc.wantValue) {
			t.Errorf("kind=%s: field value should contain %q, got %q", tc.diff.Kind, tc.wantValue, fieldValue)
		}

		srv.Close()
	}
}

// TestDiscord_Integration sends a real test message to Discord.
// Only runs when DISCORD_WEBHOOK_URL is set in the environment.
//
//	DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/... go test ./internal/notifier/ -run TestDiscord_Integration -v
func TestDiscord_Integration(t *testing.T) {
	url := os.Getenv("DISCORD_WEBHOOK_URL")
	if url == "" {
		t.Skip("DISCORD_WEBHOOK_URL not set — skipping integration test")
	}

	d := notifier.NewDiscord(url)
	if err := d.Test(); err != nil {
		t.Fatalf("integration test failed: %v", err)
	}
	t.Log("Test message sent successfully — check your Discord channel.")
}
