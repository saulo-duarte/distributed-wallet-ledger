package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/ledger/domain"
)

func TestMockPaymentGateway_ProcessPayment(t *testing.T) {
	t.Parallel()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}

	t.Run("successful standard payment", func(t *testing.T) {
		t.Parallel()
		gw := NewMockPaymentGateway()

		resp, err := gw.ProcessPayment(context.Background(), saga.ExternalPaymentRequest{
			PaymentID:        "pay-123",
			AmountMinorUnits: 1500,
			Currency:         currency,
			Recipient:        "clean_merchant",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success true, got false")
		}
		if resp.TransactionID != "gw_tx_pay-123" {
			t.Fatalf("expected transaction id gw_tx_pay-123, got %s", resp.TransactionID)
		}
		if gw.CallsCount() != 1 {
			t.Fatalf("expected 1 call, got %d", gw.CallsCount())
		}
	})

	t.Run("declined payment via recipient trigger", func(t *testing.T) {
		t.Parallel()
		gw := NewMockPaymentGateway()

		resp, err := gw.ProcessPayment(context.Background(), saga.ExternalPaymentRequest{
			PaymentID:        "pay-456",
			AmountMinorUnits: 2000,
			Currency:         currency,
			Recipient:        "declined_card_issuer",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Success {
			t.Fatalf("expected success false, got true")
		}
		if resp.ErrorMessage != ErrCardDeclined.Error() {
			t.Fatalf("expected %s, got %s", ErrCardDeclined.Error(), resp.ErrorMessage)
		}
	})

	t.Run("timeout error via recipient trigger", func(t *testing.T) {
		t.Parallel()
		gw := NewMockPaymentGateway()

		_, err := gw.ProcessPayment(context.Background(), saga.ExternalPaymentRequest{
			PaymentID:        "pay-789",
			AmountMinorUnits: 3000,
			Currency:         currency,
			Recipient:        "timeout_endpoint",
		})
		if !errors.Is(err, ErrNetworkTimeout) {
			t.Fatalf("expected error %v, got %v", ErrNetworkTimeout, err)
		}
	})

	t.Run("forced error overrides logic", func(t *testing.T) {
		t.Parallel()
		gw := NewMockPaymentGateway()
		expectedErr := errors.New("custom gateway failure")
		gw.ForceError(expectedErr)

		_, err := gw.ProcessPayment(context.Background(), saga.ExternalPaymentRequest{
			PaymentID:        "pay-999",
			AmountMinorUnits: 1000,
			Currency:         currency,
			Recipient:        "clean_merchant",
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("context cancellation during delay", func(t *testing.T) {
		t.Parallel()
		gw := NewMockPaymentGateway()
		gw.SetDelay(200 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := gw.ProcessPayment(ctx, saga.ExternalPaymentRequest{
			PaymentID:        "pay-timeout",
			AmountMinorUnits: 1000,
			Currency:         currency,
			Recipient:        "clean_merchant",
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}
