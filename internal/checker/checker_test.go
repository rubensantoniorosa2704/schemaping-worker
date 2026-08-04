package checker

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// monitorWithRetries builds a Monitor pre-configured with retry settings so
// tests don't depend on config.Load defaults.
func monitorWithRetries(retries int, backoff time.Duration) types.Monitor {
	return types.Monitor{
		Name:           "test-monitor",
		URL:            "https://example.com",
		Method:         "GET",
		Timeout:        5 * time.Second,
		ExpectedStatus: 200,
		Retries:        &retries,
		RetryBackoff:   backoff,
	}
}

// stubResponse represents a single canned HTTP response or transport error.
type stubResponse struct {
	statusCode int
	body       []byte
	err        error
}

// stubSequence returns a doRequest func that replays responses in order.
// When exhausted, the last entry repeats.
func stubSequence(responses []stubResponse) func(*http.Client, types.Monitor) (int, []byte, error) {
	call := 0
	return func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		r := responses[call]
		if call < len(responses)-1 {
			call++
		}
		return r.statusCode, r.body, r.err
	}
}

// --- success on first attempt ---

func TestRun_SuccessOnFirstAttempt(t *testing.T) {
	m := monitorWithRetries(3, 0)
	c := New(m, nil)
	c.doRequest = stubSequence([]stubResponse{
		{statusCode: 200, body: []byte(`{"ok":true}`)},
	})

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
	m := monitorWithRetries(3, 0) // zero backoff so tests don't sleep
	var events []RetryEvent
	c := New(m, func(e RetryEvent) { events = append(events, e) })
	c.doRequest = stubSequence([]stubResponse{
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
		{statusCode: 200, body: []byte(`{"ok":true}`)},
	})

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
	c := New(m, nil)
	c.doRequest = func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		callCount++
		return 0, nil, errors.New("timeout")
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

// --- non-retryable failure stops immediately (no retry) ---

func TestRun_NonRetryable_StopsImmediately(t *testing.T) {
	m := monitorWithRetries(3, 0)
	callCount := 0
	c := New(m, nil)
	c.doRequest = func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		callCount++
		return 404, []byte(`not found`), nil
	}

	snap := c.Run()
	if snap.Error == "" {
		t.Error("expected error for unexpected status, got none")
	}
	// 404 is non-retryable — should stop after exactly 1 attempt
	if callCount != 1 {
		t.Errorf("expected 1 attempt for non-retryable failure, got %d", callCount)
	}
}

// --- retries: 0 disables all retries ---

func TestRun_ZeroRetries_NoRetry(t *testing.T) {
	m := monitorWithRetries(0, 0)
	callCount := 0
	c := New(m, nil)
	c.doRequest = func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		callCount++
		return 503, nil, nil
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
	c := New(m, nil)
	c.doRequest = func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		callCount++
		if callCount < 3 {
			return 429, nil, nil
		}
		return 200, []byte(`{"ok":true}`), nil
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
	c := New(m, nil)
	c.doRequest = func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		callCount++
		if callCount == 1 {
			return 503, nil, nil
		}
		return 200, []byte(`{"ok":true}`), nil
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
	c := New(m, func(e RetryEvent) { events = append(events, e) })
	c.doRequest = stubSequence([]stubResponse{
		{err: errors.New("timeout")},
		{statusCode: 200, body: []byte(`{}`)},
	})

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
		// retries=1 → maxAttempts=2
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
	m := types.Monitor{
		Name:           "test",
		URL:            "https://example.com",
		Method:         "GET",
		ExpectedStatus: 200,
		Retries:        nil, // not configured
		RetryBackoff:   0,
	}
	callCount := 0
	c := New(m, nil)
	c.doRequest = func(_ *http.Client, _ types.Monitor) (int, []byte, error) {
		callCount++
		return 500, nil, nil
	}

	c.Run()
	// nil Retries → default 3 retries → 4 total attempts
	if callCount != 4 {
		t.Errorf("nil Retries should result in 4 total attempts, got %d", callCount)
	}
}
