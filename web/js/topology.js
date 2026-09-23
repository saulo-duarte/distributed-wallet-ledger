class ArchitectureTopology {
  constructor() {
    this.nodes = {
      client: document.getElementById("node-client"),
      saga: document.getElementById("node-saga"),
      antifraud: document.getElementById("node-antifraud"),
      circuitBreaker: document.getElementById("node-circuit-breaker"),
      gateway: document.getElementById("node-gateway"),
      postgres: document.getElementById("node-postgres"),
      outbox: document.getElementById("node-outbox"),
      sns: document.getElementById("node-sns"),
      sqs: document.getElementById("node-sqs"),
      dynamodb: document.getElementById("node-dynamo")
    };

    this.pipes = {
      clientToSaga: document.getElementById("pipe-client-saga"),
      sagaToPostgres: document.getElementById("pipe-saga-postgres"),
      sagaToAntifraud: document.getElementById("pipe-saga-antifraud"),
      sagaToCb: document.getElementById("pipe-saga-cb"),
      cbToGateway: document.getElementById("pipe-cb-gateway"),
      postgresToOutbox: document.getElementById("pipe-postgres-outbox"),
      outboxToSns: document.getElementById("pipe-outbox-sns"),
      snsToSqs: document.getElementById("pipe-sns-sqs"),
      sqsToDynamo: document.getElementById("pipe-sqs-dynamo")
    };
  }

  resetAll() {
    Object.entries(this.nodes).forEach(([key, n]) => {
      if (n) {
        if (key === "circuitBreaker") {
          // Preserve CB state class
          return;
        }
        n.className = "topology-node";
      }
    });
    Object.values(this.pipes).forEach(p => {
      if (p) p.setAttribute("class", "flow-pipe");
    });
  }

  setCircuitBreakerState(state) {
    const cbNode = this.nodes.circuitBreaker;
    const badge = document.getElementById("cbStateBadge");
    if (!cbNode || !badge) return;

    cbNode.classList.remove("cb-closed", "cb-open", "cb-half-open");

    if (state === "open") {
      cbNode.classList.add("cb-open");
      badge.textContent = "OPEN · FAIL-FAST";
      badge.style.color = "var(--red)";
    } else if (state === "half_open") {
      cbNode.classList.add("cb-half-open");
      badge.textContent = "HALF-OPEN";
      badge.style.color = "var(--amber)";
    } else {
      cbNode.classList.add("cb-closed");
      badge.textContent = "CLOSED";
      badge.style.color = "var(--green)";
    }
  }

  highlightNode(nodeKey, type = "active") {
    const node = this.nodes[nodeKey];
    if (!node) return;
    if (nodeKey === "circuitBreaker") {
      return;
    }
    node.className = `topology-node ${type === "success" ? "active-success" : type === "error" ? "active-error" : "active"}`;
  }

  unhighlightNode(nodeKey) {
    const node = this.nodes[nodeKey];
    if (node && nodeKey !== "circuitBreaker") {
      node.className = "topology-node";
    }
  }

  flowPipe(pipeKey, state = "flowing") {
    const pipe = this.pipes[pipeKey];
    if (!pipe) return;
    pipe.setAttribute("class", `flow-pipe ${state}`);
  }

  stopPipe(pipeKey) {
    const pipe = this.pipes[pipeKey];
    if (pipe) pipe.setAttribute("class", "flow-pipe");
  }

  async animateSagaSuccess() {
    this.resetAll();

    // 1. Client -> Saga
    this.highlightNode("client");
    this.flowPipe("clientToSaga");
    await this.delay(350);

    // 2. Saga -> Postgres Hold
    this.highlightNode("saga");
    this.flowPipe("sagaToPostgres");
    this.highlightNode("postgres");
    await this.delay(400);

    // 3. Saga -> Anti-Fraud Check
    this.flowPipe("sagaToAntifraud");
    this.highlightNode("antifraud", "success");
    await this.delay(400);
    this.stopPipe("sagaToAntifraud");

    // 4. Saga -> Circuit Breaker -> Payment Gateway
    this.flowPipe("sagaToCb");
    await this.delay(200);
    this.flowPipe("cbToGateway");
    this.highlightNode("gateway", "success");
    await this.delay(450);
    this.stopPipe("sagaToCb");
    this.stopPipe("cbToGateway");

    // 5. Capture in Postgres Ledger
    this.highlightNode("postgres", "success");
    this.flowPipe("postgresToOutbox");
    await this.delay(400);

    // 6. Outbox Relay with Full Jitter -> AWS SNS
    this.highlightNode("outbox");
    this.flowPipe("outboxToSns");
    this.highlightNode("sns");
    await this.delay(350);

    // 7. AWS SNS Fanout -> SQS Projections
    this.flowPipe("snsToSqs");
    this.highlightNode("sqs");
    await this.delay(350);

    // 8. SQS Consumer -> DynamoDB Write
    this.flowPipe("sqsToDynamo");
    this.highlightNode("dynamodb", "success");
    await this.delay(500);

    this.resetAll();
  }

  async animateSagaCircuitBreakerTripped() {
    this.resetAll();

    // 1. Client -> Saga
    this.highlightNode("client");
    this.flowPipe("clientToSaga");
    await this.delay(350);

    // 2. Hold in Postgres
    this.highlightNode("saga");
    this.flowPipe("sagaToPostgres");
    this.highlightNode("postgres");
    await this.delay(350);

    // 3. Anti-Fraud Approved
    this.flowPipe("sagaToAntifraud");
    this.highlightNode("antifraud", "success");
    await this.delay(350);
    this.stopPipe("sagaToAntifraud");

    // 4. Circuit Breaker BLOCKED! (Fail-fast before gateway)
    this.flowPipe("sagaToCb", "compensating");
    this.setCircuitBreakerState("open");
    await this.delay(500);
    this.stopPipe("sagaToCb");

    // 5. Compensating Action: Release Hold
    this.flowPipe("sagaToPostgres", "compensating");
    this.highlightNode("postgres", "error");
    await this.delay(600);

    this.stopPipe("clientToSaga");
    this.stopPipe("sagaToPostgres");
  }

  async animateSagaGatewayDeclined() {
    this.resetAll();

    // 1. Client -> Saga
    this.highlightNode("client");
    this.flowPipe("clientToSaga");
    await this.delay(350);

    // 2. Hold in Postgres
    this.highlightNode("saga");
    this.flowPipe("sagaToPostgres");
    this.highlightNode("postgres");
    await this.delay(350);

    // 3. Anti-Fraud Approved
    this.flowPipe("sagaToAntifraud");
    this.highlightNode("antifraud", "success");
    await this.delay(350);
    this.stopPipe("sagaToAntifraud");

    // 4. Gateway Declined through Circuit Breaker!
    this.flowPipe("sagaToCb");
    this.flowPipe("cbToGateway");
    this.highlightNode("gateway", "error");
    await this.delay(500);
    this.stopPipe("sagaToCb");
    this.stopPipe("cbToGateway");

    // 5. Compensating Action: Release Hold
    this.flowPipe("sagaToPostgres", "compensating");
    this.highlightNode("postgres", "error");
    await this.delay(600);

    this.stopPipe("clientToSaga");
    this.stopPipe("sagaToPostgres");
  }

  async animateSagaAntifraudBlock() {
    this.resetAll();

    // 1. Client -> Saga
    this.highlightNode("client");
    this.flowPipe("clientToSaga");
    await this.delay(350);

    // 2. Hold in Postgres
    this.highlightNode("saga");
    this.flowPipe("sagaToPostgres");
    this.highlightNode("postgres");
    await this.delay(350);

    // 3. Anti-Fraud Block!
    this.flowPipe("sagaToAntifraud");
    this.highlightNode("antifraud", "error");
    await this.delay(500);
    this.stopPipe("sagaToAntifraud");

    // 4. Compensating Action: Release Hold
    this.flowPipe("sagaToPostgres", "compensating");
    this.highlightNode("postgres", "error");
    await this.delay(600);

    this.stopPipe("clientToSaga");
    this.stopPipe("sagaToPostgres");
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

window.ArchitectureTopology = ArchitectureTopology;
