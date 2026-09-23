package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/resilience"
)

func TestCircuitBreakerPaymentGateway_TripsAndRejects(t *testing.T) {
	mockGW := NewMockPaymentGateway()
	mockGW.ForceError(errors.New("gateway unavailable"))

	cb := resilience.NewCircuitBreaker(resilience.Config{
		Name:             "payment-gateway",
		FailureThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cbGW := NewCircuitBreakerPaymentGateway(mockGW, cb, nil)

	curr, _ := domain.NewCurrency("BRL")
	req := saga.ExternalPaymentRequest{
		PaymentID:        "pay-123",
		AmountMinorUnits: 1000,
		Currency:         curr,
		Recipient:        "recipient@test.com",
	}

	for i := 0; i < 2; i++ {
		_, err := cbGW.ProcessPayment(context.Background(), req)
		if err == nil {
			t.Fatalf("expected error from gateway")
		}
	}

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected circuit breaker to be open")
	}

	_, err := cbGW.ProcessPayment(context.Background(), req)
	if !errors.Is(err, resilience.ErrCircuitBreakerOpen) {
		t.Fatalf("expected ErrCircuitBreakerOpen, got %v", err)
	}

	mockGW.ForceError(nil)
	mockGW.ForceResult(saga.ExternalPaymentResponse{
		Success:       true,
		TransactionID: "gw-txn-001",
	})

	time.Sleep(60 * time.Millisecond)

	resp, err := cbGW.ProcessPayment(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful call in half-open state, got: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response")
	}
}
