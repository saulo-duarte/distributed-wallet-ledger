package httpadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"financial-ledger/internal/ledger/application/saga"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/httpx"
	"financial-ledger/internal/platform/observability"

	"github.com/google/uuid"
)

type PaymentHandler struct {
	orchestrator saga.PaymentSagaOrchestrator
	logger       *slog.Logger
}

func NewPaymentHandler(
	orchestrator saga.PaymentSagaOrchestrator,
	logger *slog.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

type processPaymentRequest struct {
	HoldID              string `json:"hold_id"`
	WalletID            string `json:"wallet_id"`
	SettlementAccountID string `json:"settlement_account_id"`
	TransactionID       string `json:"transaction_id"`
	JournalEntryID      string `json:"journal_entry_id"`
	WalletPostingID     string `json:"wallet_posting_id"`
	SettlementPostingID string `json:"settlement_posting_id"`
	AmountMinorUnits    int64  `json:"amount_minor_units"`
	Currency            string `json:"currency"`
	Recipient           string `json:"recipient"`
	Description         string `json:"description"`
	HoldExpiresInSec    int64  `json:"hold_expires_in_sec"`
}

type processPaymentResponse struct {
	HoldID               string `json:"hold_id"`
	TransactionID        string `json:"transaction_id"`
	GatewayTransactionID string `json:"gateway_transaction_id"`
	Status               string `json:"status"`
}

func (h *PaymentHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if idempotencyKey == "" {
		writeHTTPError(
			w,
			r,
			http.StatusBadRequest,
			"empty_idempotency_key",
			"Idempotency-Key header is required",
		)
		return
	}

	var req processPaymentRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		writeHTTPError(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"invalid request body",
		)
		return
	}

	currency, err := domain.NewCurrency(req.Currency)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	holdIDStr := req.HoldID
	if holdIDStr == "" {
		holdIDStr = deterministicPaymentID(idempotencyKey, "hold")
	}
	holdID, err := domain.NewHoldID(holdIDStr)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	walletID, err := domain.NewWalletID(req.WalletID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	settlementAccountID, err := domain.NewAccountID(req.SettlementAccountID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	txIDStr := req.TransactionID
	if txIDStr == "" {
		txIDStr = deterministicPaymentID(idempotencyKey, "transaction")
	}
	transactionID, err := domain.NewTransactionID(txIDStr)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	journalIDStr := req.JournalEntryID
	if journalIDStr == "" {
		journalIDStr = deterministicPaymentID(idempotencyKey, "journal")
	}
	journalEntryID, err := domain.NewJournalEntryID(journalIDStr)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	wPostingIDStr := req.WalletPostingID
	if wPostingIDStr == "" {
		wPostingIDStr = deterministicPaymentID(idempotencyKey, "wallet-posting")
	}
	walletPostingID, err := domain.NewPostingID(wPostingIDStr)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	sPostingIDStr := req.SettlementPostingID
	if sPostingIDStr == "" {
		sPostingIDStr = deterministicPaymentID(idempotencyKey, "settlement-posting")
	}
	settlementPostingID, err := domain.NewPostingID(sPostingIDStr)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	expiresInSec := req.HoldExpiresInSec
	if expiresInSec <= 0 {
		expiresInSec = 3600
	}
	expiresAt := time.Now().UTC().Add(time.Duration(expiresInSec) * time.Second)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	hash := sha256.Sum256(reqBytes)
	requestHash := hex.EncodeToString(hash[:])

	cmd := saga.ProcessPaymentCommand{
		HoldID:              holdID,
		WalletID:            walletID,
		SettlementAccountID: settlementAccountID,
		TransactionID:       transactionID,
		JournalEntryID:      journalEntryID,
		WalletPostingID:     walletPostingID,
		SettlementPostingID: settlementPostingID,
		AmountMinorUnits:    req.AmountMinorUnits,
		Currency:            currency,
		Recipient:           req.Recipient,
		Description:         req.Description,
		HoldExpiresAt:       expiresAt,
		IdempotencyKey:      idempotencyKey,
		RequestHash:         requestHash,
	}

	result, err := h.orchestrator.Execute(r.Context(), cmd)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"checkout_failed",
				slog.String("operation", "payments.checkout"),
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

	resp := processPaymentResponse{
		HoldID:               result.HoldID.String(),
		TransactionID:        result.TransactionID.String(),
		GatewayTransactionID: result.GatewayTransactionID,
		Status:               result.Status,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

// deterministicPaymentID keeps server-generated aggregate IDs stable across
// retries that reuse the same idempotency key. This is important because a
// checkout spans a hold, an external payment and a ledger capture.
func deterministicPaymentID(idempotencyKey, kind string) string {
	return uuid.NewSHA1(
		uuid.NameSpaceURL,
		[]byte("goledge/payment/"+kind+":"+idempotencyKey),
	).String()
}
