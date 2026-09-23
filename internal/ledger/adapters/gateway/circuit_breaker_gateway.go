package gateway

import (
	"context"
	"errors"

	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/platform/observability"
	"financial-ledger/internal/platform/resilience"
)

type CircuitBreakerPaymentGateway struct {
	next    saga.PaymentGateway
	cb      *resilience.CircuitBreaker
	metrics *observability.Metrics
}

func NewCircuitBreakerPaymentGateway(
	next saga.PaymentGateway,
	cb *resilience.CircuitBreaker,
	metrics *observability.Metrics,
) *CircuitBreakerPaymentGateway {
	return &CircuitBreakerPaymentGateway{
		next:    next,
		cb:      cb,
		metrics: metrics,
	}
}

var _ saga.PaymentGateway = (*CircuitBreakerPaymentGateway)(nil)

func (g *CircuitBreakerPaymentGateway) ProcessPayment(
	ctx context.Context,
	req saga.ExternalPaymentRequest,
) (saga.ExternalPaymentResponse, error) {
	var resp saga.ExternalPaymentResponse

	err := g.cb.Execute(ctx, func() error {
		var execErr error
		resp, execErr = g.next.ProcessPayment(ctx, req)
		return execErr
	})

	if errors.Is(err, resilience.ErrCircuitBreakerOpen) {
		if g.metrics != nil {
			g.metrics.RecordCircuitBreakerRejection(g.cb.Name())
		}
	}

	return resp, err
}
