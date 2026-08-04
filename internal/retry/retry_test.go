package retry

import (
	"errors"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	transportErr := errors.New("transport error")

	tests := []struct {
		name         string
		statusCode   int
		transportErr error
		want         bool
	}{
		{
			name:         "transport error is retryable",
			statusCode:   0,
			transportErr: transportErr,
			want:         true,
		},
		{
			name:         "429 Too Many Requests is retryable",
			statusCode:   429,
			transportErr: nil,
			want:         true,
		},
		{
			name:         "500 Internal Server Error is retryable",
			statusCode:   500,
			transportErr: nil,
			want:         true,
		},
		{
			name:         "503 Service Unavailable is retryable",
			statusCode:   503,
			transportErr: nil,
			want:         true,
		},
		{
			name:         "404 Not Found is not retryable",
			statusCode:   404,
			transportErr: nil,
			want:         false,
		},
		{
			name:         "200 OK is not retryable",
			statusCode:   200,
			transportErr: nil,
			want:         false,
		},
		{
			name:         "parse error (nil transportErr + status 0) is not retryable",
			statusCode:   0,
			transportErr: nil,
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsRetryable(tc.statusCode, tc.transportErr)
			if got != tc.want {
				t.Errorf("IsRetryable(%d, %v) = %v, want %v", tc.statusCode, tc.transportErr, got, tc.want)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		name    string
		base    time.Duration
		attempt int
		want    time.Duration
	}{
		{
			name:    "attempt 1 returns base (2s)",
			base:    2 * time.Second,
			attempt: 1,
			want:    2 * time.Second,
		},
		{
			name:    "attempt 2 returns 2x base (4s)",
			base:    2 * time.Second,
			attempt: 2,
			want:    4 * time.Second,
		},
		{
			name:    "attempt 3 returns 4x base (8s)",
			base:    2 * time.Second,
			attempt: 3,
			want:    8 * time.Second,
		},
		{
			name:    "attempt 4 returns 8x base (16s)",
			base:    2 * time.Second,
			attempt: 4,
			want:    16 * time.Second,
		},
		{
			name:    "capped at 30s when progression exceeds maxBackoff",
			base:    2 * time.Second,
			attempt: 10,
			want:    30 * time.Second,
		},
		{
			name:    "base larger than 30s is capped immediately at attempt 1",
			base:    60 * time.Second,
			attempt: 1,
			want:    30 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Backoff(tc.base, tc.attempt)
			if got != tc.want {
				t.Errorf("Backoff(%v, %d) = %v, want %v", tc.base, tc.attempt, got, tc.want)
			}
		})
	}
}
