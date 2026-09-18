package wallet

import (
	"context"
	"encoding/json"
	"fmt"

	"financial-ledger/internal/ledger/application/outbox"
	"financial-ledger/internal/ledger/domain"
)

const WalletBalanceUpdatedEventType = "WalletBalanceUpdated"

type WalletBalanceUpdatedPayload struct {
	WalletID string `json:"wallet_id"`
}

type BalanceProjector interface {
	ProjectBalance(ctx context.Context, walletID domain.WalletID) (WalletBalance, error)
}

type WalletBalanceEventPublisher struct {
	projector BalanceProjector
}

func NewWalletBalanceEventPublisher(projector BalanceProjector) *WalletBalanceEventPublisher {
	return &WalletBalanceEventPublisher{
		projector: projector,
	}
}

var _ outbox.EventPublisher = (*WalletBalanceEventPublisher)(nil)

func (p *WalletBalanceEventPublisher) Publish(
	ctx context.Context,
	event domain.OutboxEvent,
) error {
	if event.EventType() != WalletBalanceUpdatedEventType {
		return nil
	}

	var payload WalletBalanceUpdatedPayload
	if err := json.Unmarshal(event.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal wallet balance updated event payload: %w", err)
	}

	walletID, err := domain.NewWalletID(payload.WalletID)
	if err != nil {
		return fmt.Errorf("invalid wallet id in outbox payload: %w", err)
	}

	_, err = p.projector.ProjectBalance(ctx, walletID)
	if err != nil {
		return fmt.Errorf("project wallet balance for %q: %w", walletID, err)
	}

	return nil
}
