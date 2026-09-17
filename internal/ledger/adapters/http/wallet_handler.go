package httpadapter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

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
	withdrawWallet  wallet.WithdrawWalletUseCase
	queriesEnabled  bool
	balanceEnabled  bool
	withdrawEnabled bool
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
		withdrawEnabled: false,
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
		withdrawEnabled: false,
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
		withdrawEnabled: false,
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
		withdrawEnabled: true,
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

func (h *WalletHandler) GetBalance(
	w http.ResponseWriter,
	r *http.Request,
) {
	walletID, err := domain.NewWalletID(chi.URLParam(r, "walletID"))
	if err != nil {
		writeApplicationError(w, r, err)
		return
	}

	balance, err := h.getBalance.Execute(r.Context(), walletID)
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
