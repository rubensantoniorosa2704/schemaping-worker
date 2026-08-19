// Package reporter implements the domain.Reporter port using HTTP POST.
package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

// HTTP implements domain.Reporter by sending a JSON POST to the configured endpoint.
// It is fire-and-forget: a 5-second timeout ensures it never blocks the check loop.
type HTTP struct {
	url    string
	apiKey string
	client *http.Client
}

// New returns an HTTP Reporter for the given config.
func New(cfg domain.ReportConfig) *HTTP {
	return &HTTP{
		url:    cfg.URL,
		apiKey: cfg.APIKey,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Report sends the payload as JSON to the configured endpoint.
func (r *HTTP) Report(payload domain.ReportPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("reporter: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("reporter: build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("reporter: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("reporter: unexpected status %d", resp.StatusCode)
	}

	return nil
}
