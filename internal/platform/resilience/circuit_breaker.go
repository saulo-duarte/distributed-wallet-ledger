package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half_open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

type Config struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	OnStateChange    func(name string, from, to State)
}

type CircuitBreaker struct {
	mu               sync.RWMutex
	name             string
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	onStateChange    func(name string, from, to State)

	state            State
	failureCount     int
	successCount     int
	lastStateChange  time.Time
}

func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &CircuitBreaker{
		name:             cfg.Name,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
		onStateChange:    cfg.OnStateChange,
		state:            StateClosed,
		lastStateChange:  time.Now(),
	}
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.evaluateStateChange()
	return cb.state
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := cb.beforeExecution(); err != nil {
		return err
	}

	err := fn()

	cb.afterExecution(err)
	return err
}

func (cb *CircuitBreaker) beforeExecution() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.evaluateStateChange()

	if cb.state == StateOpen {
		return ErrCircuitBreakerOpen
	}

	return nil
}

func (cb *CircuitBreaker) afterExecution(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
		return
	}

	cb.onSuccess()
}

func (cb *CircuitBreaker) evaluateStateChange() {
	if cb.state == StateOpen && time.Since(cb.lastStateChange) >= cb.timeout {
		cb.transitionTo(StateHalfOpen)
	}
}

func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) onFailure() {
	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.failureCount = 0
	cb.successCount = 0

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, newState)
	}
}
