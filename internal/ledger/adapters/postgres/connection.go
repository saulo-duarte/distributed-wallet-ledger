package postgres

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectionConfig struct {
	DatabaseURL string

	MaxConns              int32
	MinConns              int32
	MaxConnLifetime       time.Duration
	MaxConnLifetimeJitter time.Duration
	MaxConnIdleTime       time.Duration
	HealthCheckPeriod     time.Duration
	PingTimeout           time.Duration

	ConnectTimeout time.Duration
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func DefaultConnectionConfig(databaseURL string) ConnectionConfig {
	return ConnectionConfig{
		DatabaseURL: databaseURL,

		MaxConns:              10,
		MinConns:              1,
		MaxConnLifetime:       30 * time.Minute,
		MaxConnLifetimeJitter: 5 * time.Minute,
		MaxConnIdleTime:       5 * time.Minute,
		HealthCheckPeriod:     30 * time.Second,
		PingTimeout:           5 * time.Second,

		ConnectTimeout: 5 * time.Second,
		MaxAttempts:    5,
		InitialBackoff: 200 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	}
}

func OpenPool(
	ctx context.Context,
	cfg ConnectionConfig,
) (*pgxpool.Pool, error) {
	if err := validateConnectionConfig(cfg); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres connection config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnLifetimeJitter = cfg.MaxConnLifetimeJitter
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolConfig.PingTimeout = cfg.PingTimeout

	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)

		pool, err := pgxpool.NewWithConfig(
			attemptCtx,
			poolConfig.Copy(),
		)

		if err == nil {
			err = pool.Ping(attemptCtx)
		}

		cancel()

		if err == nil {
			return pool, nil
		}

		if pool != nil {
			pool.Close()
		}

		lastErr = err

		if ctx.Err() != nil {
			return nil, fmt.Errorf(
				"postgres connection canceled: %w",
				ctx.Err(),
			)
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		delay := backoffDelay(
			attempt,
			cfg.InitialBackoff,
			cfg.MaxBackoff,
		)

		if err := waitForRetry(ctx, delay); err != nil {
			return nil, fmt.Errorf(
				"postgres connection retry canceled: %w",
				err,
			)
		}
	}

	return nil, fmt.Errorf(
		"could not connect to postgres after %d attempts: %w",
		cfg.MaxAttempts,
		lastErr,
	)
}

func validateConnectionConfig(cfg ConnectionConfig) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("postgres database URL cannot be empty")
	}

	if cfg.MaxConns <= 0 {
		return fmt.Errorf("postgres max connections must be greater than zero")
	}

	if cfg.MinConns < 0 {
		return fmt.Errorf("postgres min connections cannot be negative")
	}

	if cfg.MinConns > cfg.MaxConns {
		return fmt.Errorf(
			"postgres min connections cannot exceed max connections",
		)
	}

	if cfg.ConnectTimeout <= 0 {
		return fmt.Errorf("postgres connect timeout must be greater than zero")
	}

	if cfg.MaxAttempts <= 0 {
		return fmt.Errorf("postgres max attempts must be greater than zero")
	}

	if cfg.InitialBackoff <= 0 {
		return fmt.Errorf(
			"postgres initial backoff must be greater than zero",
		)
	}

	if cfg.MaxBackoff < cfg.InitialBackoff {
		return fmt.Errorf(
			"postgres max backoff cannot be smaller than initial backoff",
		)
	}

	return nil
}

func backoffDelay(
	attempt int,
	initial time.Duration,
	maximum time.Duration,
) time.Duration {
	delay := initial

	for i := 1; i < attempt; i++ {
		if delay >= maximum/2 {
			delay = maximum
			break
		}

		delay *= 2
	}

	if delay > maximum {
		delay = maximum
	}

	jitter := 0.5 + rand.Float64()*0.5

	return time.Duration(float64(delay) * jitter)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
