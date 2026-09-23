class SagaSimulation {
  constructor(apiClient, topology, appState) {
    this.api = apiClient;
    this.topology = topology;
    this.app = appState;
  }

  async runScenario(scenarioType, walletId, amountReais) {
    const amountCents = Math.round(amountReais * 100);
    const wallet = this.app.wallets.find(w => w.id === walletId);
    if (!wallet || !this.app.clearingAccountID) {
      this.app.showToast(wallet ? "Prepare o ambiente antes de executar a saga." : "Carteira não encontrada.", true);
      return;
    }

    if (scenarioType === "trip_circuit_breaker") {
      await this.tripCircuitBreaker(wallet, amountCents);
      return;
    }

    const scenario = this.scenarioDetails(scenarioType);
    this.app.showToast(`Executando: ${scenario.label}`);
    try {
      const result = await this.api.checkoutSaga(wallet.id, this.app.clearingAccountID, amountCents, scenario.recipient, scenario.description);
      await this.topology.animateSagaSuccess();
      this.recordSuccess(result, scenarioType, wallet, amountCents, scenario.description);
      await this.app.refreshAll();
      this.app.showToast(`Pagamento capturado. Débito de ${this.app.formatBRL(amountCents)} registrado no ledger.`);
    } catch (err) {
      await this.animateFailure(scenarioType, err);
      this.recordFailure(err, scenarioType, wallet, amountCents);
      await this.app.refreshAll();
      this.app.showToast(`Fluxo compensado: ${err.message}`, true);
    }
  }

  scenarioDetails(type) {
    const scenarios = {
      success: { label: "pagamento aprovado", recipient: "merchant_clean", description: "Checkout Saga - pagamento aprovado" },
      antifraud_block: { label: "bloqueio antifraude", recipient: "fraud_blocked_user@test.com", description: "Checkout Saga - bloqueio antifraude" },
      gateway_declined: { label: "cartão recusado", recipient: "declined_card_issuer", description: "Checkout Saga - cartão recusado" },
      timeout: { label: "timeout técnico", recipient: "timeout_endpoint", description: "Checkout Saga - timeout técnico" }
    };
    return scenarios[type] || scenarios.success;
  }

  async tripCircuitBreaker(wallet, amountCents) {
    const metrics = await this.api.getMetrics();
    if (metrics?.circuitBreaker?.gateway === "open") {
      this.topology.setCircuitBreakerState("open");
      this.app.showToast("O circuit breaker já está aberto. Aguarde 5 segundos e execute compras aprovadas para recuperá-lo.", true);
      return;
    }

    this.app.showToast("Gerando 3 falhas técnicas para abrir o circuit breaker...");
    const description = "Checkout Saga - falha técnica para circuit breaker";
    let finalError;

    for (let attempt = 1; attempt <= 3; attempt++) {
      try {
        await this.api.checkoutSaga(wallet.id, this.app.clearingAccountID, amountCents, "timeout_endpoint", description);
      } catch (err) {
        finalError = err;
        await this.topology.animateSagaGatewayDeclined();
      }
    }

    try {
      await this.api.checkoutSaga(wallet.id, this.app.clearingAccountID, amountCents, "timeout_endpoint", description);
    } catch (err) {
      finalError = err;
      await this.topology.animateSagaCircuitBreakerTripped();
    }

    this.app.addSagaLog({
      id: `breaker_${Date.now().toString().slice(-6)}`,
      scenario: "trip_circuit_breaker",
      status: "CIRCUIT BREAKER OPEN · FAIL-FAST",
      wallet: wallet.owner_id,
      amount: amountCents,
      error: finalError?.message || "Falhas técnicas simuladas",
      timestamp: new Date().toISOString()
    });
    await this.app.refreshAll();
    this.app.showToast("Circuit breaker aberto. A quarta chamada foi bloqueada sem acessar o gateway.", true);
  }

  async animateFailure(type, err) {
    if (err.message?.includes("circuit breaker is open")) {
      await this.topology.animateSagaCircuitBreakerTripped();
    } else if (type === "antifraud_block") {
      await this.topology.animateSagaAntifraudBlock();
    } else {
      await this.topology.animateSagaGatewayDeclined();
    }
  }

  recordSuccess(result, type, wallet, amount, description) {
    this.app.addSagaLog({ id: result.transaction_id || result.hold_id, scenario: type, status: "COMPLETED", wallet: wallet.owner_id, amount, gatewayTx: result.gateway_transaction_id, traceId: result.trace_id || "", timestamp: new Date().toISOString() });
    this.app.addTransaction({ id: result.transaction_id, description, currency: "BRL", status: "posted" });
  }

  recordFailure(err, type, wallet, amount) {
    let status = "COMPENSATED · HOLD RELEASED";
    if (err.message?.includes("circuit breaker is open")) status = "CIRCUIT BREAKER OPEN · FAIL-FAST";
    if (err.code === "anti_fraud_rejected") status = "ANTIFRAUD REJECTED · COMPENSATED";
    if (err.code === "external_payment_failed") status = "GATEWAY FAILED · COMPENSATED";
    this.app.addSagaLog({ id: `hold_${Date.now().toString().slice(-6)}`, scenario: type, status, wallet: wallet.owner_id, amount, error: err.message, traceId: err.trace_id || "", timestamp: new Date().toISOString() });
  }
}

window.SagaSimulation = SagaSimulation;
