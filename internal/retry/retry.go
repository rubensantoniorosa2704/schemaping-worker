// Package retry provides utilities for classifying retryable HTTP failures
// and computing exponential backoff durations.
package retry

import "time"

// maxBackoff is the upper bound for any backoff interval.
const maxBackoff = 30 * time.Second

// IsRetryable reports whether a failed HTTP attempt should be retried.
//
// Retryable conditions (transient by nature):
//   - transportErr != nil: timeout, connection refused, DNS failure
//   - statusCode == 429: rate limit — server may recover shortly
//   - statusCode 500–599: server-side error — may recover on retry
//
// Non-retryable conditions (deterministic — retry won't help):
//   - status 4xx except 429 (e.g. 404): indicates a contract change, which is
//     exactly what SchemaPing is designed to detect — retrying would mask it
//   - parse error (transportErr nil, status outside retryable range): the
//     response was received but the body is invalid; retrying rarely fixes this
func IsRetryable(statusCode int, transportErr error) bool {
	if transportErr != nil {
		return true
	}
	if statusCode == 429 {
		return true
	}
	if statusCode >= 500 && statusCode <= 599 {
		return true
	}
	return false
}

// Backoff returns the wait duration before the given attempt number (1-indexed).
//
// The formula is: base * 2^(attempt-1), capped at maxBackoff (30s).
//
//	attempt 1 → base * 1
//	attempt 2 → base * 2
//	attempt 3 → base * 4
//	attempt 4 → base * 8
//
// If the computed value exceeds 30s, 30s is returned instead.
func Backoff(base time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	// Use integer shift to compute 2^(attempt-1) without floating-point.
	// Cap the shift to avoid overflow on large attempt numbers.
	shift := attempt - 1
	if shift > 62 {
		return maxBackoff
	}
	d := base * (1 << uint(shift))
	if d > maxBackoff || d < 0 { // d < 0 catches duration overflow
		return maxBackoff
	}
	return d
}
