package checker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/httpclient"
	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// Checker runs checks for a single monitor, reusing the same HTTP client across calls.
type Checker struct {
	monitor types.Monitor
	client  *http.Client
}

// New creates a Checker for the given monitor.
func New(m types.Monitor) *Checker {
	return &Checker{monitor: m, client: httpclient.New(m)}
}

// Run executes a check and returns a Snapshot.
// Failures are captured in Snapshot.Error; this function never returns an error.
func (c *Checker) Run() types.Snapshot {
	snap := types.Snapshot{
		MonitorName: c.monitor.Name,
		CapturedAt:  time.Now().UTC(),
	}

	statusCode, body, err := httpclient.Do(c.client, c.monitor)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}

	snap.StatusCode = statusCode

	if statusCode != c.monitor.ExpectedStatus {
		snap.Error = fmt.Sprintf("unexpected status: got %d, want %d", statusCode, c.monitor.ExpectedStatus)
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &snap.Body); err != nil && snap.Error == "" {
			snap.Error = fmt.Sprintf("parse body: %s", err.Error())
		}
	}

	return snap
}
