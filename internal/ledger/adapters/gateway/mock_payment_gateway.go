package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"financial-ledger/internal/ledger/application/saga"
)

var (
	ErrNetworkTimeout  = errors.New("network timeout communicating with payment provider")
	ErrCardDeclined    = errors.New("payment declined by card issuer")
	ErrAntiFraudBlock  = errors.New("payment blocked by anti-fraud system")
	ErrInvalidMerchant = errors.New("merchant account is inactive")
)

type MockPaymentGateway struct {
	mu           sync.RWMutex
	forcedError  error
	forcedResult *saga.ExternalPaymentResponse
	delay        time.Duration
	callsCount   int
}

func NewMockPaymentGateway() *MockPaymentGateway {
	return &MockPaymentGateway{}
}

var _ saga.PaymentGateway = (*MockPaymentGateway)(nil)

func (m *MockPaymentGateway) ForceError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forcedError = err
}

func (m *MockPaymentGateway) ForceResult(resp saga.ExternalPaymentResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forcedResult = &resp
}

func (m *MockPaymentGateway) SetDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = d
}

func (m *MockPaymentGateway) CallsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callsCount
}

func (m *MockPaymentGateway) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forcedError = nil
	m.forcedResult = nil
	m.delay = 0
	m.callsCount = 0
}

func (m *MockPaymentGateway) ProcessPayment(
	ctx context.Context,
	req saga.ExternalPaymentRequest,
) (saga.ExternalPaymentResponse, error) {
	m.mu.Lock()
	m.callsCount++
	delay := m.delay
	forcedErr := m.forcedError
	forcedRes := m.forcedResult
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return saga.ExternalPaymentResponse{}, ctx.Err()
		}
	}

	if forcedErr != nil {
		return saga.ExternalPaymentResponse{}, forcedErr
	}

	if forcedRes != nil {
		return *forcedRes, nil
	}

	if strings.HasPrefix(req.Recipient, "declined_") {
		return saga.ExternalPaymentResponse{
			Success:      false,
			ErrorMessage: ErrCardDeclined.Error(),
		}, nil
	}

	if strings.HasPrefix(req.Recipient, "fraud_") {
		return saga.ExternalPaymentResponse{
			Success:      false,
			ErrorMessage: ErrAntiFraudBlock.Error(),
		}, nil
	}

	if strings.HasPrefix(req.Recipient, "timeout_") {
		return saga.ExternalPaymentResponse{}, ErrNetworkTimeout
	}

	return saga.ExternalPaymentResponse{
		Success:       true,
		TransactionID: fmt.Sprintf("gw_tx_%s", req.PaymentID),
	}, nil
}
