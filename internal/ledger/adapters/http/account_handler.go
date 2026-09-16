package httpadapter

import (
	"log/slog"
	"net/http"

	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/httpx"
	"financial-ledger/internal/platform/observability"
)

type AccountHandler struct {
	createAccount account.CreateAccountUseCase
	logger        *slog.Logger
}

func NewAccountHandler(
	createAccount account.CreateAccountUseCase,
	logger *slog.Logger,
) *AccountHandler {
	return &AccountHandler{
		createAccount: createAccount,
		logger:        logger,
	}
}

type createAccountRequest struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	CurrencyCode string `json:"currency"`
}

type accountResponse struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
}

func (h *AccountHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		writeHTTPError(
			w,
			r,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
			"method not allowed",
		)
		return
	}

	var request createAccountRequest

	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		writeHTTPError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	accountID, err := domain.NewAccountID(request.ID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	createdAccount, err := h.createAccount.Execute(
		r.Context(),
		account.CreateAccountCommand{
			ID:           accountID,
			Code:         request.Code,
			Name:         request.Name,
			CurrencyCode: request.CurrencyCode,
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"account_creation_failed",
				slog.String("operation", "account.create"),
				slog.String(
					"request_id",
					observability.RequestIDFromContext(r.Context()),
				),
				slog.Any("error", err),
			)
		}
		writeApplicationError(w, r, err)
		return
	}

	if h.logger != nil {
		h.logger.InfoContext(
			r.Context(),
			"account_created",
			slog.String("operation", "account.create"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String("account_id", createdAccount.ID().String()),
			slog.String("account_code", createdAccount.Code()),
			slog.String("currency", createdAccount.Currency().String()),
		)
	}

	httpx.WriteJSON(
		w,
		http.StatusCreated,
		accountResponse{
			ID:       createdAccount.ID().String(),
			Code:     createdAccount.Code(),
			Name:     createdAccount.Name(),
			Currency: createdAccount.Currency().String(),
			Status:   string(createdAccount.Status()),
		},
	)
}
