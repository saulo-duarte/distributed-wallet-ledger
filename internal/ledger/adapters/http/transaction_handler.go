package httpadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/observability"
)

const idempotencyKeyHeader = "Idempotency-Key"

type TransactionHandler struct {
	postTransaction transaction.PostTransactionUseCase
	getTransaction  transaction.GetTransactionUseCase
	logger          *slog.Logger
}

func NewTransactionHandler(
	postTransaction transaction.PostTransactionUseCase,
	getTransaction transaction.GetTransactionUseCase,
	logger *slog.Logger,
) *TransactionHandler {
	return &TransactionHandler{
		postTransaction: postTransaction,
		getTransaction:  getTransaction,
		logger:          logger,
	}
}

type postTransactionRequest struct {
	ID             string           `json:"id"`
	JournalEntryID string           `json:"journal_entry_id"`
	Description    string           `json:"description"`
	Currency       string           `json:"currency"`
	Postings       []postingRequest `json:"postings"`
}

type postingRequest struct {
	ID               string `json:"id"`
	AccountID        string `json:"account_id"`
	Direction        string `json:"direction"`
	AmountMinorUnits int64  `json:"amount_minor_units"`
}

type transactionResponse struct {
	ID             string `json:"id"`
	JournalEntryID string `json:"journal_entry_id"`
	Description    string `json:"description"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
}

type transactionDetailsResponse struct {
	ID           string                          `json:"id"`
	Description  string                          `json:"description"`
	CreatedAt    time.Time                       `json:"created_at"`
	JournalEntry transactionJournalEntryResponse `json:"journal_entry"`
}

type transactionJournalEntryResponse struct {
	ID       string                       `json:"id"`
	Currency string                       `json:"currency"`
	PostedAt time.Time                    `json:"posted_at"`
	Postings []transactionPostingResponse `json:"postings"`
}

type transactionPostingResponse struct {
	ID               string `json:"id"`
	AccountID        string `json:"account_id"`
	Direction        string `json:"direction"`
	AmountMinorUnits int64  `json:"amount_minor_units"`
}

func (h *TransactionHandler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	transactionID, err := domain.NewTransactionID(
		r.PathValue("transactionID"),
	)
	if err != nil {
		writeApplicationError(w, err)
		return
	}

	details, err := h.getTransaction.Execute(
		r.Context(),
		transactionID,
	)
	if err != nil {
		if errors.Is(err, transaction.ErrTransactionNotFound) {
			writeError(w, http.StatusNotFound, "transaction not found")
			return
		}

		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"transaction_details_failed",
				slog.String("operation", "transaction.get"),
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

	postings := make(
		[]transactionPostingResponse,
		0,
		len(details.JournalEntry.Postings),
	)
	for _, posting := range details.JournalEntry.Postings {
		postings = append(postings, transactionPostingResponse{
			ID:               posting.ID.String(),
			AccountID:        posting.AccountID.String(),
			Direction:        string(posting.Direction),
			AmountMinorUnits: posting.AmountMinorUnits,
		})
	}

	writeJSON(w, http.StatusOK, transactionDetailsResponse{
		ID:          details.ID.String(),
		Description: details.Description,
		CreatedAt:   details.CreatedAt,
		JournalEntry: transactionJournalEntryResponse{
			ID:       details.JournalEntry.ID.String(),
			Currency: details.JournalEntry.Currency,
			PostedAt: details.JournalEntry.PostedAt,
			Postings: postings,
		},
	})
}

func (h *TransactionHandler) Post(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idempotencyKey := r.Header.Get(idempotencyKeyHeader)
	if idempotencyKey == "" {
		writeError(
			w,
			http.StatusBadRequest,
			transaction.ErrEmptyIdempotencyKey.Error(),
		)
		return
	}

	defer r.Body.Close()

	decoder := json.NewDecoder(
		http.MaxBytesReader(w, r.Body, maxRequestBodySize),
	)
	decoder.DisallowUnknownFields()

	var request postTransactionRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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

	command, err := buildPostTransactionCommand(request, idempotencyKey)
	if err != nil {
		writeApplicationError(w, err)
		return
	}

	postedTransaction, err := h.postTransaction.Execute(
		r.Context(),
		command,
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"transaction_posting_failed",
				slog.String("operation", "transaction.post"),
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
			"transaction_posted",
			slog.String("operation", "transaction.post"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String("transaction_id", postedTransaction.ID().String()),
		)
	}

	writeJSON(
		w,
		http.StatusCreated,
		transactionResponse{
			ID:             postedTransaction.ID().String(),
			JournalEntryID: postedTransaction.JournalEntry().ID().String(),
			Description:    postedTransaction.Description(),
			Currency:       postedTransaction.JournalEntry().Currency().String(),
			Status:         "posted",
		},
	)
}

func buildPostTransactionCommand(
	request postTransactionRequest,
	idempotencyKey string,
) (transaction.PostTransactionCommand, error) {
	transactionID, err := domain.NewTransactionID(request.ID)
	if err != nil {
		return transaction.PostTransactionCommand{}, err
	}

	journalEntryID, err := domain.NewJournalEntryID(request.JournalEntryID)
	if err != nil {
		return transaction.PostTransactionCommand{}, err
	}

	postings := make(
		[]transaction.PostingCommand,
		0,
		len(request.Postings),
	)

	for _, posting := range request.Postings {
		postingID, err := domain.NewPostingID(posting.ID)
		if err != nil {
			return transaction.PostTransactionCommand{}, err
		}

		accountID, err := domain.NewAccountID(posting.AccountID)
		if err != nil {
			return transaction.PostTransactionCommand{}, err
		}

		postings = append(postings, transaction.PostingCommand{
			ID:               postingID,
			AccountID:        accountID,
			Direction:        domain.PostingDirection(posting.Direction),
			AmountMinorUnits: posting.AmountMinorUnits,
		})
	}

	canonicalPayload, err := json.Marshal(request)
	if err != nil {
		return transaction.PostTransactionCommand{}, err
	}

	hash := sha256.Sum256(canonicalPayload)

	return transaction.PostTransactionCommand{
		ID:             transactionID,
		JournalEntryID: journalEntryID,
		Description:    request.Description,
		CurrencyCode:   request.Currency,
		IdempotencyKey: idempotencyKey,
		RequestHash:    hex.EncodeToString(hash[:]),
		Postings:       postings,
	}, nil
}

func isTransactionClientError(err error) bool {
	return errors.Is(err, transaction.ErrEmptyIdempotencyKey) ||
		errors.Is(err, transaction.ErrEmptyRequestHash)
}
