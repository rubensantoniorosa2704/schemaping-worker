package checker

import (
	"errors"
	"testing"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/internal/domain"
)

// monitorWithRetries builds a Monitor pre-configured with retry settings so
// tests don't depend on config.Load defaults.
func monitorWithRetries(retries int, backoff time.Duration) domain.Monitor {
	return domain.Monitor{
		Name:           "test-monitor",
		URL:            "https://example.com",
		Method:         "GET",
		Timeout:        5 * time.Second,
		ExpectedStatus: 200,
		Retries:        &retries,
		RetryBackoff:   backoff,
	}
}

// stubFetcher is a domain.Fetcher that replays a sequence of canned Snapshots.
type stubFetcher struct {
	responses []domain.Snapshot
	call      int
}

func (s *stubFetcher) Fetch(_ domain.Monitor) domain.Snapshot {
	snap := s.responses[s.call]
	if s.call < len(s.responses)-1 {
		s.call++
	}
	return snap
}

// snapOK returns a successful Snapshot.
func snapOK() domain.Snapshot {
	return domain.Snapshot{StatusCode: 200, Body: map[string]any{"ok": true}}
}

// snapTransportErr returns a Snapshot representing a transport-level failure.
func snapTransportErr(msg string) domain.Snapshot {
	err := errors.New(msg)
	return domain.Snapshot{Error: msg, TransportErr: err}
}

// snapStatus returns a Snapshot with a non-success status (non-retryable unless 5xx/429).
func snapStatus(code int) domain.Snapshot {
	return domain.Snapshot{
		StatusCode: code,
		Error:      "unexpected status",
	}
}

// --- success on first attempt ---

func TestRun_SuccessOnFirstAttempt(t *testing.T) {
	m := monitorWithRetries(3, 0)
	f := &stubFetcher{responses: []domain.Snapshot{snapOK()}}
	c := New(m, f, nil)

	snap := c.Run()
	if snap.Error != "" {
		t.Errorf("expected no error, got: %s", snap.Error)
	}
	if snap.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", snap.StatusCode)
	}
}

// --- retries on transport error then succeeds ---

func TestRun_RetriesOnTransportError_ThenSucceeds(t *testing.T) {
	m := monitorWithRetries(3, 0)
	var events []RetryEvent
	f := &stubFetcher{responses: []domain.Snapshot{
		snapTransportErr("connection refused"),
		snapTransportErr("connection refused"),
		snapOK(),
	}}
	c := New(m, f, func(e RetryEvent) { events = append(events, e) })

	snap := c.Run()
	if snap.Error != "" {
		t.Errorf("expected success, got error: %s", snap.Error)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 retry events, got %d", len(events))
	}
	if events[0].Attempt != 1 || events[1].Attempt != 2 {
		t.Errorf("unexpected attempt numbers: %v", events)
	}
}

// --- exhausts all retries and returns last error ---

func TestRun_ExhaustsRetries(t *testing.T) {
	retries := 2
	m := monitorWithRetries(retries, 0)
	callCount := 0
	c := &Checker{
		monitor: m,
		fetcher: fetcherFunc(func(_ domain.Monitor) domain.Snapshot {
			callCount++
			return snapTransportErr("timeout")
		}),
	}

	snap := c.Run()
	if snap.Error == "" {
		t.Error("expected error after exhausting retries, got none")
	}
	// retries=2 means 3 total attempts (1 original + 2 retries)
	if callCount != 3 {
		t.Errorf("expected 3 total attempts, got %d", callCount)
	}
}

// --- non-retryable failure stops immediately ---

func TestRun_NonRetryable_StopsImmediately(t *testing.T) {
	m := monitorWithRetries(3, 0)
	callCount := 0
	c := &Checker{
		monitor: m,
		fetcher: fetcherFunc(func(_ domain.Monitor) domain.Snapshot {
			callCount++
			return snapStatus(404)
		}),
	}

	snap := c.Run()
	if snap.Error == "" {
		t.Error("expected error for unexpected status, got none")
	}
	if callCount != 1 {
		t.Errorf("expected 1 attempt for non-retryable failure, got %d", callCount)
	}
}

// --- retries: 0 disables all retries ---

func TestRun_ZeroRetries_NoRetry(t *testing.T) {
	m := monitorWithRetries(0, 0)
	callCount := 0
	c := &Checker{
		monitor: m,
		fetcher: fetcherFunc(func(_ domain.Monitor) domain.Snapshot {
			callCount++
			return snapStatus(503)
		}),
	}

	snap := c.Run()
	if snap.Error == "" {
		t.Error("expected error, got none")
	}
	if callCount != 1 {
		t.Errorf("retries=0 should make exactly 1 attempt, got %d", callCount)
	}
}

// --- 429 is retryable ---

func TestRun_Status429_IsRetried(t *testing.T) {
	m := monitorWithRetries(2, 0)
	callCount := 0
	c := &Checker{
		monitor: m,
		fetcher: fetcherFunc(func(_ domain.Monitor) domain.Snapshot {
			callCount++
			if callCount < 3 {
				return snapStatus(429)
			}
			return snapOK()
		}),
	}

	snap := c.Run()
	if snap.Error != "" {
		t.Errorf("expected success after retrying 429, got: %s", snap.Error)
	}
	if callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", callCount)
	}
}

// --- 5xx is retryable ---

func TestRun_Status503_IsRetried(t *testing.T) {
	m := monitorWithRetries(1, 0)
	callCount := 0
	c := &Checker{
		monitor: m,
		fetcher: fetcherFunc(func(_ domain.Monitor) domain.Snapshot {
			callCount++
			if callCount == 1 {
				return snapStatus(503)
			}
			return snapOK()
		}),
	}

	snap := c.Run()
	if snap.Error != "" {
		t.Errorf("expected success after retrying 503, got: %s", snap.Error)
	}
	if callCount != 2 {
		t.Errorf("expected 2 attempts, got %d", callCount)
	}
}

// --- onRetry callback receives correct fields ---

func TestRun_OnRetryCallback_ReceivesCorrectFields(t *testing.T) {
	retries := 1
	backoff := 5 * time.Millisecond
	m := monitorWithRetries(retries, backoff)
	var events []RetryEvent
	f := &stubFetcher{responses: []domain.Snapshot{
		snapTransportErr("timeout"),
		snapOK(),
	}}
	c := New(m, f, func(e RetryEvent) { events = append(events, e) })

	snap := c.Run()
	if snap.Error != "" {
		t.Errorf("expected success, got: %s", snap.Error)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 retry event, got %d", len(events))
	}
	e := events[0]
	if e.Attempt != 1 {
		t.Errorf("Attempt: want 1, got %d", e.Attempt)
	}
	if e.MaxAttempts != 2 {
		t.Errorf("MaxAttempts: want 2, got %d", e.MaxAttempts)
	}
	if e.NextIn != backoff {
		t.Errorf("NextIn: want %v, got %v", backoff, e.NextIn)
	}
	if e.Reason == "" {
		t.Error("Reason should not be empty")
	}
}

// --- nil Retries treated as default 3 retries (4 total attempts) ---

func TestRun_NilRetries_UsesDefault(t *testing.T) {
	m := domain.Monitor{
		Name:           "test",
		URL:            "https://example.com",
		Method:         "GET",
		ExpectedStatus: 200,
		Retries:        nil,
		RetryBackoff:   0,
	}
	callCount := 0
	c := &Checker{
		monitor: m,
		fetcher: fetcherFunc(func(_ domain.Monitor) domain.Snapshot {
			callCount++
			return snapStatus(500)
		}),
	}

	c.Run()
	if callCount != 4 {
		t.Errorf("nil Retries should result in 4 total attempts, got %d", callCount)
	}
}

// fetcherFunc is an adapter to use a plain function as domain.Fetcher.
type fetcherFunc func(domain.Monitor) domain.Snapshot

func (f fetcherFunc) Fetch(m domain.Monitor) domain.Snapshot { return f(m) }
