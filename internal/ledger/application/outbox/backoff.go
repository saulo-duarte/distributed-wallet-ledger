package outbox

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
)

type BackoffStrategy interface {
	NextDelay(retryCount int) time.Duration
}

type ExponentialBackoff struct {
	initialInterval time.Duration
	maxInterval     time.Duration
	multiplier      float64
}

func NewExponentialBackoff(
	initialInterval time.Duration,
	maxInterval time.Duration,
	multiplier float64,
) ExponentialBackoff {
	if initialInterval <= 0 {
		initialInterval = 500 * time.Millisecond
	}
	if maxInterval <= 0 {
		maxInterval = 30 * time.Second
	}
	if multiplier <= 1.0 {
		multiplier = 2.0
	}
	return ExponentialBackoff{
		initialInterval: initialInterval,
		maxInterval:     maxInterval,
		multiplier:      multiplier,
	}
}

func (b ExponentialBackoff) NextDelay(retryCount int) time.Duration {
	if retryCount <= 0 {
		return 0
	}

	factor := math.Pow(b.multiplier, float64(retryCount-1))
	delay := time.Duration(float64(b.initialInterval) * factor)

	if delay > b.maxInterval || delay <= 0 {
		delay = b.maxInterval
	}

	maxNanos := delay.Nanoseconds()
	if maxNanos <= 0 {
		return delay
	}

	randomNanos, err := rand.Int(rand.Reader, big.NewInt(maxNanos))
	if err != nil {
		return delay
	}

	return time.Duration(randomNanos.Int64())
}
