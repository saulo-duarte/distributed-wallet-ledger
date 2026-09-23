package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type TestClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 500,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

type TestContext struct {
	AliceWalletID       string
	BobWalletID         string
	ClearingAccountID   string
	SettlementAccountID string
}

func (c *TestClient) SetupContext(ctx context.Context) (*TestContext, error) {
	clearingID := uuid.NewString()
	_ = c.createAccount(ctx, clearingID, fmt.Sprintf("CLR-%s", clearingID[:6]), "Clearing Vault")

	settlementID := uuid.NewString()
	_ = c.createAccount(ctx, settlementID, fmt.Sprintf("SET-%s", settlementID[:6]), "Settlement Account")

	aliceAccID := uuid.NewString()
	_ = c.createAccount(ctx, aliceAccID, fmt.Sprintf("ACC-A-%s", aliceAccID[:6]), "Alice Ledger Account")

	bobAccID := uuid.NewString()
	_ = c.createAccount(ctx, bobAccID, fmt.Sprintf("ACC-B-%s", bobAccID[:6]), "Bob Ledger Account")

	aliceWalletID := uuid.NewString()
	if err := c.createWallet(ctx, aliceWalletID, uuid.NewString(), aliceAccID); err != nil {
		return nil, fmt.Errorf("create Alice wallet: %w", err)
	}

	bobWalletID := uuid.NewString()
	if err := c.createWallet(ctx, bobWalletID, uuid.NewString(), bobAccID); err != nil {
		return nil, fmt.Errorf("create Bob wallet: %w", err)
	}

	if err := c.deposit(ctx, aliceWalletID, clearingID, 100000000); err != nil {
		return nil, fmt.Errorf("deposit Alice: %w", err)
	}
	if err := c.deposit(ctx, bobWalletID, clearingID, 100000000); err != nil {
		return nil, fmt.Errorf("deposit Bob: %w", err)
	}

	return &TestContext{
		AliceWalletID:       aliceWalletID,
		BobWalletID:         bobWalletID,
		ClearingAccountID:   clearingID,
		SettlementAccountID: settlementID,
	}, nil
}

func (c *TestClient) createAccount(ctx context.Context, id, code, name string) error {
	url := fmt.Sprintf("%s/accounts", c.baseURL)
	payload := map[string]string{
		"id":       id,
		"code":     code,
		"name":     name,
		"currency": "BRL",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *TestClient) createWallet(ctx context.Context, walletID, ownerID, ledgerAccountID string) error {
	url := fmt.Sprintf("%s/wallets", c.baseURL)
	payload := map[string]string{
		"id":                walletID,
		"owner_id":          ownerID,
		"ledger_account_id": ledgerAccountID,
		"currency":          "BRL",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create wallet returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *TestClient) deposit(ctx context.Context, walletID, clearingAccountID string, amount int64) error {
	url := fmt.Sprintf("%s/wallets/%s/deposits", c.baseURL, walletID)
	payload := map[string]any{
		"clearing_account_id": clearingAccountID,
		"transaction_id":     uuid.NewString(),
		"journal_entry_id":    uuid.NewString(),
		"clearing_posting_id": uuid.NewString(),
		"wallet_posting_id":   uuid.NewString(),
		"amount_minor_units":  amount,
		"description":         "Initial deposit",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deposit returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *TestClient) ExecuteTransfer(ctx context.Context, tCtx *TestContext) Result {
	start := time.Now()
	from := tCtx.AliceWalletID
	to := tCtx.BobWalletID
	if rand.Intn(2) == 0 {
		from, to = to, from
	}

	url := fmt.Sprintf("%s/wallets/%s/transfers", c.baseURL, from)
	payload := map[string]any{
		"destination_wallet_id":  to,
		"transaction_id":         uuid.NewString(),
		"journal_entry_id":        uuid.NewString(),
		"source_posting_id":      uuid.NewString(),
		"destination_posting_id": uuid.NewString(),
		"amount_minor_units":     100,
		"description":            "Peer-to-peer transfer",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Scenario: "transfer"}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Scenario: "transfer"}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return Result{
		Duration:   time.Since(start),
		StatusCode: resp.StatusCode,
		Scenario:   "transfer",
	}
}

func (c *TestClient) ExecuteSagaCheckout(ctx context.Context, tCtx *TestContext) Result {
	start := time.Now()
	walletID := tCtx.AliceWalletID
	if rand.Intn(2) == 0 {
		walletID = tCtx.BobWalletID
	}

	url := fmt.Sprintf("%s/payments/checkout", c.baseURL)
	payload := map[string]any{
		"hold_id":               uuid.NewString(),
		"wallet_id":             walletID,
		"settlement_account_id": tCtx.SettlementAccountID,
		"transaction_id":        uuid.NewString(),
		"journal_entry_id":      uuid.NewString(),
		"wallet_posting_id":     uuid.NewString(),
		"settlement_posting_id": uuid.NewString(),
		"amount_minor_units":    500,
		"currency":              "BRL",
		"recipient":             "E-Commerce Merchant",
		"description":           "Saga e-commerce checkout",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Scenario: "checkout"}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Scenario: "checkout"}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return Result{
		Duration:   time.Since(start),
		StatusCode: resp.StatusCode,
		Scenario:   "checkout",
	}
}

func (c *TestClient) ExecuteCQRSBalance(ctx context.Context, tCtx *TestContext) Result {
	start := time.Now()
	walletID := tCtx.AliceWalletID
	if rand.Intn(2) == 0 {
		walletID = tCtx.BobWalletID
	}

	url := fmt.Sprintf("%s/wallets/%s/balance", c.baseURL, walletID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Scenario: "cqrs_balance"}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Scenario: "cqrs_balance"}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return Result{
		Duration:   time.Since(start),
		StatusCode: resp.StatusCode,
		Scenario:   "cqrs_balance",
	}
}
