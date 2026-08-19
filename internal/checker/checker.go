// Package checker orchestrates a single monitor check, producing a Snapshot.
// It handles retry logic with exponential backoff for transient failures.
package checker

import (
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
	"github.com/rubensantoniorosa2704/schemaping-worker/internal/retry"
)

// RetryEvent carries information about a failed attempt, delivered to the
// onRetry callback before the checker sleeps and retries.
type RetryEvent struct {
	Attempt     int           // number of the attempt that failed (starts at 1)
	MaxAttempts int           // total configured attempts (retries + 1)
	Reason      string        // error message from the failed attempt
	NextIn      time.Duration // wait interval before next attempt
}

// Checker runs checks for a single monitor, delegating fetching to a domain.Fetcher.
type Checker struct {
	monitor domain.Monitor
	fetcher domain.Fetcher
	onRetry func(RetryEvent)
}

// New creates a Checker for the given monitor.
// fetcher is the port implementation used to retrieve snapshots.
// onRetry is called before each retry sleep with details about the failed attempt; it may be nil.
func New(m domain.Monitor, fetcher domain.Fetcher, onRetry func(RetryEvent)) *Checker {
	return &Checker{
		monitor: m,
		fetcher: fetcher,
		onRetry: onRetry,
	}
}

// Run executes the check with retry logic and returns a single final Snapshot.
// Failures are captured in Snapshot.Error; this method never returns an error.
func (c *Checker) Run() domain.Snapshot {
	// Determine total attempts: retries + 1 original. A nil Retries means the
	// config default of 3 was not yet applied — treat conservatively as 4 total.
	maxAttempts := 4
	if c.monitor.Retries != nil {
		maxAttempts = *c.monitor.Retries + 1
	}

	var lastSnap domain.Snapshot

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		snap := c.fetcher.Fetch(c.monitor)
		lastSnap = snap

		// Success — return immediately.
		if snap.Error == "" {
			return snap
		}

		// Non-retryable failure — return immediately (e.g. 404, parse error).
		if !retry.IsRetryable(snap.StatusCode, snap.TransportErr) {
			return snap
		}

		// Exhausted all attempts.
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
