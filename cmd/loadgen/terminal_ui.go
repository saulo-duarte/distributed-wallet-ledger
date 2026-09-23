package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type UI struct {
	totalDuration time.Duration
	scenario      string
	concurrency   int
}

func NewUI(totalDuration time.Duration, scenario string, concurrency int) *UI {
	return &UI{
		totalDuration: totalDuration,
		scenario:      scenario,
		concurrency:   concurrency,
	}
}

func (ui *UI) RenderDashboard(snap Snapshot, podCount int) {
	fmt.Print("\033[H\033[2J")
	fmt.Println("\033[1;36m==============================================================================\033[0m")
	fmt.Println("\033[1;36m       GOLEDGE LOAD TESTING & CONCURRENCY AUTOSCALING DASHBOARD               \033[0m")
	fmt.Println("\033[1;36m==============================================================================\033[0m")

	percent := int((snap.ElapsedTime.Seconds() / ui.totalDuration.Seconds()) * 100)
	if percent > 100 {
		percent = 100
	}
	bar := renderProgressBar(percent, 30)

	fmt.Printf(" \033[1;33mTempo Decorrido:\033[0m %s / %s  [%s] %d%%\n",
		formatDuration(snap.ElapsedTime),
		formatDuration(ui.totalDuration),
		bar,
		percent,
	)
	fmt.Printf(" \033[1;33mCenário:\033[0m %-15s | \033[1;33mWorkers Concorrentes:\033[0m %d\n", ui.scenario, ui.concurrency)
	if podCount > 0 {
		fmt.Printf(" \033[1;32m☸️ Pods K8s Ativos:\033[0m \033[1;32m%d Pods\033[0m\n", podCount)
	} else {
		fmt.Printf(" \033[1;32m☸️ Pods K8s Ativos:\033[0m \033[1;30m(Consultando...)\033[0m\n")
	}
	fmt.Println("\033[0;36m------------------------------------------------------------------------------\033[0m")

	fmt.Println("\033[1;37m📊 THROUGHPUT & TAXA DE REQUISIÇÕES:\033[0m")
	fmt.Printf("   • Total Requisições:   \033[1;36m%d\033[0m\n", snap.TotalRequests)
	fmt.Printf("   • RPS Instantâneo:     \033[1;32m%.1f req/s\033[0m\n", snap.CurrentRPS)
	fmt.Printf("   • RPS Médio:           \033[1;32m%.1f req/s\033[0m\n", snap.AvgRPS)

	fmt.Println("\n\033[1;37m⚡ LATÊNCIA (TEMPO DE RESPOSTA):\033[0m")
	fmt.Printf("   • p50:  \033[1;32m%-10s\033[0m | • p90: \033[1;32m%-10s\033[0m\n", formatDuration(snap.P50), formatDuration(snap.P90))
	fmt.Printf("   • p95:  \033[1;33m%-10s\033[0m | • p99: \033[1;31m%-10s\033[0m\n", formatDuration(snap.P95), formatDuration(snap.P99))
	fmt.Printf("   • Min:  %-10s | • Max: %-10s\n", formatDuration(snap.Min), formatDuration(snap.Max))

	fmt.Println("\n\033[1;37m🎯 CÓDIGOS DE RESPOSTA HTTP:\033[0m")
	success2xx := snap.StatusCodes[200] + snap.StatusCodes[201] + snap.StatusCodes[202]
	handled4xx := snap.StatusCodes[400] + snap.StatusCodes[409] + snap.StatusCodes[422]
	errors5xx := snap.StatusCodes[500] + snap.StatusCodes[502] + snap.StatusCodes[503] + snap.StatusCodes[504]
	var otherErrors int
	for _, count := range snap.Errors {
		otherErrors += count
	}

	fmt.Printf("   • \033[1;32m2xx (Sucesso Total):\033[0m       %d\n", success2xx)
	fmt.Printf("   • \033[1;33m4xx (Concorrência/Validação):\033[0m %d\n", handled4xx)
	fmt.Printf("   • \033[1;31m5xx (Erros do Servidor):\033[0m     %d\n", errors5xx)
	if otherErrors > 0 {
		fmt.Printf("   • \033[1;31mFalhas de Conexão/Rede:\033[0m      %d\n", otherErrors)
	}
	fmt.Println("\033[1;36m==============================================================================\033[0m")
}

func renderProgressBar(percent int, width int) string {
	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}
	var sb strings.Builder
	for i := 0; i < filled; i++ {
		sb.WriteString("█")
	}
	for i := filled; i < width; i++ {
		sb.WriteString("░")
	}
	return sb.String()
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func QueryK8sPodCount() int {
	out, err := exec.Command("kubectl", "get", "pods", "-n", "goledge", "-l", "app.kubernetes.io/name=goledge", "--field-selector=status.phase=Running", "--no-headers").Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	count := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			count++
		}
	}
	return count
}
