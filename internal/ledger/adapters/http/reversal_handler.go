package httpadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/httpx"
	"financial-ledger/internal/platform/observability"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ReversalHandler struct {
	reverseTransaction transaction.ReverseTransactionUseCase
	logger             *slog.Logger
}

func NewReversalHandler(
	reverseTransaction transaction.ReverseTransactionUseCase,
	logger *slog.Logger,
) *ReversalHandler {
	return &ReversalHandler{
		reverseTransaction: reverseTransaction,
		logger:             logger,
	}
}

type reverseTransactionRequest struct {
	ID             string   `json:"id"`
	JournalEntryID string   `json:"journal_entry_id"`
	PostingIDs     []string `json:"posting_ids"`
	Description    string   `json:"description"`
}

type reversalHashPayload struct {
	OriginalTransactionID string                    `json:"original_transaction_id"`
	Request               reverseTransactionRequest `json:"request"`
}

type reversalResponse struct {
	ID                    string `json:"id"`
	JournalEntryID        string `json:"journal_entry_id"`
	ReversedTransactionID string `json:"reversed_transaction_id"`
	Description           string `json:"description"`
	Currency              string `json:"currency"`
	Status                string `json:"status"`
}

func (h *ReversalHandler) Reverse(
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

	originalTransactionID, err := domain.NewTransactionID(
		chi.URLParam(r, "transactionID"),
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	idempotencyKey := r.Header.Get(idempotencyKeyHeader)
	if idempotencyKey == "" {
		writeHTTPError(
			w,
			r,
			http.StatusBadRequest,
			"empty_idempotency_key",
			transaction.ErrEmptyIdempotencyKey.Error(),
		)
		return
	}

	var request reverseTransactionRequest

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

	command, err := buildReverseTransactionCommand(
		originalTransactionID,
		request,
		idempotencyKey,
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	reversedTransaction, err := h.reverseTransaction.Execute(
		r.Context(),
		command,
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"transaction_reversal_failed",
				slog.String("operation", "transaction.reverse"),
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

	originalID := reversedTransaction.ReversesTransactionID()

	reversedTransactionID := ""
	if originalID != nil {
		reversedTransactionID = originalID.String()
	}

	if h.logger != nil {
		h.logger.InfoContext(
			r.Context(),
			"transaction_reversed",
			slog.String("operation", "transaction.reverse"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String(
				"transaction_id",
				reversedTransaction.ID().String(),
			),
			slog.String(
				"reversed_transaction_id",
				reversedTransactionID,
			),
		)
	}

	httpx.WriteJSON(
		w,
		http.StatusCreated,
		reversalResponse{
			ID:                    reversedTransaction.ID().String(),
			JournalEntryID:        reversedTransaction.JournalEntry().ID().String(),
			ReversedTransactionID: reversedTransactionID,
			Description:           reversedTransaction.Description(),
			Currency:              reversedTransaction.JournalEntry().Currency().String(),
			Status:                "posted",
		},
	)
}

func buildReverseTransactionCommand(
	originalTransactionID domain.TransactionID,
	request reverseTransactionRequest,
	idempotencyKey string,
) (transaction.ReverseTransactionCommand, error) {
	newTransactionID, err := domain.NewTransactionID(request.ID)
	if err != nil {
		return transaction.ReverseTransactionCommand{}, err
	}

	newJournalEntryID, err := domain.NewJournalEntryID(
		request.JournalEntryID,
	)
	if err != nil {
		return transaction.ReverseTransactionCommand{}, err
	}

	newPostingIDs := make(
		[]domain.PostingID,
		0,
		len(request.PostingIDs),
	)

	for _, rawPostingID := range request.PostingIDs {
		postingID, err := domain.NewPostingID(rawPostingID)
		if err != nil {
			return transaction.ReverseTransactionCommand{}, err
		}

		newPostingIDs = append(newPostingIDs, postingID)
	}

	hashPayload := reversalHashPayload{
		OriginalTransactionID: originalTransactionID.String(),
		Request:               request,
	}

	canonicalPayload, err := json.Marshal(hashPayload)
	if err != nil {
		return transaction.ReverseTransactionCommand{}, err
	}

	hash := sha256.Sum256(canonicalPayload)

	return transaction.ReverseTransactionCommand{
		OriginalTransactionID: originalTransactionID,
		NewTransactionID:      newTransactionID,
		NewJournalEntryID:     newJournalEntryID,
		NewPostingIDs:         newPostingIDs,
		Description:           request.Description,
		IdempotencyKey:        idempotencyKey,
		RequestHash:           hex.EncodeToString(hash[:]),
	}, nil
}
