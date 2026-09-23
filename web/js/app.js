class GoledgeApp {
  constructor() {
    this.api = new ApiClient();
    this.topology = null;
    this.saga = null;
    this.wallets = [];
    this.transactions = [];
    this.sagaLogs = [];
    this.clearingAccountID = localStorage.getItem("studio_clearing_id") || "";
  }

  init() {
    const savedEndpoint = localStorage.getItem("studio_endpoint") || "http://localhost:8082";
    document.getElementById("apiEndpoint").value = savedEndpoint;
    this.api.setBaseUrl(savedEndpoint);

    try {
      this.wallets = JSON.parse(localStorage.getItem("studio_wallets") || "[]");
      this.transactions = JSON.parse(localStorage.getItem("studio_txs") || "[]");
      this.sagaLogs = JSON.parse(localStorage.getItem("studio_saga_logs") || "[]");
    } catch (e) {}

    this.topology = new ArchitectureTopology();
    this.saga = new SagaSimulation(this.api, this.topology, this);

    this.refreshIcons();
    this.testConnection();
    this.startMetricsPolling();
  }

  startMetricsPolling() {
    setInterval(async () => {
      await this.pollMetrics();
    }, 2000);
  }

  async pollMetrics() {
    const metrics = await this.api.getMetrics();
    if (!metrics) return;

    this.topology.setCircuitBreakerState(metrics.circuitBreaker.gateway);

    // Update UI elements
    const cbStateEl = document.getElementById("metricCbState");
    const cbRejEl = document.getElementById("metricCbRejections");
    const cbSummaryEl = document.getElementById("cbRejectionsSummary");

    if (cbStateEl) {
      cbStateEl.textContent = metrics.circuitBreaker.gateway.toUpperCase();
      if (metrics.circuitBreaker.gateway === "open") {
        cbStateEl.style.color = "var(--red)";
      } else if (metrics.circuitBreaker.gateway === "half_open") {
        cbStateEl.style.color = "var(--amber)";
      } else {
        cbStateEl.style.color = "var(--green)";
      }
    }

    if (cbRejEl) {
      cbRejEl.textContent = `${metrics.circuitBreaker.rejections} chamadas bloqueadas (fail-fast)`;
    }
    if (cbSummaryEl) {
      cbSummaryEl.textContent = `${metrics.circuitBreaker.rejections} Rejeições Fail-Fast`;
    }

    const httpTotalEl = document.getElementById("metricHttpTotal");
    const httpBreakdownEl = document.getElementById("metricHttpBreakdown");
    if (httpTotalEl) httpTotalEl.textContent = metrics.requestsTotal;
    if (httpBreakdownEl) {
      httpBreakdownEl.textContent = `2xx: ${metrics.requests2xx} | 4xx: ${metrics.requests4xx} | 5xx: ${metrics.requests5xx}`;
    }

    const txsCountEl = document.getElementById("metricTxsCount");
    if (txsCountEl) txsCountEl.textContent = metrics.transactionsTotal;

    const sagaCountEl = document.getElementById("metricSagaCount");
    if (sagaCountEl) sagaCountEl.textContent = metrics.sagaExecutions;

    const outboxCountEl = document.getElementById("metricOutboxCount");
    if (outboxCountEl) outboxCountEl.textContent = metrics.outboxPublished;

    const sqsCountEl = document.getElementById("metricSqsCount");
    if (sqsCountEl) sqsCountEl.textContent = metrics.sqsConsumed;

    const promSummary = document.getElementById("promSummary");
    if (promSummary) {
      promSummary.textContent = `${metrics.requestsTotal} reqs processadas`;
    }

    // Refresh balances live
    await this.syncBalances();
    this.renderWalletsGrid();
  }

  async testConnection() {
    const input = document.getElementById("apiEndpoint").value.trim().replace(/\/$/, "");
    this.api.setBaseUrl(input);
    localStorage.setItem("studio_endpoint", input);

    const dot = document.getElementById("connDot");
    const text = document.getElementById("connText");

    try {
      const ok = await this.api.checkHealth();
      if (ok) {
        dot.className = "status-dot online";
        text.textContent = "Conectado (8082)";
        this.refreshAll();
      } else {
        throw new Error();
      }
    } catch (e) {
      dot.className = "status-dot offline";
      text.textContent = "Desconectado";
      this.showToast("API em " + input + " não respondeu. Certifique-se que o backend está ativo.", true);
    }
  }

  async refreshAll() {
    await this.syncBalances();
    this.renderWalletsGrid();
    this.renderTransactions();
    this.renderSagaLogs();
    this.populateDropdowns();
    await this.pollMetrics();
  }

  async syncBalances() {
    for (let w of this.wallets) {
      try {
        const b = await this.api.getBalance(w.id);
        w.ledger_balance = b.ledger_balance_minor_units;
        w.available_balance = b.available_balance_minor_units;
      } catch (e) {
        console.warn("Erro ao ler saldo:", w.id, e);
      }
    }
    localStorage.setItem("studio_wallets", JSON.stringify(this.wallets));
  }

  renderWalletsGrid() {
    const container = document.getElementById("walletsGrid");
    document.getElementById("walletCount").textContent = this.wallets.length;

    if (this.wallets.length === 0) {
      container.innerHTML = `
        <div class="empty-card">
          <i data-lucide="wallet-cards"></i>
          <p>Nenhuma carteira ativa encontrada.</p>
          <button class="btn btn-primary" onclick="app.setupDefaultEnvironment()">Criar ambiente de demonstração</button>
        </div>
      `;
      this.refreshIcons();
      return;
    }

    container.innerHTML = "";
    this.wallets.forEach(w => {
      const card = document.createElement("div");
      card.className = "wallet-card";
      card.innerHTML = `
        <div class="wallet-header">
          <div>
            <div class="wallet-owner">
              <span>${w.owner_id}</span>
            </div>
            <div class="wallet-ids">
              Wallet: ${w.id}<br/>
              Ledger Account: ${w.ledger_account_id}
            </div>
          </div>
          <span class="wallet-badge">${w.currency}</span>
        </div>

        <div class="balance-box">
          <div class="balance-col">
            <span class="balance-label">Saldo Disponível</span>
            <span class="balance-val balance-avail">${this.formatBRL(w.available_balance || 0)}</span>
          </div>
          <div class="balance-col" style="text-align: right;">
            <span class="balance-label">Saldo Contábil</span>
            <span class="balance-val" style="font-size: 1.15rem; color: var(--text-muted);">${this.formatBRL(w.ledger_balance || 0)}</span>
          </div>
        </div>

        <div class="wallet-actions">
          <button class="btn btn-primary" onclick="app.openSagaModal('${w.id}')">Checkout</button>
          <button class="btn btn-quiet" onclick="app.openDepositModal('${w.id}')">Depositar</button>
          <button class="btn btn-quiet" onclick="app.openTransferModal('${w.id}')">Transferir</button>
          <button class="btn btn-quiet" onclick="app.openWithdrawModal('${w.id}')">Sacar</button>
          <button class="btn btn-quiet" onclick="app.viewEntries('${w.ledger_account_id}')">Extrato</button>
        </div>
      `;
      container.appendChild(card);
    });
    this.refreshIcons();
  }

  populateDropdowns() {
    const fromSelect = document.getElementById("transferFrom");
    const toSelect = document.getElementById("transferTo");
    const depSelect = document.getElementById("depositTarget");
    const withSelect = document.getElementById("withdrawTarget");
    const sagaSelect = document.getElementById("sagaWalletSelect");
    const accSelect = document.getElementById("selectAccountForEntries");

    const options = this.wallets.map(w => `<option value="${w.id}">${w.owner_id} - Disponível: ${this.formatBRL(w.available_balance || 0)}</option>`).join("");

    if (fromSelect) fromSelect.innerHTML = options;
    if (toSelect) toSelect.innerHTML = options;
    if (depSelect) depSelect.innerHTML = options;
    if (withSelect) withSelect.innerHTML = options;
    if (sagaSelect) sagaSelect.innerHTML = options;

    let accOptions = "";
    if (this.clearingAccountID) {
      accOptions += `<option value="${this.clearingAccountID}">Clearing Geral (${this.clearingAccountID.slice(0, 8)}...)</option>`;
    }
    this.wallets.forEach(w => {
      accOptions += `<option value="${w.ledger_account_id}">${w.owner_id} (${w.ledger_account_id.slice(0, 8)}...)</option>`;
    });
    if (accSelect) accSelect.innerHTML = accOptions || `<option value="">Nenhuma conta registrada</option>`;
  }

  async setupDefaultEnvironment() {
    this.showToast("Criando contas e carteiras no PostgreSQL...");

    try {
      const suffix = Date.now().toString().slice(-4);

      this.clearingAccountID = this.api.generateUUID();
      localStorage.setItem("studio_clearing_id", this.clearingAccountID);
      await this.api.createAccount(this.clearingAccountID, `CLR-${suffix}`, "Conta Garantidora Clearing BRL");

      const aliceAccId = this.api.generateUUID();
      const aliceWalletId = this.api.generateUUID();
      await this.api.createAccount(aliceAccId, `ALICE-${suffix}`, "Passivo - Alice Silva");
      const aliceW = await this.api.createWallet(aliceWalletId, "Alice Silva", aliceAccId, "BRL");

      const bobAccId = this.api.generateUUID();
      const bobWalletId = this.api.generateUUID();
      await this.api.createAccount(bobAccId, `BOB-${suffix}`, "Passivo - Bob Santos");
      const bobW = await this.api.createWallet(bobWalletId, "Bob Santos", bobAccId, "BRL");

      this.wallets = [
        { id: aliceW.id, owner_id: aliceW.owner_id, ledger_account_id: aliceW.ledger_account_id, currency: "BRL" },
        { id: bobW.id, owner_id: bobW.owner_id, ledger_account_id: bobW.ledger_account_id, currency: "BRL" }
      ];

      await this.api.deposit(aliceWalletId, this.clearingAccountID, 100000, "Aporte inicial de liquidez para Alice");

      localStorage.setItem("studio_wallets", JSON.stringify(this.wallets));
      await this.refreshAll();
      this.showToast("Setup concluído! Alice iniciou com R$ 1.000,00!");
    } catch (err) {
      console.error(err);
      this.showToast("Erro no setup: " + err.message, true);
    }
  }

  addSagaLog(log) {
    this.sagaLogs.unshift(log);
    if (this.sagaLogs.length > 30) this.sagaLogs.pop();
    localStorage.setItem("studio_saga_logs", JSON.stringify(this.sagaLogs));
    this.renderSagaLogs();
  }

  addTransaction(transaction) {
    if (!transaction?.id) return;
    this.transactions = this.transactions.filter(item => item.id !== transaction.id);
    this.transactions.unshift(transaction);
    localStorage.setItem("studio_txs", JSON.stringify(this.transactions));
    this.renderTransactions();
  }

  renderSagaLogs() {
    const tbody = document.getElementById("sagaLogsBody");
    if (!tbody) return;

    if (this.sagaLogs.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center; color:var(--text-muted); padding:1.5rem;">Nenhuma execução de Saga registrada nesta sessão.</td></tr>`;
      return;
    }

    tbody.innerHTML = "";
    this.sagaLogs.forEach(l => {
      const isOk = l.status === "COMPLETED";
      const tr = document.createElement("tr");
      const traceLink = l.traceId 
        ? `<a href="http://localhost:16686/trace/${l.traceId}" target="_blank" class="mono">${l.traceId.slice(0, 8)}...</a>`
        : `<span class="mono">—</span>`;

      tr.innerHTML = `
        <td class="mono"><strong>${l.id}</strong></td>
        <td><span class="badge ${isOk ? "badge-credit" : "badge-debit"}">${l.status}</span></td>
        <td>${l.wallet}</td>
        <td class="mono" style="color:var(--cyan-light); font-weight:600;">${this.formatBRL(l.amount)}</td>
        <td>${traceLink}</td>
        <td style="font-size:0.75rem; color:var(--text-dim);">${new Date(l.timestamp).toLocaleTimeString("pt-BR")}</td>
      `;
      tbody.appendChild(tr);
    });
  }

  renderTransactions() {
    const tbody = document.getElementById("txsTableBody");
    if (!tbody) return;
    if (this.transactions.length === 0) {
      tbody.innerHTML = `<tr><td colspan="5" style="text-align:center; color:var(--text-muted); padding:2rem;">Nenhuma transação executada nesta sessão.</td></tr>`;
      return;
    }
    tbody.innerHTML = "";
    this.transactions.forEach(t => {
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td class="mono"><strong>${t.id}</strong></td>
        <td>${t.description}</td>
        <td><span class="badge" style="background:#1e293b; color:var(--cyan-light);">${t.currency || "BRL"}</span></td>
        <td><span class="badge ${t.status === "reversed" ? "badge-debit" : "badge-credit"}">${t.status || "posted"}</span></td>
        <td>
          ${t.status !== "reversed" ? `<button class="btn btn-outline btn-quick" style="color: var(--danger);" onclick="app.reverseTransaction('${t.id}')">Estornar</button>` : `<span style="font-size:0.75rem; color:var(--text-dim);">Estornada</span>`}
        </td>
      `;
      tbody.appendChild(tr);
    });
  }

  async loadAccountEntries(accId) {
    const targetId = accId || document.getElementById("selectAccountForEntries").value;
    if (!targetId) return;

    const tbody = document.getElementById("entriesTableBody");
    tbody.innerHTML = `<tr><td colspan="6" style="text-align:center; padding: 2rem;">Buscando lançamentos no banco de dados...</td></tr>`;

    try {
      const data = await this.api.getAccountEntries(targetId);
      const items = data.items || [];
      if (items.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align:center; padding: 2rem; color: var(--text-muted);">Nenhum lançamento registrado nesta conta contábil.</td></tr>`;
        return;
      }
      tbody.innerHTML = "";
      items.forEach(it => {
        const tr = document.createElement("tr");
        tr.innerHTML = `
          <td class="mono"><strong>${it.posting_id}</strong></td>
          <td class="mono">${it.transaction_id}</td>
          <td>${it.description}</td>
          <td><span class="badge ${it.direction === "debit" ? "badge-debit" : "badge-credit"}">${it.direction.toUpperCase()}</span></td>
          <td class="mono" style="font-weight:600; ${it.direction === "debit" ? "color:#fda4af;" : "color:#6ee7b7;"}">
            ${it.direction === "debit" ? "-" : "+"}${this.formatBRL(it.amount_minor_units)}
          </td>
          <td style="font-size:0.75rem; color:var(--text-dim);">${new Date(it.created_at).toLocaleString("pt-BR")}</td>
        `;
        tbody.appendChild(tr);
      });
    } catch (e) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center; padding: 2rem; color: var(--danger);">Erro ao carregar extrato da conta ${targetId}.</td></tr>`;
    }
  }

  viewEntries(accId) {
    this.switchTab("entries");
    document.getElementById("selectAccountForEntries").value = accId;
    this.loadAccountEntries(accId);
  }

  switchTab(tab) {
    document.querySelectorAll(".tabs-header .tab-btn").forEach(b => b.classList.remove("active"));
    document.querySelectorAll(".panel").forEach(p => p.classList.remove("active"));
    
    const panel = document.getElementById(`panel-${tab}`);
    if (panel) panel.classList.add("active");

    const activeBtn = Array.from(document.querySelectorAll(".tabs-header .tab-btn")).find(b => b.getAttribute("onclick") && b.getAttribute("onclick").includes(`'${tab}'`));
    if (activeBtn) activeBtn.classList.add("active");
  }

  formatBRL(cents) {
    return new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" }).format(cents / 100);
  }

  showToast(msg, isError = false) {
    const t = document.getElementById("toast");
    t.textContent = msg;
    t.style.borderColor = isError ? "var(--danger)" : "var(--cyan-primary)";
    t.classList.add("show");
    setTimeout(() => t.classList.remove("show"), 4500);
  }

  refreshIcons() {
    window.lucide?.createIcons();
  }

  openModal(id) { document.getElementById(id).classList.add("active"); }
  closeModal(id) { document.getElementById(id).classList.remove("active"); }

  openSagaModal(walletId) {
    if (walletId) document.getElementById("sagaWalletSelect").value = walletId;
    this.openModal("modalSagaCheckout");
  }

  openTransferModal(fromId) {
    if (fromId) document.getElementById("transferFrom").value = fromId;
    this.openModal("modalTransfer");
  }

  openDepositModal(targetId) {
    if (targetId) document.getElementById("depositTarget").value = targetId;
    this.openModal("modalDeposit");
  }

  openWithdrawModal(targetId) {
    if (targetId) document.getElementById("withdrawTarget").value = targetId;
    this.openModal("modalWithdraw");
  }

  resetLocalStorage() {
    if (!confirm("Deseja limpar as referências salvas localmente no navegador?")) return;
    localStorage.removeItem("studio_wallets");
    localStorage.removeItem("studio_txs");
    localStorage.removeItem("studio_saga_logs");
    localStorage.removeItem("studio_clearing_id");
    this.wallets = [];
    this.transactions = [];
    this.sagaLogs = [];
    this.clearingAccountID = "";
    this.refreshAll();
    this.showToast("Dados locais limpos!");
  }

  async handleTransferSubmit(e) {
    e.preventDefault();
    const fromId = document.getElementById("transferFrom").value;
    const toId = document.getElementById("transferTo").value;
    const amountReais = parseFloat(document.getElementById("transferAmount").value);
    const desc = document.getElementById("transferDesc").value;

    if (fromId === toId) {
      this.showToast("A carteira de origem e destino devem ser diferentes.", true);
      return;
    }

    const fromWallet = this.wallets.find(w => w.id === fromId);
    const amountCents = Math.round(amountReais * 100);

    if ((fromWallet.available_balance || 0) < amountCents) {
      this.showToast(`Saldo insuficiente! ${fromWallet.owner_id} possui apenas ${this.formatBRL(fromWallet.available_balance || 0)}`, true);
      return;
    }

    try {
      const tx = await this.api.transfer(fromId, toId, amountCents, desc);
      this.transactions.unshift(tx);
      localStorage.setItem("studio_txs", JSON.stringify(this.transactions));
      this.closeModal("modalTransfer");
      await this.refreshAll();
      this.showToast(`Transferência de ${this.formatBRL(amountCents)} realizada com sucesso!`);
    } catch (err) {
      this.showToast("Falha na transferência: " + err.message, true);
    }
  }

  async handleDepositSubmit(e) {
    e.preventDefault();
    const targetId = document.getElementById("depositTarget").value;
    const amountReais = parseFloat(document.getElementById("depositAmount").value);
    const desc = document.getElementById("depositDesc").value;
    const amountCents = Math.round(amountReais * 100);

    try {
      if (!this.clearingAccountID) {
        this.clearingAccountID = this.api.generateUUID();
        localStorage.setItem("studio_clearing_id", this.clearingAccountID);
        await this.api.createAccount(this.clearingAccountID, `CLR-${Date.now().toString().slice(-4)}`, "Conta Clearing");
      }

      const tx = await this.api.deposit(targetId, this.clearingAccountID, amountCents, desc);
      this.transactions.unshift(tx);
      localStorage.setItem("studio_txs", JSON.stringify(this.transactions));
      this.closeModal("modalDeposit");
      await this.refreshAll();
      this.showToast(`Depósito de ${this.formatBRL(amountCents)} creditado com sucesso!`);
    } catch (err) {
      this.showToast("Falha no depósito: " + err.message, true);
    }
  }

  async handleWithdrawSubmit(e) {
    e.preventDefault();
    const walletId = document.getElementById("withdrawTarget").value;
    const amountReais = parseFloat(document.getElementById("withdrawAmount").value);
    const desc = document.getElementById("withdrawDesc").value;
    const amountCents = Math.round(amountReais * 100);

    try {
      if (!this.clearingAccountID) {
        this.clearingAccountID = this.api.generateUUID();
        localStorage.setItem("studio_clearing_id", this.clearingAccountID);
        await this.api.createAccount(this.clearingAccountID, `CLR-${Date.now().toString().slice(-4)}`, "Conta Clearing");
      }

      const tx = await this.api.withdraw(walletId, this.clearingAccountID, amountCents, desc);
      this.transactions.unshift(tx);
      localStorage.setItem("studio_txs", JSON.stringify(this.transactions));
      this.closeModal("modalWithdraw");
      await this.refreshAll();
      this.showToast(`Saque de ${this.formatBRL(amountCents)} processado com sucesso!`);
    } catch (err) {
      this.showToast("Falha no saque: " + err.message, true);
    }
  }

  async handleSagaCheckoutSubmit(e) {
    e.preventDefault();
    const walletId = document.getElementById("sagaWalletSelect").value;
    const scenario = document.getElementById("sagaScenarioSelect").value;
    const amount = parseFloat(document.getElementById("sagaAmountInput").value);
    this.closeModal("modalSagaCheckout");
    await this.saga.runScenario(scenario, walletId, amount);
  }
}

const app = new GoledgeApp();
window.app = app;
window.addEventListener("DOMContentLoaded", () => app.init());
