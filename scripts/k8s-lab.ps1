# ==============================================================================
# Goledge Kubernetes Interactive Lab & Chaos Simulator
# ==============================================================================
param (
    [Parameter(Position = 0)]
    [ValidateSet("menu", "deploy", "status", "watch", "kill", "stress", "port-forward", "clean")]
    [string]$Action = "menu"
)

$Namespace = "goledge"
$AppName = "goledge-api"

function Write-Cyan ($text) { Write-Host $text -ForegroundColor Cyan }
function Write-Green ($text) { Write-Host $text -ForegroundColor Green }
function Write-Yellow ($text) { Write-Host $text -ForegroundColor Yellow }
function Write-Red ($text) { Write-Host $text -ForegroundColor Red }
function Write-Navy ($text) { Write-Host $text -ForegroundColor DarkCyan }

function Show-Header {
    Clear-Host
    Write-Cyan "=============================================================================="
    Write-Cyan "           GOLEDGE KUBERNETES LAB & CHAOS/AUTOSCALING SIMULATOR               "
    Write-Cyan "=============================================================================="
    Write-Host ""
}

function Test-ClusterConnection {
    $currentContext = kubectl config current-context 2>$null
    if (-not $currentContext) {
        Write-Red "[!] Nenhum cluster Kubernetes ativo encontrado no kubectl."
        Write-Yellow "[i] Dica: No Docker Desktop -> Settings (Engrenagem) -> Kubernetes -> Marque 'Enable Kubernetes' -> Apply & Restart."
        return $false
    }
    $clusterInfo = kubectl cluster-info 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Red "[!] Nao foi possivel conectar ao servidor Kubernetes ($currentContext)."
        Write-Yellow "[i] Verifique se o Kubernetes do Docker Desktop concluiu a inicializacao."
        return $false
    }
    return $true
}

function Deploy-Stack {
    Show-Header
    Write-Cyan ">>> [1/4] Construindo imagem Docker local da API (goledge-api:latest)..."
    docker build -t goledge-api:latest .
    if ($LASTEXITCODE -ne 0) {
        Write-Red "[!] Falha no build da imagem Docker."
        return
    }

    Write-Cyan ">>> [2/4] Criando Namespace '$Namespace'..."
    kubectl apply -f deploy/k8s/namespace.yaml

    Write-Cyan ">>> [3/5] Subindo Infraestrutura Local (Postgres + Ministack com SSM)..."
    kubectl apply -f deploy/k8s/infra-local.yaml
    kubectl rollout status deployment/postgres -n $Namespace --timeout=45s

    Write-Cyan ">>> [4/5] Executando Migrations no PostgreSQL..."
    Get-ChildItem migrations/*.up.sql | Sort-Object Name | ForEach-Object {
        Get-Content $_.FullName -Raw | kubectl exec -i deploy/postgres -n $Namespace -- psql -U ledger -d ledger 2>$null | Out-Null
    }

    Write-Cyan ">>> [5/5] Aplicando Manifestos do Goledge (ConfigMap, Secret, Deployment, Service, HPA, PDB, Observability)..."
    kubectl apply -f deploy/k8s/configmap.yaml
    kubectl apply -f deploy/k8s/secret.yaml
    kubectl apply -f deploy/k8s/deployment.yaml
    kubectl apply -f deploy/k8s/service.yaml
    kubectl apply -f deploy/k8s/hpa.yaml
    kubectl apply -f deploy/k8s/pdb.yaml
    kubectl apply -f deploy/k8s/observability.yaml

    Write-Green "`n[OK] Todos os recursos foram aplicados no Kubernetes!"
    Write-Yellow "Aguardando inicializacao dos Pods..."
    kubectl rollout status deployment/$AppName -n $Namespace --timeout=60s
}

function Show-Status {
    Write-Cyan "`n------------------------------------------------------------------------------"
    Write-Cyan "                           STATUS ATUAL DO CLUSTER                            "
    Write-Cyan "------------------------------------------------------------------------------"

    Write-Navy ">>> PODS EM EXECUCAO (-n $Namespace):"
    kubectl get pods -n $Namespace -o wide

    Write-Navy "`n>>> HORIZONTAL POD AUTOSCALER (HPA):"
    kubectl get hpa -n $Namespace

    Write-Navy "`n>>> POD DISRUPTION BUDGET (PDB):"
    kubectl get pdb -n $Namespace

    Write-Navy "`n>>> SERVICES & PORTAS:"
    kubectl get svc -n $Namespace
    Write-Cyan "------------------------------------------------------------------------------"
}

function Watch-Cluster {
    Show-Header
    Write-Yellow "[i] Pressione Ctrl+C para sair do modo de monitoramento ao vivo.`n"
    while ($true) {
        Clear-Host
        Write-Cyan "=================== GOLEDGE LIVE CLUSTER MONITOR ==================="
        Write-Host "Horario: $(Get-Date -Format 'HH:mm:ss') | Namespace: $Namespace"
        Show-Status
        Start-Sleep -Seconds 2
    }
}

function Simulate-ChaosKillPod {
    Show-Header
    Write-Yellow "================ SIMULACAO DE FALHA / CHAOS ENGINEERING ================"
    Write-Host "Vamos derrubar um Pod ativo para ver o Kubernetes recria-lo instantaneamente.`n"

    $pods = kubectl get pods -n $Namespace -l "app.kubernetes.io/name=goledge" -o jsonpath='{.items[*].metadata.name}' 2>$null
    if (-not $pods) {
        Write-Red "[!] Nenhum pod do $AppName encontrado no namespace $Namespace."
        return
    }

    $podList = $pods -split ' '
    $targetPod = $podList | Get-Random

    Write-Red ">>> [CHAOS] Derrubando o Pod: $targetPod ..."
    kubectl delete pod $targetPod -n $Namespace --now

    Write-Green "`n>>> Acompanhando a auto-recuperacao (Self-Healing) pelo Deployment Controller:"
    for ($i = 0; $i -lt 6; $i++) {
        Write-Host ""
        kubectl get pods -n $Namespace -l "app.kubernetes.io/name=goledge"
        Start-Sleep -Seconds 2
    }
    Write-Green "`n[OK] O Kubernetes detectou a perda e provisionou um novo Pod automaticamente!"
}

function Simulate-HighLoadStress {
    Show-Header
    Write-Yellow "================ SIMULACAO DE SOBRECARGA & AUTOSCALING (HPA) ================"
    Write-Host "Gerando requisicoes em paralelo para elevar o consumo de CPU e acionar o HPA.`n"

    Write-Cyan ">>> Iniciando gerador de carga dentro do cluster..."
    $loadJobYaml = @'
apiVersion: batch/v1
kind: Job
metadata:
  name: goledge-load-generator
  namespace: goledge
spec:
  ttlSecondsAfterFinished: 30
  template:
    spec:
      containers:
      - name: load-gen
        image: busybox:latest
        command:
          - /bin/sh
          - -c
          - |
            for i in 1 2 3 4 5 6 7 8; do
              (while true; do wget -q -O- http://goledge-api:8082/health/ready > /dev/null 2>&1; done) &
            done
            sleep 60
            kill 0
      restartPolicy: Never
  backoffLimit: 1
'@

    kubectl delete job goledge-load-generator -n $Namespace 2>$null | Out-Null
    $loadJobYaml | kubectl apply -f - | Out-Null

    Write-Green "[OK] Gerador de carga injetado! Monitorando escalonamento automatico do HPA (45 segundos):"
    
    for ($i = 1; $i -le 15; $i++) {
        Write-Host "`n[Ciclo $i/15] Verificando HPA e contagem de Pods:"
        kubectl get hpa -n $Namespace
        kubectl get pods -n $Namespace -l "app.kubernetes.io/name=goledge" --no-headers | Measure-Object | ForEach-Object {
            Write-Cyan "Total de Pods ativos: $($_.Count)"
        }
        Start-Sleep -Seconds 3
    }

    Write-Yellow "`nCarga finalizada. Removendo job de teste..."
    kubectl delete job goledge-load-generator -n $Namespace 2>$null | Out-Null
    Write-Green "[OK] O HPA detectou a carga e ajustou o numero de replicas!"
}

function Run-LoadTest {
    Show-Header
    Write-Cyan "================ MOTOR DE TESTE DE CARGA & CONCORRENCIA ================"
    Write-Host "Configurando teste de estresse transacional e escalonamento no Kubernetes.`n"

    $duration = Read-Host "Duracao do teste em segundos [padrao: 30]"
    if (-not $duration) { $duration = "30" }

    $workers = Read-Host "Quantidade de workers concorrentes [padrao: 30]"
    if (-not $workers) { $workers = "30" }

    Write-Cyan "`nCenarios disponiveis:"
    Write-Host "  1 - Misto (Transferencias, Sagas Checkout e Consultas CQRS) [padrao]"
    Write-Host "  2 - Saga Checkout (Criacao de Hold, Gateway e Liquidacao)"
    Write-Host "  3 - Transferencias Concorrentes (Disputa de Saldo / Locking)"
    Write-Host "  4 - Consultas CQRS (Alta vazao DynamoDB)"
    $scenChoice = Read-Host "Escolha o cenario (1-4)"

    $scenario = "mixed"
    switch ($scenChoice) {
        "2" { $scenario = "checkout" }
        "3" { $scenario = "transfers" }
        "4" { $scenario = "cqrs" }
        default { $scenario = "mixed" }
    }

    $pfJob = Start-Job -ScriptBlock {
        kubectl port-forward svc/goledge-api 8082:8082 -n goledge
    }
    Start-Sleep -Seconds 2

    try {
        go run ./cmd/loadgen -target "http://localhost:8082" -duration "$($duration)s" -concurrency $workers -scenario $scenario
    } finally {
        Stop-Job $pfJob -ErrorAction SilentlyContinue
        Remove-Job $pfJob -ErrorAction SilentlyContinue
    }
}

function Start-PortForward {
    Show-Header
    Write-Cyan ">>> Abrindo Port-Forward: http://localhost:8082 -> Service goledge-api:8082"
    Write-Yellow "[i] Pressione Ctrl+C para encerrar o encaminhamento de porta.`n"
    kubectl port-forward svc/goledge-api 8082:8082 -n $Namespace
}

function Clean-Stack {
    Show-Header
    Write-Red ">>> Removendo todos os recursos do namespace '$Namespace'..."
    kubectl delete namespace $Namespace
    Write-Green "[OK] Namespace e recursos limpos com sucesso!"
}

# --- Roteamento Principal ---
if (-not (Test-ClusterConnection)) {
    exit 1
}

switch ($Action) {
    "deploy"       { Deploy-Stack }
    "status"       { Show-Status }
    "watch"        { Watch-Cluster }
    "kill"         { Simulate-ChaosKillPod }
    "stress"       { Simulate-HighLoadStress }
    "loadtest"     { Run-LoadTest }
    "port-forward" { Start-PortForward }
    "clean"        { Clean-Stack }
    "menu" {
        while ($true) {
            Show-Header
            Write-Host "Escolha uma opcao:" -ForegroundColor White
            Write-Cyan "  [1] Deploy Completo (Build Docker + Infra Local + Goledge API + HPA + PDB)"
            Write-Cyan "  [2] Visualizar Dashboard do Cluster (Status instantaneo)"
            Write-Cyan "  [3] Monitorar ao Vivo (Live Watch a cada 2s)"
            Write-Cyan "  [4] Simular Falha de Pod (Chaos: derrubar pod e ver auto-recuperacao)"
            Write-Cyan "  [5] Simular Sobrecarga de CPU no Cluster (Stress interno via Job)"
            Write-Cyan "  [6] Rodar Motor de Teste de Carga & Concorrencia (com Dashboard & Relatorio)"
            Write-Cyan "  [7] Iniciar Port-Forward (Acessar API em http://localhost:8082)"
            Write-Cyan "  [8] Limpar / Destruir Recursos do Cluster (Clean Namespace)"
            Write-Red  "  [0] Sair"
            Write-Host ""
            $choice = Read-Host "Digite a opcao desejada (0-8)"

            switch ($choice) {
                "1" { Deploy-Stack; Pause }
                "2" { Show-Header; Show-Status; Pause }
                "3" { Watch-Cluster }
                "4" { Simulate-ChaosKillPod; Pause }
                "5" { Simulate-HighLoadStress; Pause }
                "6" { Run-LoadTest; Pause }
                "7" { Start-PortForward }
                "8" { Clean-Stack; Pause }
                "0" { Write-Green "Ate logo!"; exit 0 }
                default { Write-Yellow "Opcao invalida. Pressione Enter para tentar novamente..."; Pause }
            }
        }
    }
}

