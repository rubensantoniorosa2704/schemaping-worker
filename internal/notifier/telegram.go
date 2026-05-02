package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// Telegram sends schema-change alerts via the Telegram Bot API sendMessage method.
// Reference: https://core.telegram.org/bots/api#sendmessage
type Telegram struct {
	url    string // full API URL: https://api.telegram.org/bot<TOKEN>/sendMessage
	chatID string
}

// NewTelegram creates a Telegram notifier.
func NewTelegram(url, chatID string) *Telegram {
	return &Telegram{url: url, chatID: chatID}
}

type telegramPayload struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK bool `json:"ok"`
}

// Notify sends a message listing all detected diffs for the monitor.
func (t *Telegram) Notify(monitorName string, diffs []types.DiffResult) error {
	text := formatTelegramMessage(monitorName, diffs)

	payload, err := json.Marshal(telegramPayload{
		ChatID:    t.chatID,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	resp, err := http.Post(t.url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram: http post: %w", err)
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("telegram: decode response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram: API returned ok=false (status %d)", resp.StatusCode)
	}
	return nil
}

// Test sends a test message to verify the bot and chat ID are correctly configured.
func (t *Telegram) Test() error {
	return t.Notify("schemaping-test", []types.DiffResult{
		{Kind: types.ChangeKindAdded, Path: "example.field", After: "string"},
	})
}

func formatTelegramMessage(monitorName string, diffs []types.DiffResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "⚠️ <b>Schema change detected: %s</b>\n", monitorName)
	fmt.Fprintf(&b, "%d change(s):\n\n", len(diffs))
	for _, d := range diffs {
		b.WriteString(formatTelegramDiff(d))
		b.WriteByte('\n')
	}
	return b.String()
}

func formatTelegramDiff(d types.DiffResult) string {
	switch d.Kind {
	case types.ChangeKindAdded:
		return fmt.Sprintf("➕ <code>%s</code> added (<i>%s</i>)", d.Path, d.After)
	case types.ChangeKindRemoved:
		return fmt.Sprintf("➖ <code>%s</code> removed (<i>%s</i>)", d.Path, d.Before)
	case types.ChangeKindTypeChanged:
		return fmt.Sprintf("🔄 <code>%s</code> type changed: <i>%s</i> → <i>%s</i>", d.Path, d.Before, d.After)
	case types.ChangeKindNullabilityChanged:
		return fmt.Sprintf("🔄 <code>%s</code> nullability changed: <i>%s</i> → <i>%s</i>", d.Path, d.Before, d.After)
	case types.ChangeKindStatusChanged:
		return fmt.Sprintf("🔄 HTTP status changed: <i>%s</i> → <i>%s</i>", d.Before, d.After)
	default:
		return fmt.Sprintf("<code>%s</code> changed", d.Path)
	}
}
