package domain

import (
	"errors"
	"testing"
)

func TestNewPosting(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")
	validAmount, err := NewMoney(brl, 10000)
	if err != nil {
		t.Fatalf("falha ao criar Money válido para o teste: %v", err)
	}

	validPostingID := fakeValidPostingID()
	validAccountID := fakeValidAccountID()

	tests := []struct {
		name          string
		id            PostingID
		accountID     AccountID
		direction     PostingDirection
		amount        Money
		wantErr       error
		wantDirection PostingDirection
	}{
		{
			name:          "creates a debit posting",
			id:            validPostingID,
			accountID:     validAccountID,
			direction:     PostingDirectionDebit,
			amount:        validAmount,
			wantDirection: PostingDirectionDebit,
		},
		{
			name:          "creates a credit posting",
			id:            validPostingID,
			accountID:     validAccountID,
			direction:     PostingDirectionCredit,
			amount:        validAmount,
			wantDirection: PostingDirectionCredit,
		},
		{
			name:      "rejects a posting with an empty ID",
			id:        PostingID(""),
			accountID: validAccountID,
			direction: PostingDirectionDebit,
			amount:    validAmount,
			wantErr:   ErrInvalidID,
		},
		{
			name:      "rejects a posting with an empty account ID",
			id:        validPostingID,
			accountID: AccountID(""),
			direction: PostingDirectionDebit,
			amount:    validAmount,
			wantErr:   ErrInvalidID,
		},
		{
			name:      "rejects an invalid direction",
			id:        validPostingID,
			accountID: validAccountID,
			direction: PostingDirection("unknown"),
			amount:    validAmount,
			wantErr:   ErrInvalidPostingDirection,
		},
		{
			name:      "rejects a zero amount",
			id:        validPostingID,
			accountID: validAccountID,
			direction: PostingDirectionDebit,
			amount:    Money{currency: brl, amountMinorUnits: 0},
			wantErr:   ErrAmountMustBePositive,
		},
		{
			name:      "rejects a negative amount",
			id:        validPostingID,
			accountID: validAccountID,
			direction: PostingDirectionDebit,
			amount:    Money{currency: brl, amountMinorUnits: -1},
			wantErr:   ErrAmountMustNotBeNegative,
		},
		{
			name:      "rejects an invalid currency",
			id:        validPostingID,
			accountID: validAccountID,
			direction: PostingDirectionDebit,
			amount:    Money{currency: Currency{code: "BR"}, amountMinorUnits: 10000},
			wantErr:   ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			posting, err := NewPosting(tt.id, tt.accountID, tt.direction, tt.amount)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewPosting() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if posting.ID() != tt.id {
				t.Errorf("ID() = %q, want %q", posting.ID(), tt.id)
			}

			if posting.AccountID() != tt.accountID {
				t.Errorf("AccountID() = %q, want %q", posting.AccountID(), tt.accountID)
			}

			if posting.Direction() != tt.wantDirection {
				t.Errorf("Direction() = %q, want %q", posting.Direction(), tt.wantDirection)
			}

			if !posting.Amount().Equal(tt.amount) {
				t.Errorf("Amount() = %v, want %v", posting.Amount(), tt.amount)
			}
		})
	}
}

func TestPosting_Validate(t *testing.T) {
	t.Parallel()

	brl := fakeValidCurrency("BRL")

	tests := []struct {
		name    string
		posting Posting
		wantErr error
	}{
		{
			name: "posting válido",
			posting: Posting{
				id:        fakeValidPostingID(),
				accountID: fakeValidAccountID(),
				direction: PostingDirectionDebit,
				amount:    Money{currency: brl, amountMinorUnits: 10000},
			},
		},
		{
			name:    "zero value",
			posting: Posting{},
			wantErr: ErrInvalidID,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.posting.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
