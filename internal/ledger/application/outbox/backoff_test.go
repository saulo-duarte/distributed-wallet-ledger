package outbox

import (
	"testing"
	"time"
)

func TestExponentialBackoff_NextDelay(t *testing.T) {
	t.Parallel()

	backoff := NewExponentialBackoff(
		500*time.Millisecond,
		10*time.Second,
		2.0,
	)

	tests := []struct {
		name       string
		retryCount int
		maxCeiling time.Duration
	}{
		{
			name:       "retry count 0 has no delay",
			retryCount: 0,
			maxCeiling: 0,
		},
		{
			name:       "retry count 1 has max initial delay",
			retryCount: 1,
			maxCeiling: 500 * time.Millisecond,
		},
		{
			name:       "retry count 2 max delay is 1s",
			retryCount: 2,
			maxCeiling: 1 * time.Second,
		},
		{
			name:       "retry count 3 max delay is 2s",
			retryCount: 3,
			maxCeiling: 2 * time.Second,
		},
		{
			name:       "retry count 4 max delay is 4s",
			retryCount: 4,
			maxCeiling: 4 * time.Second,
		},
		{
			name:       "retry count 5 max delay is 8s",
			retryCount: 5,
			maxCeiling: 8 * time.Second,
		},
		{
			name:       "retry count 6 caps at max delay",
			retryCount: 6,
			maxCeiling: 10 * time.Second,
		},
		{
			name:       "large retry count caps at max delay",
			retryCount: 50,
			maxCeiling: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := backoff.NextDelay(tt.retryCount)
			if tt.retryCount == 0 && got != 0 {
				t.Errorf("NextDelay(0) = %v, want 0", got)
			}
			if got > tt.maxCeiling || got < 0 {
				t.Errorf("NextDelay(%d) = %v, out of range [0, %v]", tt.retryCount, got, tt.maxCeiling)
			}
		})
	}
}

func TestExponentialBackoff_Defaults(t *testing.T) {
	t.Parallel()

	backoff := NewExponentialBackoff(0, 0, 0)

	if backoff.initialInterval != 500*time.Millisecond {
		t.Errorf("initialInterval = %v, want 500ms", backoff.initialInterval)
	}
	if backoff.maxInterval != 30*time.Second {
		t.Errorf("maxInterval = %v, want 30s", backoff.maxInterval)
	}
	if backoff.multiplier != 2.0 {
		t.Errorf("multiplier = %v, want 2.0", backoff.multiplier)
	}
}
