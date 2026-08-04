// Package checker orchestrates a single monitor check, producing a Snapshot.
// It handles retry logic with exponential backoff for transient failures.
package checker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/httpclient"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/retry"
	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// RetryEvent carries information about a failed attempt, delivered to the
// onRetry callback before the checker sleeps and retries.
type RetryEvent struct {
	Attempt     int           // number of the attempt that failed (starts at 1)
	MaxAttempts int           // total configured attempts (retries + 1)
	Reason      string        // error message from the failed attempt
	NextIn      time.Duration // wait interval before next attempt
}

// Checker runs checks for a single monitor, reusing the same HTTP client across calls.
type Checker struct {
	monitor   types.Monitor
	client    *http.Client
	onRetry   func(RetryEvent)
	doRequest func(client *http.Client, m types.Monitor) (int, []byte, error)
}

// New creates a Checker for the given monitor. onRetry is called before each
// retry sleep with details about the failed attempt; it may be nil.
func New(m types.Monitor, onRetry func(RetryEvent)) *Checker {
	return &Checker{
		monitor:   m,
		client:    httpclient.New(m),
		onRetry:   onRetry,
		doRequest: httpclient.Do,
	}
}

// runOnce executes a single HTTP request and returns the resulting Snapshot
// along with the raw transport error (needed by retry.IsRetryable).
// A non-empty Snapshot.Error indicates a failure, but the transport error may
// still be nil (e.g. unexpected status code or JSON parse failure).
func (c *Checker) runOnce() (types.Snapshot, error) {
	snap := types.Snapshot{
		MonitorName: c.monitor.Name,
		CapturedAt:  time.Now().UTC(),
	}

	statusCode, body, transportErr := c.doRequest(c.client, c.monitor)
	if transportErr != nil {
		snap.Error = transportErr.Error()
		return snap, transportErr
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

	return snap, nil
}

// Run executes the check with retry logic and returns a single final Snapshot.
// Failures are captured in Snapshot.Error; this function never returns an error.
// The retry loop honours m.Retries (default 3 additional attempts) and
// m.RetryBackoff (default 2s base) set by config.Load.
func (c *Checker) Run() types.Snapshot {
	// Determine total attempts: retries + 1 original. A nil Retries means the
	// config default of 3 was not yet applied — treat conservatively as 3 retries
	// (4 total), though in practice config.Load always sets a non-nil value.
	maxAttempts := 4
	if c.monitor.Retries != nil {
		maxAttempts = *c.monitor.Retries + 1
	}

	var lastSnap types.Snapshot

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		snap, transportErr := c.runOnce()
		lastSnap = snap

		// Success — return immediately without any retry.
		if snap.Error == "" {
			return snap
		}

		// Non-retryable failure — return immediately (e.g. 404, parse error).
		if !retry.IsRetryable(snap.StatusCode, transportErr) {
			return snap
		}

		// Exhausted all attempts — return the last failure.
		if attempt == maxAttempts {
			return snap
		}

		// Compute backoff and notify before sleeping.
		backoff := retry.Backoff(c.monitor.RetryBackoff, attempt)
		if c.onRetry != nil {
			c.onRetry(RetryEvent{
				Attempt:     attempt,
				MaxAttempts: maxAttempts,
				Reason:      snap.Error,
				NextIn:      backoff,
			})
		}
		time.Sleep(backoff)
	}

	return lastSnap
}
