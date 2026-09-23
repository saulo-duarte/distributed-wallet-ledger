package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	targetURL := flag.String("target", "http://localhost:8082", "Target API base URL")
	duration := flag.Duration("duration", 30*time.Second, "Duration of the load test")
	concurrency := flag.Int("concurrency", 20, "Number of concurrent worker goroutines")
	scenario := flag.String("scenario", "mixed", "Scenario: mixed, checkout, transfers, cqrs")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("\033[1;36m==============================================================================\033[0m")
	fmt.Println("\033[1;36m       INICIANDO MOTOR DE TESTE DE CARGA & CONCORRÊNCIA GOLEDGE               \033[0m")
	fmt.Println("\033[1;36m==============================================================================\033[0m")
	fmt.Printf("• Alvo:        %s\n", *targetURL)
	fmt.Printf("• Duração:     %s\n", *duration)
	fmt.Printf("• Concorrência: %d workers\n", *concurrency)
	fmt.Printf("• Cenário:     %s\n", *scenario)
	fmt.Println("------------------------------------------------------------------------------")
	fmt.Println("Preparando contas e saldos de teste na API...")

	client := NewTestClient(*targetURL)
	setupCtx, setupCancel := context.WithTimeout(ctx, 15*time.Second)
	tCtx, err := client.SetupContext(setupCtx)
	setupCancel()
	if err != nil {
		fmt.Printf("\033[1;31m[ERRO] Falha ao inicializar contexto de teste: %v\033[0m\n", err)
		os.Exit(1)
	}

	fmt.Println("\033[1;32m[OK] Carteiras de teste criadas e provisionadas com sucesso!\033[0m")
	time.Sleep(1 * time.Second)

	metrics := NewMetricsCollector()
	ui := NewUI(*duration, *scenario, *concurrency)

	testCtx, testCancel := context.WithTimeout(ctx, *duration)
	defer testCancel()

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(testCtx, client, tCtx, *scenario, metrics)
		}(i)
	}

	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	podCheckTicker := time.NewTicker(2 * time.Second)
	defer podCheckTicker.Stop()

	currentPodCount := QueryK8sPodCount()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

loop:
	for {
		select {
		case <-done:
			break loop
		case <-podCheckTicker.C:
			currentPodCount = QueryK8sPodCount()
		case <-ticker.C:
			metrics.UpdateRPS()
			ui.RenderDashboard(metrics.Snapshot(), currentPodCount)
		}
	}

	GenerateFinalReport(metrics.Snapshot(), *scenario, *concurrency, *targetURL)
}

func runWorker(
	ctx context.Context,
	client *TestClient,
	tCtx *TestContext,
	scenario string,
	metrics *MetricsCollector,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var res Result
			switch scenario {
			case "checkout":
				res = client.ExecuteSagaCheckout(ctx, tCtx)
			case "transfers":
				res = client.ExecuteTransfer(ctx, tCtx)
			case "cqrs":
				res = client.ExecuteCQRSBalance(ctx, tCtx)
			default:
				r := rand.Intn(100)
				if r < 40 {
					res = client.ExecuteTransfer(ctx, tCtx)
				} else if r < 75 {
					res = client.ExecuteSagaCheckout(ctx, tCtx)
				} else {
					res = client.ExecuteCQRSBalance(ctx, tCtx)
				}
			}
			if ctx.Err() == nil {
				metrics.Record(res)
			}
		}
	}
}
