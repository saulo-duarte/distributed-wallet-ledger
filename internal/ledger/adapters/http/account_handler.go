package httpadapter

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/observability"
)

const maxRequestBodySize = 1 << 20

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

type errorResponse struct {
	Error string `json:"error"`
}

func (h *AccountHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	defer r.Body.Close()

	decoder := json.NewDecoder(
		http.MaxBytesReader(
			w,
			r.Body,
			maxRequestBodySize,
		),
	)

	decoder.DisallowUnknownFields()

	var request createAccountRequest

	if err := decoder.Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	if err := ensureSingleJSONValue(decoder); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"request body must contain only one JSON object",
		)
		return
	}

	accountID, err := domain.NewAccountID(request.ID)
	if err != nil {
		writeApplicationError(w, err)
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
		writeApplicationError(w, err)
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

	writeJSON(
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

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any

	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}

	if err == nil {
		return errors.New("multiple JSON values")
	}

	return err
}

func writeApplicationError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	message := "internal server error"

	if isClientError(err) {
		statusCode = http.StatusBadRequest
		message = err.Error()
	}

	writeError(w, statusCode, message)
}

func isClientError(err error) bool {
	return errors.Is(err, domain.ErrInvalidID) ||
		errors.Is(err, domain.ErrInvalidCurrency) ||
		errors.Is(err, domain.ErrCurrencyMismatch) ||
		errors.Is(err, domain.ErrAmountMustNotBeNegative) ||
		errors.Is(err, domain.ErrAmountMustBePositive) ||
		errors.Is(err, domain.ErrAmountOverflow) ||
		errors.Is(err, domain.ErrInvalidPostingDirection) ||
		errors.Is(err, domain.ErrJournalEntryWithoutPostings) ||
		errors.Is(err, domain.ErrJournalEntryWithoutDebit) ||
		errors.Is(err, domain.ErrJournalEntryWithoutCredit) ||
		errors.Is(err, domain.ErrUnbalancedJournalEntry) ||
		errors.Is(err, domain.ErrEmptyTransactionDescription) ||
		errors.Is(err, domain.ErrEmptyAccountCode) ||
		errors.Is(err, domain.ErrEmptyAccountName) ||
		errors.Is(err, domain.ErrInvalidAccount) ||
		errors.Is(err, domain.ErrInvalidAccountStatus) ||
		isTransactionClientError(err)
}

func writeError(
	w http.ResponseWriter,
	statusCode int,
	message string,
) {
	writeJSON(
		w,
		statusCode,
		errorResponse{
			Error: message,
		},
	)
}

func writeJSON(
	w http.ResponseWriter,
	statusCode int,
	payload any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}
