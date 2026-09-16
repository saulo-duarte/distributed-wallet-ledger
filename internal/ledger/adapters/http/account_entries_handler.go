package httpadapter

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/httpx"
	"financial-ledger/internal/platform/observability"

	"github.com/go-chi/chi/v5"
)

const (
	accountEntriesLimitQuery  = "limit"
	accountEntriesCursorQuery = "cursor"
)

type AccountEntriesHandler struct {
	listEntries account.ListAccountEntriesUseCase
	logger      *slog.Logger
}

func NewAccountEntriesHandler(
	listEntries account.ListAccountEntriesUseCase,
	logger *slog.Logger,
) *AccountEntriesHandler {
	return &AccountEntriesHandler{
		listEntries: listEntries,
		logger:      logger,
	}
}

type accountEntriesResponse struct {
	Items      []accountEntryResponse `json:"items"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

type accountEntryResponse struct {
	PostingID        string    `json:"posting_id"`
	JournalEntryID   string    `json:"journal_entry_id"`
	TransactionID    string    `json:"transaction_id"`
	Description      string    `json:"description"`
	Currency         string    `json:"currency"`
	Direction        string    `json:"direction"`
	AmountMinorUnits int64     `json:"amount_minor_units"`
	CreatedAt        time.Time `json:"created_at"`
}

func (h *AccountEntriesHandler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	accountID, err := domain.NewAccountID(
		chi.URLParam(r, "accountID"),
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	limit, err := parseAccountEntriesLimit(r.URL.Query().Get(accountEntriesLimitQuery))
	if err != nil {
		writeHTTPError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_page_size",
			err.Error(),
		)
		return
	}

	cursor, err := decodeAccountEntriesCursor(
		r.URL.Query().Get(accountEntriesCursorQuery),
	)
	if err != nil {
		writeHTTPError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_cursor",
			"invalid cursor",
		)
		return
	}

	page, err := h.listEntries.Execute(
		r.Context(),
		account.ListAccountEntriesCommand{
			AccountID: accountID,
			Limit:     limit,
			Cursor:    cursor,
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"account_entries_listing_failed",
				slog.String("operation", "account.entries.list"),
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

	response := accountEntriesResponse{
		Items: make([]accountEntryResponse, 0, len(page.Entries)),
	}

	for _, entry := range page.Entries {
		response.Items = append(response.Items, accountEntryResponse{
			PostingID:        entry.PostingID.String(),
			JournalEntryID:   entry.JournalEntryID.String(),
			TransactionID:    entry.TransactionID.String(),
			Description:      entry.Description,
			Currency:         entry.Currency.String(),
			Direction:        string(entry.Direction),
			AmountMinorUnits: entry.AmountMinor,
			CreatedAt:        entry.CreatedAt,
		})
	}

	if page.NextCursor != nil {
		response.NextCursor, err = encodeAccountEntriesCursor(*page.NextCursor)
		if err != nil {
			writeApplicationError(w, r, err)
			return
		}
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func parseAccountEntriesLimit(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, account.ErrInvalidPageSize
	}

	return limit, nil
}

type accountEntriesCursorPayload struct {
	CreatedAt string `json:"created_at"`
	PostingID string `json:"posting_id"`
}

func encodeAccountEntriesCursor(
	cursor account.AccountEntriesCursor,
) (string, error) {
	payload, err := json.Marshal(accountEntriesCursorPayload{
		CreatedAt: cursor.CreatedAt.UTC().Format(time.RFC3339Nano),
		PostingID: cursor.PostingID.String(),
	})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeAccountEntriesCursor(
	value string,
) (*account.AccountEntriesCursor, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	var decoded accountEntriesCursorPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	createdAt, err := time.Parse(time.RFC3339Nano, decoded.CreatedAt)
	if err != nil {
		return nil, err
	}

	postingID, err := domain.NewPostingID(decoded.PostingID)
	if err != nil {
		return nil, err
	}

	return &account.AccountEntriesCursor{
		CreatedAt: createdAt,
		PostingID: postingID,
	}, nil
}
