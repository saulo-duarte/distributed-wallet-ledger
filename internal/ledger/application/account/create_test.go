package account

import (
	"context"
	"errors"
	"testing"

	"financial-ledger/internal/ledger/domain"
)

func TestCreateAccountUseCase_Execute(t *testing.T) {
	t.Parallel()

	databaseError := errors.New("database unavailable")

	tests := []struct {
		name             string
		command          CreateAccountCommand
		repositoryError  error
		wantErr          error
		wantCreateCalls  int
		wantCreatedCode  string
		wantCreatedName  string
		wantCreatedMoney string
	}{
		{
			name: "creates and persists a valid account",
			command: CreateAccountCommand{
				ID:           domain.AccountID("account-001"),
				Code:         "  ACC-001  ",
				Name:         "  Primary Account  ",
				CurrencyCode: "brl",
			},
			wantCreateCalls:  1,
			wantCreatedCode:  "ACC-001",
			wantCreatedName:  "Primary Account",
			wantCreatedMoney: "BRL",
		},
		{
			name: "rejects an invalid currency before persistence",
			command: CreateAccountCommand{
				ID:           domain.AccountID("account-001"),
				Code:         "ACC-001",
				Name:         "Primary Account",
				CurrencyCode: "BR",
			},
			wantErr:         domain.ErrInvalidCurrency,
			wantCreateCalls: 0,
		},
		{
			name: "rejects an invalid account before persistence",
			command: CreateAccountCommand{
				ID:           domain.AccountID("account-001"),
				Code:         "   ",
				Name:         "Primary Account",
				CurrencyCode: "BRL",
			},
			wantErr:         domain.ErrEmptyAccountCode,
			wantCreateCalls: 0,
		},
		{
			name: "returns the repository error",
			command: CreateAccountCommand{
				ID:           domain.AccountID("account-001"),
				Code:         "ACC-001",
				Name:         "Primary Account",
				CurrencyCode: "BRL",
			},
			repositoryError: databaseError,
			wantErr:         databaseError,
			wantCreateCalls: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeAccountRepository{
				createErr: tt.repositoryError,
			}
			useCase := NewCreateAccountUseCase(repository)

			got, err := useCase.Execute(context.Background(), tt.command)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}

			if repository.createCalls != tt.wantCreateCalls {
				t.Fatalf("repository Create() calls = %d, want %d", repository.createCalls, tt.wantCreateCalls)
			}

			if tt.wantErr != nil {
				if !got.ID().IsZero() {
					t.Errorf("returned account ID = %q, want zero value", got.ID())
				}
				return
			}

			if got.Code() != tt.wantCreatedCode {
				t.Errorf("Code() = %q, want %q", got.Code(), tt.wantCreatedCode)
			}
			if got.Name() != tt.wantCreatedName {
				t.Errorf("Name() = %q, want %q", got.Name(), tt.wantCreatedName)
			}
			if got.Currency().String() != tt.wantCreatedMoney {
				t.Errorf("Currency() = %q, want %q", got.Currency(), tt.wantCreatedMoney)
			}
			if !got.IsOpen() {
				t.Error("new account should be open")
			}
		})
	}
}

func TestCreateAccountUseCase_PassesTheContextToRepository(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), contextKey("request-id"), "request-001")
	repository := &fakeAccountRepository{}
	useCase := NewCreateAccountUseCase(repository)

	_, err := useCase.Execute(ctx, CreateAccountCommand{
		ID:           domain.AccountID("account-001"),
		Code:         "ACC-001",
		Name:         "Primary Account",
		CurrencyCode: "BRL",
	})
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	if repository.receivedContext != ctx {
		t.Error("repository did not receive the application context")
	}
}

type fakeAccountRepository struct {
	createCalls     int
	createdAccount  domain.Account
	receivedContext context.Context
	createErr       error
}

func (f *fakeAccountRepository) Create(ctx context.Context, account domain.Account) error {
	f.createCalls++
	f.createdAccount = account
	f.receivedContext = ctx
	return f.createErr
}

type contextKey string
