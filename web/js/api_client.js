class ApiClient {
  constructor(baseUrl = "http://localhost:8082") {
    this.baseUrl = baseUrl.replace(/\/$/, "");
  }

  setBaseUrl(url) {
    this.baseUrl = url.replace(/\/$/, "");
  }

  generateUUID() {
    if (typeof crypto !== "undefined" && crypto.randomUUID) {
      return crypto.randomUUID();
    }
    return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (c) {
      const r = (Math.random() * 16) | 0,
        v = c === "x" ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  }

  async checkHealth() {
    const res = await fetch(`${this.baseUrl}/health/live`);
    return res.ok;
  }

  async createAccount(id, code, name, currency = "BRL") {
    const res = await fetch(`${this.baseUrl}/accounts`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ id, code, name, currency })
    });
    if (!res.ok && res.status !== 409) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.message || `Erro ao criar conta ${code}`);
    }
    return res.json().catch(() => ({ id, code, name }));
  }

  async createWallet(walletId, ownerId, ledgerAccountId, currency = "BRL") {
    const res = await fetch(`${this.baseUrl}/wallets`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        id: walletId,
        owner_id: ownerId,
        ledger_account_id: ledgerAccountId,
        currency: currency
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.message || `Erro ao criar carteira para ${ownerId}`);
    }
    return res.json();
  }

  async getBalance(walletId, consistency = "strong") {
    const query = consistency === "strong" ? "?consistency=strong" : "";
    const res = await fetch(`${this.baseUrl}/wallets/${walletId}/balance${query}`);
    if (!res.ok) throw new Error("Erro ao consultar saldo: " + res.status);
    return res.json();
  }

  async deposit(walletId, clearingAccountId, amountCents, description) {
    const res = await fetch(`${this.baseUrl}/wallets/${walletId}/deposits`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": this.generateUUID()
      },
      body: JSON.stringify({
        clearing_account_id: clearingAccountId,
        transaction_id: this.generateUUID(),
        journal_entry_id: this.generateUUID(),
        clearing_posting_id: this.generateUUID(),
        wallet_posting_id: this.generateUUID(),
        amount_minor_units: amountCents,
        description: description
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.message || `Erro ${res.status}`);
    }
    return res.json();
  }

  async withdraw(walletId, clearingAccountId, amountCents, description) {
    const res = await fetch(`${this.baseUrl}/wallets/${walletId}/withdrawals`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": this.generateUUID()
      },
      body: JSON.stringify({
        clearing_account_id: clearingAccountId,
        transaction_id: this.generateUUID(),
        journal_entry_id: this.generateUUID(),
        wallet_posting_id: this.generateUUID(),
        clearing_posting_id: this.generateUUID(),
        amount_minor_units: amountCents,
        description: description
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.message || `Erro ${res.status}`);
    }
    return res.json();
  }

  async transfer(fromWalletId, toWalletId, amountCents, description) {
    const res = await fetch(`${this.baseUrl}/wallets/${fromWalletId}/transfers`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": this.generateUUID()
      },
      body: JSON.stringify({
        destination_wallet_id: toWalletId,
        transaction_id: this.generateUUID(),
        journal_entry_id: this.generateUUID(),
        source_posting_id: this.generateUUID(),
        destination_posting_id: this.generateUUID(),
        amount_minor_units: amountCents,
        description: description
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.message || `Erro ${res.status}`);
    }
    return res.json();
  }

  async checkoutSaga(walletId, settlementAccountId, amountCents, recipient, description, idempotencyKey) {
    const key = idempotencyKey || this.generateUUID();
    const res = await fetch(`${this.baseUrl}/payments/checkout`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": key
      },
      body: JSON.stringify({
        wallet_id: walletId,
        settlement_account_id: settlementAccountId,
        amount_minor_units: amountCents,
        currency: "BRL",
        recipient: recipient,
        description: description,
        hold_expires_in_sec: 3600
      })
    });

    const traceId = res.headers.get("X-Trace-ID");
    const data = await res.json().catch(() => ({}));
    if (traceId && data) {
      data.trace_id = traceId;
    }
    if (!res.ok) {
      const error = new Error(data.message || `Erro na Saga: ${res.status}`);
      error.status = res.status;
      error.code = data.code;
      error.trace_id = traceId;
      throw error;
    }
    return data;
  }

  async getAccountEntries(accountId, limit = 50) {
    const res = await fetch(`${this.baseUrl}/accounts/${accountId}/entries?limit=${limit}`);
    if (!res.ok) throw new Error("Erro ao obter extrato");
    return res.json();
  }

  async reverseTransaction(txId) {
    const res = await fetch(`${this.baseUrl}/transactions/${txId}/reversal`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": this.generateUUID()
      },
      body: JSON.stringify({
        id: this.generateUUID(),
        journal_entry_id: this.generateUUID(),
        posting_ids: [this.generateUUID(), this.generateUUID()],
        description: "Estorno contábil compensatório"
      })
    });
    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.message || `Erro ao estornar: ${res.status}`);
    }
    return res.json();
  }

  async getMetrics() {
    try {
      const res = await fetch(`${this.baseUrl}/metrics`);
      if (!res.ok) return null;
      const text = await res.text();
      return this.parsePrometheusMetrics(text);
    } catch (e) {
      return null;
    }
  }

  parsePrometheusMetrics(text) {
    const result = {
      circuitBreaker: {
        gateway: "closed",
        rejections: 0
      },
      requestsTotal: 0,
      requests2xx: 0,
      requests4xx: 0,
      requests5xx: 0,
      transactionsTotal: 0,
      activeHolds: 0,
      outboxPublished: 0,
      sqsConsumed: 0,
      sagaExecutions: 0
    };

    const lines = text.split("\n");
    for (let line of lines) {
      line = line.trim();
      if (!line || line.startsWith("#")) continue;

      if (line.startsWith("ledger_circuit_breaker_state{")) {
        if (line.includes('state="open"') && line.endsWith(" 1")) {
          result.circuitBreaker.gateway = "open";
        } else if (line.includes('state="half_open"') && line.endsWith(" 1")) {
          result.circuitBreaker.gateway = "half_open";
        } else if (line.includes('state="closed"') && line.endsWith(" 1")) {
          result.circuitBreaker.gateway = "closed";
        }
      }

      if (line.startsWith("ledger_circuit_breaker_rejections_total")) {
        const parts = line.split(" ");
        result.circuitBreaker.rejections = parseInt(parts[parts.length - 1], 10) || 0;
      }

      if (line.startsWith("ledger_http_requests_total{")) {
        const parts = line.split(" ");
        const count = parseInt(parts[parts.length - 1], 10) || 0;
        result.requestsTotal += count;
        if (line.includes('status_code="2')) result.requests2xx += count;
        if (line.includes('status_code="4')) result.requests4xx += count;
        if (line.includes('status_code="5')) result.requests5xx += count;
      }

      if (line.startsWith("ledger_transactions_total{")) {
        const parts = line.split(" ");
        result.transactionsTotal += parseInt(parts[parts.length - 1], 10) || 0;
      }

      if (line.startsWith("ledger_active_holds_count{")) {
        const parts = line.split(" ");
        result.activeHolds += parseInt(parts[parts.length - 1], 10) || 0;
      }

      if (line.startsWith("ledger_outbox_events_published_total{")) {
        const parts = line.split(" ");
        result.outboxPublished += parseInt(parts[parts.length - 1], 10) || 0;
      }

      if (line.startsWith("ledger_sqs_messages_consumed_total{")) {
        const parts = line.split(" ");
        result.sqsConsumed += parseInt(parts[parts.length - 1], 10) || 0;
      }

      if (line.startsWith("ledger_saga_executions_total{")) {
        const parts = line.split(" ");
        result.sagaExecutions += parseInt(parts[parts.length - 1], 10) || 0;
      }
    }

    return result;
  }
}

window.ApiClient = ApiClient;
