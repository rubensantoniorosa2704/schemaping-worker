package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// discordColor maps change kinds to Discord embed sidebar colors (decimal RGB).
const (
	discordColorYellow = 16776960 // added / type changed
	discordColorRed    = 15158332 // removed
	discordColorBlue   = 3447003  // status changed / nullability
)

// Discord sends schema-change alerts to a Discord channel via an Incoming Webhook.
// Reference: https://discord.com/developers/docs/resources/webhook#execute-webhook
type Discord struct {
	url string
}

// NewDiscord creates a Discord notifier for the given webhook URL.
func NewDiscord(url string) *Discord {
	return &Discord{url: url}
}

// discordEmbed is the subset of the Discord embed object used for alerts.
type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Fields      []discordField `json:"fields,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// Test sends a test message to verify the webhook is reachable and correctly configured.
func (d *Discord) Test() error {
	return d.Notify("schemaping-test", []types.DiffResult{
		{Kind: types.ChangeKindAdded, Path: "example.field", After: "string"},
	})
}

// Notify posts a Discord embed listing all detected diffs for the monitor.
func (d *Discord) Notify(monitorName string, diffs []types.DiffResult) error {
	fields := make([]discordField, 0, len(diffs))
	for _, diff := range diffs {
		fields = append(fields, discordField{
			Name:   formatDiffTitle(diff),
			Value:  formatDiffValue(diff),
			Inline: false,
		})
	}

	embed := discordEmbed{
		Title:       fmt.Sprintf("⚠️ Schema change detected: %s", monitorName),
		Description: fmt.Sprintf("%d change(s) detected", len(diffs)),
		Color:       discordColorYellow,
		Fields:      fields,
	}

	payload, err := json.Marshal(map[string]any{"embeds": []discordEmbed{embed}})
	if err != nil {
		return fmt.Errorf("discord: marshal payload: %w", err)
	}

	resp, err := http.Post(d.url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("discord: http post: %w", err)
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success.
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func formatDiffTitle(d types.DiffResult) string {
	switch d.Kind {
	case types.ChangeKindAdded:
		return fmt.Sprintf("➕ `%s` added", d.Path)
	case types.ChangeKindRemoved:
		return fmt.Sprintf("➖ `%s` removed", d.Path)
	case types.ChangeKindTypeChanged:
		return fmt.Sprintf("🔄 `%s` type changed", d.Path)
	case types.ChangeKindNullabilityChanged:
		return fmt.Sprintf("🔄 `%s` nullability changed", d.Path)
	case types.ChangeKindStatusChanged:
		return "🔄 HTTP status changed"
	default:
		return fmt.Sprintf("`%s` changed", d.Path)
	}
}

func formatDiffValue(d types.DiffResult) string {
	var parts []string
	if d.Before != "" {
		parts = append(parts, fmt.Sprintf("before: `%s`", d.Before))
	}
	if d.After != "" {
		parts = append(parts, fmt.Sprintf("after: `%s`", d.After))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, " → ")
}
