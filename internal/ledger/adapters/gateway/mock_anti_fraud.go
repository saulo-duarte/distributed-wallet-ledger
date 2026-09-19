package gateway

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"financial-ledger/internal/ledger/application/saga"
)

var (
	ErrAntiFraudServiceUnavailable = errors.New("anti-fraud service unavailable")
	ErrHighRiskTransaction         = errors.New("transaction rejected by anti-fraud: high risk score")
)

const (
	MaxAllowedAmountMinorUnits = 5_000_000
	HighRiskScoreThreshold     = 85
)

type MockAntiFraudService struct {
	mu            sync.RWMutex
	forcedError   error
	forcedScore   *int
	delay         time.Duration
	evaluatedRisk map[string]int
	callsCount    int
}

func NewMockAntiFraudService() *MockAntiFraudService {
	return &MockAntiFraudService{
		evaluatedRisk: make(map[string]int),
	}
}

var _ saga.AntiFraudChecker = (*MockAntiFraudService)(nil)

func (m *MockAntiFraudService) ForceError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forcedError = err
}

func (m *MockAntiFraudService) ForceScore(score int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forcedScore = &score
}

func (m *MockAntiFraudService) SetDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = d
}

func (m *MockAntiFraudService) CallsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callsCount
}

func (m *MockAntiFraudService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forcedError = nil
	m.forcedScore = nil
	m.delay = 0
	m.callsCount = 0
	m.evaluatedRisk = make(map[string]int)
}

func (m *MockAntiFraudService) Evaluate(
	ctx context.Context,
	req saga.AntiFraudRequest,
) (saga.AntiFraudResponse, error) {
	m.mu.Lock()
	m.callsCount++
	delay := m.delay
	forcedErr := m.forcedError
	forcedScore := m.forcedScore
	m.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return saga.AntiFraudResponse{}, ctx.Err()
		}
	}

	if forcedErr != nil {
		return saga.AntiFraudResponse{}, forcedErr
	}

	if forcedScore != nil {
		approved := *forcedScore < HighRiskScoreThreshold
		var reason string
		if !approved {
			reason = ErrHighRiskTransaction.Error()
		}
		return saga.AntiFraudResponse{
			Approved:     approved,
			RiskScore:    *forcedScore,
			RejectReason: reason,
		}, nil
	}

	if strings.HasPrefix(req.Recipient, "timeout_fraud") {
		return saga.AntiFraudResponse{}, ErrAntiFraudServiceUnavailable
	}

	if strings.HasPrefix(req.Recipient, "fraud_") {
		return saga.AntiFraudResponse{
			Approved:     false,
			RiskScore:    99,
			RejectReason: "recipient flagged in fraud blacklist",
		}, nil
	}

	if req.AmountMinorUnits > MaxAllowedAmountMinorUnits {
		return saga.AntiFraudResponse{
			Approved:     false,
			RiskScore:    90,
			RejectReason: "transaction amount exceeds risk limit",
		}, nil
	}

	return saga.AntiFraudResponse{
		Approved:     true,
		RiskScore:    10,
		RejectReason: "",
	}, nil
}
