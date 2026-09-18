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
		expected   time.Duration
	}{
		{
			name:       "retry count 0 has no delay",
			retryCount: 0,
			expected:   0,
		},
		{
			name:       "retry count 1 has initial delay",
			retryCount: 1,
			expected:   500 * time.Millisecond,
		},
		{
			name:       "retry count 2 doubles delay",
			retryCount: 2,
			expected:   1 * time.Second,
		},
		{
			name:       "retry count 3 quadruples delay",
			retryCount: 3,
			expected:   2 * time.Second,
		},
		{
			name:       "retry count 4 delay is 4s",
			retryCount: 4,
			expected:   4 * time.Second,
		},
		{
			name:       "retry count 5 delay is 8s",
			retryCount: 5,
			expected:   8 * time.Second,
		},
		{
			name:       "retry count 6 caps at max delay",
			retryCount: 6,
			expected:   10 * time.Second,
		},
		{
			name:       "large retry count caps at max delay",
			retryCount: 50,
			expected:   10 * time.Second,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := backoff.NextDelay(tt.retryCount)
			if got != tt.expected {
				t.Errorf("NextDelay(%d) = %v, want %v", tt.retryCount, got, tt.expected)
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
