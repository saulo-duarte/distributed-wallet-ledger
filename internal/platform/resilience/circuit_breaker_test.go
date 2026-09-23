package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_InitialStateIsClosed(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name: "test-service",
	})

	if cb.State() != StateClosed {
		t.Fatalf("expected initial state to be closed, got %s", cb.State())
	}
}

func TestCircuitBreaker_TransitionsToOpenAfterThresholdFailures(t *testing.T) {
	var stateChangeCount int
	cb := NewCircuitBreaker(Config{
		Name:             "test-service",
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		OnStateChange: func(name string, from, to State) {
			stateChangeCount++
			if from != StateClosed || to != StateOpen {
				t.Errorf("unexpected state transition: %s -> %s", from, to)
			}
		},
	})

	errDummy := errors.New("temporary error")

	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return errDummy
		})
		if !errors.Is(err, errDummy) {
			t.Fatalf("expected dummy error, got %v", err)
		}
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected state to be open, got %s", cb.State())
	}

	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("expected ErrCircuitBreakerOpen, got %v", err)
	}

	if stateChangeCount != 1 {
		t.Fatalf("expected 1 state change, got %d", stateChangeCount)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpenAndClosesOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "test-service",
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          30 * time.Millisecond,
	})

	errDummy := errors.New("failure")

	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func() error {
			return errDummy
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected state to be open, got %s", cb.State())
	}

	time.Sleep(40 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state to be half-open after timeout, got %s", cb.State())
	}

	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state to still be half-open after 1 success, got %s", cb.State())
	}

	err = cb.Execute(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cb.State() != StateClosed {
		t.Fatalf("expected state to be closed after 2 successes, got %s", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenReopensImmediatelyOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "test-service",
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          30 * time.Millisecond,
	})

	errDummy := errors.New("failure")

	_ = cb.Execute(context.Background(), func() error {
		return errDummy
	})

	if cb.State() != StateOpen {
		t.Fatalf("expected state to be open, got %s", cb.State())
	}

	time.Sleep(40 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected state to be half-open, got %s", cb.State())
	}

	err := cb.Execute(context.Background(), func() error {
		return errDummy
	})
	if !errors.Is(err, errDummy) {
		t.Fatalf("expected dummy error, got %v", err)
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected state to re-open immediately on failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_RespectsContextCancellation(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name: "test-service",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Execute(ctx, func() error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(Config{
		Name:             "concurrent-service",
		FailureThreshold: 10,
		Timeout:          50 * time.Millisecond,
	})

	var wg sync.WaitGroup
	var executedCount int64

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = cb.Execute(context.Background(), func() error {
				atomic.AddInt64(&executedCount, 1)
				if id%2 == 0 {
					return errors.New("err")
				}
				return nil
			})
		}(i)
	}

	wg.Wait()

	if atomic.LoadInt64(&executedCount) == 0 {
		t.Fatalf("expected some executions to complete")
	}
}
