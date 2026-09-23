package httpadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/httpx"
	"financial-ledger/internal/platform/observability"

	"github.com/go-chi/chi/v5"
)

type WalletHandler struct {
	createWallet    wallet.CreateWalletUseCase
	getWallet       wallet.GetWalletUseCase
	listWallets     wallet.ListWalletsByOwnerUseCase
	getBalance      wallet.GetWalletBalanceUseCase
	depositWallet   wallet.DepositWalletUseCase
	withdrawWallet  wallet.WithdrawWalletUseCase
	transferWallet  wallet.TransferWalletUseCase
	createHold      wallet.CreateHoldUseCase
	releaseHold     wallet.ReleaseHoldUseCase
	expireHold      wallet.ExpireHoldUseCase
	captureHold     wallet.CaptureHoldUseCase
	queriesEnabled  bool
	balanceEnabled  bool
	depositEnabled  bool
	withdrawEnabled bool
	transferEnabled bool
	holdEnabled     bool
	logger          *slog.Logger
}

func NewWalletHandler(
	createWallet wallet.CreateWalletUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		createWallet:    createWallet,
		queriesEnabled:  false,
		balanceEnabled:  false,
		depositEnabled:  false,
		withdrawEnabled: false,
		transferEnabled: false,
		holdEnabled:     false,
		logger:          logger,
	}
}

func NewWalletHandlerWithQueries(
	createWallet wallet.CreateWalletUseCase,
	getWallet wallet.GetWalletUseCase,
	listWallets wallet.ListWalletsByOwnerUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		createWallet:    createWallet,
		getWallet:       getWallet,
		listWallets:     listWallets,
		queriesEnabled:  true,
		balanceEnabled:  false,
		depositEnabled:  false,
		withdrawEnabled: false,
		transferEnabled: false,
		holdEnabled:     false,
		logger:          logger,
	}
}

func NewWalletHandlerWithBalance(
	createWallet wallet.CreateWalletUseCase,
	getWallet wallet.GetWalletUseCase,
	listWallets wallet.ListWalletsByOwnerUseCase,
	getBalance wallet.GetWalletBalanceUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		createWallet:    createWallet,
		getWallet:       getWallet,
		listWallets:     listWallets,
		getBalance:      getBalance,
		queriesEnabled:  true,
		balanceEnabled:  true,
		depositEnabled:  false,
		withdrawEnabled: false,
		transferEnabled: false,
		holdEnabled:     false,
		logger:          logger,
	}
}

func NewWalletHandlerWithWithdrawal(
	createWallet wallet.CreateWalletUseCase,
	getWallet wallet.GetWalletUseCase,
	listWallets wallet.ListWalletsByOwnerUseCase,
	getBalance wallet.GetWalletBalanceUseCase,
	withdrawWallet wallet.WithdrawWalletUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		createWallet:    createWallet,
		getWallet:       getWallet,
		listWallets:     listWallets,
		getBalance:      getBalance,
		withdrawWallet:  withdrawWallet,
		queriesEnabled:  true,
		balanceEnabled:  true,
		depositEnabled:  false,
		withdrawEnabled: true,
		transferEnabled: false,
		holdEnabled:     false,
		logger:          logger,
	}
}

func NewWalletHandlerWithTransfer(
	createWallet wallet.CreateWalletUseCase,
	getWallet wallet.GetWalletUseCase,
	listWallets wallet.ListWalletsByOwnerUseCase,
	getBalance wallet.GetWalletBalanceUseCase,
	withdrawWallet wallet.WithdrawWalletUseCase,
	transferWallet wallet.TransferWalletUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return NewWalletHandlerWithOperations(
		createWallet,
		getWallet,
		listWallets,
		getBalance,
		wallet.DepositWalletUseCase{},
		withdrawWallet,
		transferWallet,
		logger,
	)
}

func NewWalletHandlerWithOperations(
	createWallet wallet.CreateWalletUseCase,
	getWallet wallet.GetWalletUseCase,
	listWallets wallet.ListWalletsByOwnerUseCase,
	getBalance wallet.GetWalletBalanceUseCase,
	depositWallet wallet.DepositWalletUseCase,
	withdrawWallet wallet.WithdrawWalletUseCase,
	transferWallet wallet.TransferWalletUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		createWallet:    createWallet,
		getWallet:       getWallet,
		listWallets:     listWallets,
		getBalance:      getBalance,
		depositWallet:   depositWallet,
		withdrawWallet:  withdrawWallet,
		transferWallet:  transferWallet,
		queriesEnabled:  true,
		balanceEnabled:  true,
		depositEnabled:  true,
		withdrawEnabled: true,
		transferEnabled: true,
		holdEnabled:     false,
		logger:          logger,
	}
}

func NewWalletHandlerWithHolds(
	createWallet wallet.CreateWalletUseCase,
	getWallet wallet.GetWalletUseCase,
	listWallets wallet.ListWalletsByOwnerUseCase,
	getBalance wallet.GetWalletBalanceUseCase,
	depositWallet wallet.DepositWalletUseCase,
	withdrawWallet wallet.WithdrawWalletUseCase,
	transferWallet wallet.TransferWalletUseCase,
	createHold wallet.CreateHoldUseCase,
	releaseHold wallet.ReleaseHoldUseCase,
	expireHold wallet.ExpireHoldUseCase,
	captureHold wallet.CaptureHoldUseCase,
	logger *slog.Logger,
) *WalletHandler {
	return &WalletHandler{
		createWallet:    createWallet,
		getWallet:       getWallet,
		listWallets:     listWallets,
		getBalance:      getBalance,
		depositWallet:   depositWallet,
		withdrawWallet:  withdrawWallet,
		transferWallet:  transferWallet,
		createHold:      createHold,
		releaseHold:     releaseHold,
		expireHold:      expireHold,
		captureHold:     captureHold,
		queriesEnabled:  true,
		balanceEnabled:  true,
		depositEnabled:  true,
		withdrawEnabled: true,
		transferEnabled: true,
		holdEnabled:     true,
		logger:          logger,
	}
}

type createWalletRequest struct {
	ID              string `json:"id"`
	OwnerID         string `json:"owner_id"`
	LedgerAccountID string `json:"ledger_account_id"`
	CurrencyCode    string `json:"currency"`
}

type walletResponse struct {
	ID              string `json:"id"`
	OwnerID         string `json:"owner_id"`
	LedgerAccountID string `json:"ledger_account_id"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
}

type walletBalanceResponse struct {
	WalletID                   string `json:"wallet_id"`
	Currency                   string `json:"currency"`
	LedgerBalanceMinorUnits    int64  `json:"ledger_balance_minor_units"`
	AvailableBalanceMinorUnits int64  `json:"available_balance_minor_units"`
}

type withdrawWalletRequest struct {
	ClearingAccountID string `json:"clearing_account_id"`
	TransactionID     string `json:"transaction_id"`
	JournalEntryID    string `json:"journal_entry_id"`
	WalletPostingID   string `json:"wallet_posting_id"`
	ClearingPostingID string `json:"clearing_posting_id"`
	AmountMinorUnits  int64  `json:"amount_minor_units"`
	Description       string `json:"description"`
}

type depositWalletRequest struct {
	ClearingAccountID string `json:"clearing_account_id"`
	TransactionID     string `json:"transaction_id"`
	JournalEntryID    string `json:"journal_entry_id"`
	ClearingPostingID string `json:"clearing_posting_id"`
	WalletPostingID   string `json:"wallet_posting_id"`
	AmountMinorUnits  int64  `json:"amount_minor_units"`
	Description       string `json:"description"`
}

type transferWalletRequest struct {
	DestinationWalletID  string `json:"destination_wallet_id"`
	TransactionID        string `json:"transaction_id"`
	JournalEntryID       string `json:"journal_entry_id"`
	SourcePostingID      string `json:"source_posting_id"`
	DestinationPostingID string `json:"destination_posting_id"`
	AmountMinorUnits     int64  `json:"amount_minor_units"`
	Description          string `json:"description"`
}

type createHoldRequest struct {
	ID               string    `json:"id"`
	AmountMinorUnits int64     `json:"amount_minor_units"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type captureHoldRequest struct {
	SettlementAccountID string `json:"settlement_account_id"`
	TransactionID       string `json:"transaction_id"`
	JournalEntryID      string `json:"journal_entry_id"`
	WalletPostingID     string `json:"wallet_posting_id"`
	SettlementPostingID string `json:"settlement_posting_id"`
	Description         string `json:"description"`
}

type holdResponse struct {
	ID               string    `json:"id"`
	WalletID         string    `json:"wallet_id"`
	AmountMinorUnits int64     `json:"amount_minor_units"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type captureHoldResponse struct {
	Hold        holdResponse        `json:"hold"`
	Transaction transactionResponse `json:"transaction"`
}

func (h *WalletHandler) GetBalance(
	w http.ResponseWriter,
	r *http.Request,
) {
	walletID, err := domain.NewWalletID(chi.URLParam(r, "walletID"))
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	var balance wallet.WalletBalance
	if r.URL.Query().Get("consistency") == "strong" {
		balance, err = h.getBalance.ExecuteFromLedger(r.Context(), walletID)
	} else {
		balance, err = h.getBalance.Execute(r.Context(), walletID)
	}
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_balance_failed",
				slog.String("operation", "wallet.balance"),
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

	httpx.WriteJSON(w, http.StatusOK, walletBalanceResponse{
		WalletID:                   balance.WalletID.String(),
		Currency:                   balance.Currency.String(),
		LedgerBalanceMinorUnits:    balance.LedgerBalanceMinorUnits,
		AvailableBalanceMinorUnits: balance.AvailableBalanceMinorUnits,
	})
}

func (h *WalletHandler) CreateHold(
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

	walletID, err := domain.NewWalletID(chi.URLParam(r, "walletID"))
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
			"idempotency key cannot be empty",
		)
		return
	}

	var request createHoldRequest
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

	holdID, err := domain.NewHoldID(request.ID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	hash := sha256.Sum256(requestBytes)

	createdHold, err := h.createHold.Execute(
		r.Context(),
		wallet.CreateHoldCommand{
			ID:               holdID,
			WalletID:         walletID,
			AmountMinorUnits: request.AmountMinorUnits,
			ExpiresAt:        request.ExpiresAt,
			IdempotencyKey:   idempotencyKey,
			RequestHash:      hex.EncodeToString(hash[:]),
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_hold_creation_failed",
				slog.String("operation", "wallet.hold.create"),
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

	httpx.WriteJSON(w, http.StatusCreated, toHoldResponse(createdHold))
}

func (h *WalletHandler) ReleaseHold(
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

	holdID, err := domain.NewHoldID(chi.URLParam(r, "holdID"))
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	releasedHold, err := h.releaseHold.Execute(r.Context(), holdID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toHoldResponse(releasedHold))
}

func (h *WalletHandler) ExpireHold(
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

	holdID, err := domain.NewHoldID(chi.URLParam(r, "holdID"))
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	expiredHold, err := h.expireHold.Execute(r.Context(), holdID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toHoldResponse(expiredHold))
}

func (h *WalletHandler) CaptureHold(
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

	holdID, err := domain.NewHoldID(chi.URLParam(r, "holdID"))
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
			"idempotency key cannot be empty",
		)
		return
	}

	var request captureHoldRequest
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

	settlementAccountID, err := domain.NewAccountID(
		request.SettlementAccountID,
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	transactionID, err := domain.NewTransactionID(request.TransactionID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	journalEntryID, err := domain.NewJournalEntryID(request.JournalEntryID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	walletPostingID, err := domain.NewPostingID(request.WalletPostingID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	settlementPostingID, err := domain.NewPostingID(
		request.SettlementPostingID,
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	hash := sha256.Sum256(requestBytes)

	result, err := h.captureHold.Execute(
		r.Context(),
		wallet.CaptureHoldCommand{
			HoldID:              holdID,
			SettlementAccountID: settlementAccountID,
			TransactionID:       transactionID,
			JournalEntryID:      journalEntryID,
			WalletPostingID:     walletPostingID,
			SettlementPostingID: settlementPostingID,
			Description:         request.Description,
			IdempotencyKey:      idempotencyKey,
			RequestHash:         hex.EncodeToString(hash[:]),
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_hold_capture_failed",
				slog.String("operation", "wallet.hold.capture"),
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

	httpx.WriteJSON(w, http.StatusOK, captureHoldResponse{
		Hold: toHoldResponse(result.Hold),
		Transaction: transactionResponse{
			ID:             result.Transaction.ID().String(),
			JournalEntryID: result.Transaction.JournalEntry().ID().String(),
			Description:    result.Transaction.Description(),
			Currency:       result.Transaction.JournalEntry().Currency().String(),
			Status:         "posted",
		},
	})
}

func (h *WalletHandler) Withdraw(
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

	walletID, err := domain.NewWalletID(chi.URLParam(r, "walletID"))
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
			"idempotency key cannot be empty",
		)
		return
	}

	var request withdrawWalletRequest
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

	clearingAccountID, err := domain.NewAccountID(request.ClearingAccountID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	transactionID, err := domain.NewTransactionID(request.TransactionID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	journalEntryID, err := domain.NewJournalEntryID(request.JournalEntryID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	walletPostingID, err := domain.NewPostingID(request.WalletPostingID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	clearingPostingID, err := domain.NewPostingID(request.ClearingPostingID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	hash := sha256.Sum256(requestBytes)

	postedTransaction, err := h.withdrawWallet.Execute(
		r.Context(),
		wallet.WithdrawWalletCommand{
			WalletID:          walletID,
			ClearingAccountID: clearingAccountID,
			TransactionID:     transactionID,
			JournalEntryID:    journalEntryID,
			WalletPostingID:   walletPostingID,
			ClearingPostingID: clearingPostingID,
			AmountMinorUnits:  request.AmountMinorUnits,
			Description:       request.Description,
			IdempotencyKey:    idempotencyKey,
			RequestHash:       hex.EncodeToString(hash[:]),
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_withdrawal_failed",
				slog.String("operation", "wallet.withdraw"),
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
			"wallet_withdrawal_posted",
			slog.String("operation", "wallet.withdraw"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String("wallet_id", walletID.String()),
			slog.String("transaction_id", postedTransaction.ID().String()),
		)
	}

	httpx.WriteJSON(
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

func (h *WalletHandler) Deposit(
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

	walletID, err := domain.NewWalletID(chi.URLParam(r, "walletID"))
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
			"idempotency key cannot be empty",
		)
		return
	}

	var request depositWalletRequest
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

	clearingAccountID, err := domain.NewAccountID(request.ClearingAccountID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	transactionID, err := domain.NewTransactionID(request.TransactionID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	journalEntryID, err := domain.NewJournalEntryID(request.JournalEntryID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	clearingPostingID, err := domain.NewPostingID(request.ClearingPostingID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	walletPostingID, err := domain.NewPostingID(request.WalletPostingID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	hash := sha256.Sum256(requestBytes)

	postedTransaction, err := h.depositWallet.Execute(
		r.Context(),
		wallet.DepositWalletCommand{
			WalletID:          walletID,
			ClearingAccountID: clearingAccountID,
			TransactionID:     transactionID,
			JournalEntryID:    journalEntryID,
			ClearingPostingID: clearingPostingID,
			WalletPostingID:   walletPostingID,
			AmountMinorUnits:  request.AmountMinorUnits,
			Description:       request.Description,
			IdempotencyKey:    idempotencyKey,
			RequestHash:       hex.EncodeToString(hash[:]),
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_deposit_failed",
				slog.String("operation", "wallet.deposit"),
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
			"wallet_deposit_posted",
			slog.String("operation", "wallet.deposit"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String("wallet_id", walletID.String()),
			slog.String("transaction_id", postedTransaction.ID().String()),
		)
	}

	httpx.WriteJSON(
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

func (h *WalletHandler) Transfer(
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

	sourceWalletID, err := domain.NewWalletID(
		chi.URLParam(r, "walletID"),
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
			"idempotency key cannot be empty",
		)
		return
	}

	var request transferWalletRequest
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

	destinationWalletID, err := domain.NewWalletID(
		request.DestinationWalletID,
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	transactionID, err := domain.NewTransactionID(request.TransactionID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	journalEntryID, err := domain.NewJournalEntryID(request.JournalEntryID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	sourcePostingID, err := domain.NewPostingID(request.SourcePostingID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	destinationPostingID, err := domain.NewPostingID(
		request.DestinationPostingID,
	)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}
	hash := sha256.Sum256(requestBytes)

	postedTransaction, err := h.transferWallet.Execute(
		r.Context(),
		wallet.TransferWalletCommand{
			SourceWalletID:       sourceWalletID,
			DestinationWalletID:  destinationWalletID,
			TransactionID:        transactionID,
			JournalEntryID:       journalEntryID,
			SourcePostingID:      sourcePostingID,
			DestinationPostingID: destinationPostingID,
			AmountMinorUnits:     request.AmountMinorUnits,
			Description:          request.Description,
			IdempotencyKey:       idempotencyKey,
			RequestHash:          hex.EncodeToString(hash[:]),
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_transfer_failed",
				slog.String("operation", "wallet.transfer"),
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
			"wallet_transfer_posted",
			slog.String("operation", "wallet.transfer"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String("source_wallet_id", sourceWalletID.String()),
			slog.String(
				"destination_wallet_id",
				destinationWalletID.String(),
			),
			slog.String("transaction_id", postedTransaction.ID().String()),
		)
	}

	httpx.WriteJSON(
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

func (h *WalletHandler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	walletID, err := domain.NewWalletID(chi.URLParam(r, "walletID"))
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	foundWallet, err := h.getWallet.Execute(r.Context(), walletID)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_get_failed",
				slog.String("operation", "wallet.get"),
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

	httpx.WriteJSON(w, http.StatusOK, toWalletResponse(foundWallet))
}

func (h *WalletHandler) ListByOwner(
	w http.ResponseWriter,
	r *http.Request,
) {
	ownerID, err := domain.NewOwnerID(chi.URLParam(r, "ownerID"))
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	foundWallets, err := h.listWallets.Execute(r.Context(), ownerID)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_listing_failed",
				slog.String("operation", "wallet.list_by_owner"),
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

	response := make([]walletResponse, 0, len(foundWallets))
	for _, foundWallet := range foundWallets {
		response = append(response, toWalletResponse(foundWallet))
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *WalletHandler) Create(
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

	var request createWalletRequest
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

	walletID, err := domain.NewWalletID(request.ID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	ownerID, err := domain.NewOwnerID(request.OwnerID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	ledgerAccountID, err := domain.NewAccountID(request.LedgerAccountID)
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	createdWallet, err := h.createWallet.Execute(
		r.Context(),
		wallet.CreateWalletCommand{
			ID:              walletID,
			OwnerID:         ownerID,
			LedgerAccountID: ledgerAccountID,
			CurrencyCode:    request.CurrencyCode,
		},
	)
	if err != nil {
		if !isClientError(err) && h.logger != nil {
			h.logger.ErrorContext(
				r.Context(),
				"wallet_creation_failed",
				slog.String("operation", "wallet.create"),
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
			"wallet_created",
			slog.String("operation", "wallet.create"),
			slog.String(
				"request_id",
				observability.RequestIDFromContext(r.Context()),
			),
			slog.String("wallet_id", createdWallet.ID().String()),
			slog.String("owner_id", createdWallet.OwnerID().String()),
			slog.String("currency", createdWallet.Currency().String()),
		)
	}

	httpx.WriteJSON(
		w,
		http.StatusCreated,
		walletResponse{
			ID:              createdWallet.ID().String(),
			OwnerID:         createdWallet.OwnerID().String(),
			LedgerAccountID: createdWallet.LedgerAccountID().String(),
			Currency:        createdWallet.Currency().String(),
			Status:          string(createdWallet.Status()),
		},
	)
}

func toWalletResponse(foundWallet domain.Wallet) walletResponse {
	return walletResponse{
		ID:              foundWallet.ID().String(),
		OwnerID:         foundWallet.OwnerID().String(),
		LedgerAccountID: foundWallet.LedgerAccountID().String(),
		Currency:        foundWallet.Currency().String(),
		Status:          string(foundWallet.Status()),
	}
}

func toHoldResponse(foundHold domain.Hold) holdResponse {
	return holdResponse{
		ID:               foundHold.ID().String(),
		WalletID:         foundHold.WalletID().String(),
		AmountMinorUnits: foundHold.Amount().AmountMinorUnits(),
		Currency:         foundHold.Amount().Currency().String(),
		Status:           string(foundHold.Status()),
		ExpiresAt:        foundHold.ExpiresAt(),
	}
}
