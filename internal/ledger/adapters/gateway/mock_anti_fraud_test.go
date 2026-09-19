package gateway

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/ledger/domain"
)

func TestMockAntiFraudService_Evaluate(t *testing.T) {
	t.Parallel()

	currency, err := domain.NewCurrency("BRL")
	if err != nil {
		t.Fatalf("failed to create currency: %v", err)
	}
	walletID, err := domain.NewWalletID("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	if err != nil {
		t.Fatalf("failed to create wallet id: %v", err)
	}

	t.Run("approved standard transaction", func(t *testing.T) {
		t.Parallel()
		service := NewMockAntiFraudService()

		resp, err := service.Evaluate(context.Background(), saga.AntiFraudRequest{
			WalletID:         walletID,
			AmountMinorUnits: 5000,
			Currency:         currency,
			Recipient:        "merchant_123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Approved {
			t.Fatalf("expected approved true, got false")
		}
		if resp.RiskScore != 10 {
			t.Fatalf("expected risk score 10, got %d", resp.RiskScore)
		}
		if service.CallsCount() != 1 {
			t.Fatalf("expected 1 call, got %d", service.CallsCount())
		}
	})

	t.Run("rejected flagged recipient", func(t *testing.T) {
		t.Parallel()
		service := NewMockAntiFraudService()

		resp, err := service.Evaluate(context.Background(), saga.AntiFraudRequest{
			WalletID:         walletID,
			AmountMinorUnits: 5000,
			Currency:         currency,
			Recipient:        "fraud_suspicious_account",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Approved {
			t.Fatalf("expected approved false, got true")
		}
		if resp.RiskScore != 99 {
			t.Fatalf("expected risk score 99, got %d", resp.RiskScore)
		}
	})

	t.Run("rejected amount exceeding limit", func(t *testing.T) {
		t.Parallel()
		service := NewMockAntiFraudService()

		resp, err := service.Evaluate(context.Background(), saga.AntiFraudRequest{
			WalletID:         walletID,
			AmountMinorUnits: MaxAllowedAmountMinorUnits + 1,
			Currency:         currency,
			Recipient:        "normal_merchant",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Approved {
			t.Fatalf("expected approved false, got true")
		}
		if resp.RiskScore != 90 {
			t.Fatalf("expected risk score 90, got %d", resp.RiskScore)
		}
	})

	t.Run("timeout error trigger", func(t *testing.T) {
		t.Parallel()
		service := NewMockAntiFraudService()

		_, err := service.Evaluate(context.Background(), saga.AntiFraudRequest{
			WalletID:         walletID,
			AmountMinorUnits: 1000,
			Currency:         currency,
			Recipient:        "timeout_fraud_checker",
		})
		if !errors.Is(err, ErrAntiFraudServiceUnavailable) {
			t.Fatalf("expected %v, got %v", ErrAntiFraudServiceUnavailable, err)
		}
	})

	t.Run("forced score overrides evaluation", func(t *testing.T) {
		t.Parallel()
		service := NewMockAntiFraudService()
		service.ForceScore(95)

		resp, err := service.Evaluate(context.Background(), saga.AntiFraudRequest{
			WalletID:         walletID,
			AmountMinorUnits: 100,
			Currency:         currency,
			Recipient:        "merchant_safe",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Approved {
			t.Fatalf("expected approved false for score 95, got true")
		}
		if resp.RiskScore != 95 {
			t.Fatalf("expected risk score 95, got %d", resp.RiskScore)
		}
	})
}
