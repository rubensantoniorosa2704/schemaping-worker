package notifier_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/notifier"
)

// startTelegramServer starts a fake Telegram Bot API server.
// okResponse controls whether the response body has ok=true or ok=false.
func startTelegramServer(t *testing.T, statusCode int, okResponse bool, handler func(body map[string]any)) *httptest.Server {
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if okResponse {
			w.Write([]byte(`{"ok":true}`))
		} else {
			w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
		}
	}))
}

func TestTelegram_Notify_Success(t *testing.T) {
	var received map[string]any
	srv := startTelegramServer(t, http.StatusOK, true, func(body map[string]any) {
		received = body
	})
	defer srv.Close()

	diffs := []domain.DiffResult{
		{Kind: domain.ChangeKindAdded, Path: "customer.phone", After: "string"},
		{Kind: domain.ChangeKindRemoved, Path: "customer.document", Before: "string"},
		{Kind: domain.ChangeKindTypeChanged, Path: "amount", Before: "string", After: "number"},
		{Kind: domain.ChangeKindStatusChanged, Path: "status", Before: "200", After: "404"},
	}

	tg := notifier.NewTelegram(srv.URL, "123456")
	if err := tg.Notify("payments-api", diffs); err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}

	// chat_id must match
	if received["chat_id"] != "123456" {
		t.Errorf("chat_id: want 123456, got %v", received["chat_id"])
	}

	// parse_mode must be HTML
	if received["parse_mode"] != "HTML" {
		t.Errorf("parse_mode: want HTML, got %v", received["parse_mode"])
	}

	// text must contain monitor name and all diff paths
	text, _ := received["text"].(string)
	if !strings.Contains(text, "payments-api") {
		t.Errorf("text should contain monitor name, got: %s", text)
	}
	for _, d := range diffs {
		if d.Path != "status" && !strings.Contains(text, d.Path) {
			t.Errorf("text should contain path %q, got: %s", d.Path, text)
		}
	}
}

func TestTelegram_Notify_APIReturnsFalse(t *testing.T) {
	srv := startTelegramServer(t, http.StatusUnauthorized, false, nil)
	defer srv.Close()

	tg := notifier.NewTelegram(srv.URL, "123456")
	err := tg.Notify("my-api", []domain.DiffResult{
		{Kind: domain.ChangeKindAdded, Path: "field", After: "string"},
	})
	if err == nil {
		t.Fatal("expected error when ok=false, got nil")
	}
	if !strings.Contains(err.Error(), "ok=false") {
		t.Errorf("error should mention ok=false, got: %v", err)
	}
}

func TestTelegram_Notify_ConnectionRefused(t *testing.T) {
	tg := notifier.NewTelegram("http://127.0.0.1:1", "123456")
	err := tg.Notify("my-api", []domain.DiffResult{
		{Kind: domain.ChangeKindAdded, Path: "field", After: "string"},
	})
	if err == nil {
		t.Fatal("expected error for connection refused, got nil")
	}
}

func TestTelegram_Notify_MessageFormats(t *testing.T) {
	cases := []struct {
		diff     domain.DiffResult
		wantText string
	}{
		{
			diff:     domain.DiffResult{Kind: domain.ChangeKindAdded, Path: "foo", After: "string"},
			wantText: "foo",
		},
		{
			diff:     domain.DiffResult{Kind: domain.ChangeKindRemoved, Path: "bar", Before: "number"},
			wantText: "bar",
		},
		{
			diff:     domain.DiffResult{Kind: domain.ChangeKindTypeChanged, Path: "baz", Before: "string", After: "number"},
			wantText: "string",
		},
		{
			diff:     domain.DiffResult{Kind: domain.ChangeKindStatusChanged, Before: "200", After: "500"},
			wantText: "200",
		},
	}

	for _, tc := range cases {
		var received map[string]any
		srv := startTelegramServer(t, http.StatusOK, true, func(body map[string]any) {
			received = body
		})

		tg := notifier.NewTelegram(srv.URL, "123")
		if err := tg.Notify("test", []domain.DiffResult{tc.diff}); err != nil {
			t.Errorf("diff %v: unexpected error: %v", tc.diff.Kind, err)
			srv.Close()
			continue
		}

		text, _ := received["text"].(string)
		if !strings.Contains(text, tc.wantText) {
			t.Errorf("kind=%s: text should contain %q, got: %s", tc.diff.Kind, tc.wantText, text)
		}

		srv.Close()
	}
}

// TestTelegram_Integration sends a real message via the Telegram Bot API.
// Only runs when TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are set.
//
//	TELEGRAM_BOT_TOKEN=... TELEGRAM_CHAT_ID=... go test ./internal/notifier/ -run TestTelegram_Integration -v
func TestTelegram_Integration(t *testing.T) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		t.Skip("TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID not set — skipping integration test")
	}

	url := "https://api.telegram.org/bot" + token + "/sendMessage"
	tg := notifier.NewTelegram(url, chatID)
	if err := tg.Test(); err != nil {
		t.Fatalf("integration test failed: %v", err)
	}
	t.Log("Test message sent successfully — check your Telegram chat.")
}
