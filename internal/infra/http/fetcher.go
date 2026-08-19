// Package http implements the domain.Fetcher port using HTTP requests.
package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

// Fetcher implements domain.Fetcher by executing HTTP requests.
type Fetcher struct {
	client *http.Client
}

// New returns a Fetcher configured with the given monitor's timeout.
func New(m domain.Monitor) *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: m.Timeout},
	}
}

// Fetch executes an HTTP request for the given monitor and returns a Snapshot.
// All errors are captured inside Snapshot.Error.
func (f *Fetcher) Fetch(m domain.Monitor) domain.Snapshot {
	snap := domain.Snapshot{
		MonitorName: m.Name,
		CapturedAt:  time.Now().UTC(),
	}

	req, err := http.NewRequest(m.Method, m.URL, nil)
	if err != nil {
		snap.Error = fmt.Sprintf("httpclient: build request: %s", err)
		snap.TransportErr = err
		return snap
	}

	for k, v := range m.Headers {
		req.Header.Set(k, v)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		snap.Error = fmt.Sprintf("httpclient: execute request: %s", err)
		snap.TransportErr = err
		return snap
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		snap.Error = fmt.Sprintf("httpclient: read body: %s", err)
		snap.TransportErr = err
		return snap
	}

	snap.StatusCode = resp.StatusCode

	if resp.StatusCode != m.ExpectedStatus {
		snap.Error = fmt.Sprintf("unexpected status: got %d, want %d", resp.StatusCode, m.ExpectedStatus)
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &snap.Body); err != nil && snap.Error == "" {
			snap.Error = fmt.Sprintf("parse body: %s", err)
		}
	}

	return snap
}
