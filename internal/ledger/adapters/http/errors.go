package httpadapter

import (
	"errors"
	"net/http"

	"financial-ledger/internal/ledger/application/account"
	"financial-ledger/internal/ledger/application/transaction"
	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"
	"financial-ledger/internal/platform/httpx"
	"financial-ledger/internal/platform/observability"
)

type errorMapping struct {
	statusCode int
	code       string
}

func writeApplicationError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	mapping := mapApplicationError(err)
	message := err.Error()
	if mapping.statusCode >= http.StatusInternalServerError {
		message = "internal server error"
	}

	httpx.WriteError(
		w,
		mapping.statusCode,
		mapping.code,
		message,
		observability.RequestIDFromContext(r.Context()),
	)
}

func writeHTTPError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	code string,
	message string,
) {
	httpx.WriteError(
		w,
		statusCode,
		code,
		message,
		observability.RequestIDFromContext(r.Context()),
	)
}

func mapApplicationError(err error) errorMapping {
	switch {
	case errors.Is(err, transaction.ErrTransactionNotFound):
		return errorMapping{http.StatusNotFound, "transaction_not_found"}
	case errors.Is(err, wallet.ErrWalletNotFound):
		return errorMapping{http.StatusNotFound, "wallet_not_found"}
	case errors.Is(err, wallet.ErrInsufficientFunds):
		return errorMapping{http.StatusUnprocessableEntity, "insufficient_funds"}
	case errors.Is(err, transaction.ErrInsufficientBalance):
		return errorMapping{http.StatusUnprocessableEntity, "insufficient_funds"}
	case errors.Is(err, wallet.ErrWithdrawalSameAccount):
		return errorMapping{http.StatusBadRequest, "withdrawal_same_account"}
	case errors.Is(err, wallet.ErrTransferSameWallet):
		return errorMapping{http.StatusBadRequest, "transfer_same_wallet"}
	case errors.Is(err, wallet.ErrTransferSameAccount):
		return errorMapping{http.StatusBadRequest, "transfer_same_account"}
	case errors.Is(err, domain.ErrInvalidID):
		return errorMapping{http.StatusBadRequest, "invalid_id"}
	case errors.Is(err, domain.ErrInvalidCurrency):
		return errorMapping{http.StatusBadRequest, "invalid_currency"}
	case errors.Is(err, domain.ErrCurrencyMismatch):
		return errorMapping{http.StatusBadRequest, "currency_mismatch"}
	case errors.Is(err, domain.ErrAmountMustNotBeNegative):
		return errorMapping{http.StatusBadRequest, "negative_amount"}
	case errors.Is(err, domain.ErrAmountMustBePositive):
		return errorMapping{http.StatusBadRequest, "non_positive_amount"}
	case errors.Is(err, domain.ErrAmountOverflow):
		return errorMapping{http.StatusBadRequest, "amount_overflow"}
	case errors.Is(err, domain.ErrInvalidPostingDirection):
		return errorMapping{http.StatusBadRequest, "invalid_posting_direction"}
	case errors.Is(err, domain.ErrJournalEntryWithoutPostings):
		return errorMapping{http.StatusBadRequest, "journal_entry_without_postings"}
	case errors.Is(err, domain.ErrJournalEntryWithoutDebit):
		return errorMapping{http.StatusBadRequest, "journal_entry_without_debit"}
	case errors.Is(err, domain.ErrJournalEntryWithoutCredit):
		return errorMapping{http.StatusBadRequest, "journal_entry_without_credit"}
	case errors.Is(err, domain.ErrUnbalancedJournalEntry):
		return errorMapping{http.StatusBadRequest, "unbalanced_journal_entry"}
	case errors.Is(err, domain.ErrEmptyTransactionDescription):
		return errorMapping{http.StatusBadRequest, "empty_transaction_description"}
	case errors.Is(err, domain.ErrCannotReverseReversal):
		return errorMapping{http.StatusBadRequest, "cannot_reverse_reversal"}
	case errors.Is(err, domain.ErrReversalSameTransactionID):
		return errorMapping{http.StatusBadRequest, "reversal_same_transaction_id"}
	case errors.Is(err, domain.ErrInvalidReversalPostingIDCount):
		return errorMapping{http.StatusBadRequest, "invalid_reversal_posting_count"}
	case errors.Is(err, domain.ErrEmptyAccountCode):
		return errorMapping{http.StatusBadRequest, "empty_account_code"}
	case errors.Is(err, domain.ErrEmptyAccountName):
		return errorMapping{http.StatusBadRequest, "empty_account_name"}
	case errors.Is(err, domain.ErrInvalidAccount):
		return errorMapping{http.StatusBadRequest, "invalid_account"}
	case errors.Is(err, domain.ErrInvalidAccountStatus):
		return errorMapping{http.StatusBadRequest, "invalid_account_status"}
	case errors.Is(err, domain.ErrInvalidWalletStatus):
		return errorMapping{http.StatusBadRequest, "invalid_wallet_status"}
	case errors.Is(err, domain.ErrEmptyWalletOwnerID):
		return errorMapping{http.StatusBadRequest, "empty_wallet_owner_id"}
	case errors.Is(err, domain.ErrWalletSuspended):
		return errorMapping{http.StatusConflict, "wallet_suspended"}
	case errors.Is(err, domain.ErrWalletClosed):
		return errorMapping{http.StatusConflict, "wallet_closed"}
	case errors.Is(err, domain.ErrWalletAlreadySuspended):
		return errorMapping{http.StatusConflict, "wallet_already_suspended"}
	case errors.Is(err, domain.ErrWalletAlreadyClosed):
		return errorMapping{http.StatusConflict, "wallet_already_closed"}
	case errors.Is(err, account.ErrInvalidPageSize):
		return errorMapping{http.StatusBadRequest, "invalid_page_size"}
	case errors.Is(err, transaction.ErrEmptyIdempotencyKey):
		return errorMapping{http.StatusBadRequest, "empty_idempotency_key"}
	case errors.Is(err, transaction.ErrEmptyRequestHash):
		return errorMapping{http.StatusBadRequest, "empty_request_hash"}
	case errors.Is(err, transaction.ErrIdempotencyKeyConflict):
		return errorMapping{http.StatusConflict, "idempotency_key_conflict"}
	case errors.Is(err, httpx.ErrInvalidRequestBody):
		return errorMapping{http.StatusBadRequest, "invalid_request"}
	case errors.Is(err, httpx.ErrMultipleJSONValues):
		return errorMapping{http.StatusBadRequest, "multiple_json_values"}
	default:
		return errorMapping{http.StatusInternalServerError, "internal_error"}
	}
}

func isClientError(err error) bool {
	return mapApplicationError(err).statusCode < http.StatusInternalServerError
}
