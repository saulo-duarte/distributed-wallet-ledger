package main

import (
	"sort"
	"sync"
	"time"
)

type Result struct {
	Duration   time.Duration
	StatusCode int
	Err        error
	Scenario   string
}

type MetricsCollector struct {
	mu           sync.Mutex
	latencies    []time.Duration
	statusCodes  map[int]int
	errors       map[string]int
	scenarioHits map[string]int
	startTime    time.Time
	recentCount  int
	lastRPSCheck time.Time
	currentRPS   float64
}

func NewMetricsCollector() *MetricsCollector {
	now := time.Now()
	return &MetricsCollector{
		statusCodes:  make(map[int]int),
		errors:       make(map[string]int),
		scenarioHits: make(map[string]int),
		startTime:    now,
		lastRPSCheck: now,
	}
}

func (m *MetricsCollector) Record(res Result) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencies = append(m.latencies, res.Duration)
	m.scenarioHits[res.Scenario]++
	m.recentCount++

	if res.Err != nil {
		m.errors[res.Err.Error()]++
		return
	}

	m.statusCodes[res.StatusCode]++
}

func (m *MetricsCollector) UpdateRPS() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastRPSCheck).Seconds()
	if elapsed >= 0.5 {
		m.currentRPS = float64(m.recentCount) / elapsed
		m.recentCount = 0
		m.lastRPSCheck = now
	}
	return m.currentRPS
}

type Snapshot struct {
	TotalRequests int
	CurrentRPS    float64
	AvgRPS        float64
	P50           time.Duration
	P90           time.Duration
	P95           time.Duration
	P99           time.Duration
	Min           time.Duration
	Max           time.Duration
	StatusCodes   map[int]int
	Errors        map[string]int
	Scenarios     map[string]int
	ElapsedTime   time.Duration
}

func (m *MetricsCollector) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := len(m.latencies)
	elapsed := time.Since(m.startTime)
	var avgRPS float64
	if elapsed.Seconds() > 0 {
		avgRPS = float64(total) / elapsed.Seconds()
	}

	statusCopy := make(map[int]int, len(m.statusCodes))
	for k, v := range m.statusCodes {
		statusCopy[k] = v
	}

	errCopy := make(map[string]int, len(m.errors))
	for k, v := range m.errors {
		errCopy[k] = v
	}

	scenCopy := make(map[string]int, len(m.scenarioHits))
	for k, v := range m.scenarioHits {
		scenCopy[k] = v
	}

	if total == 0 {
		return Snapshot{
			TotalRequests: 0,
			CurrentRPS:    m.currentRPS,
			AvgRPS:        avgRPS,
			StatusCodes:   statusCopy,
			Errors:        errCopy,
			Scenarios:     scenCopy,
			ElapsedTime:   elapsed,
		}
	}

	sorted := make([]time.Duration, total)
	copy(sorted, m.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	return Snapshot{
		TotalRequests: total,
		CurrentRPS:    m.currentRPS,
		AvgRPS:        avgRPS,
		P50:           percentile(sorted, 50),
		P90:           percentile(sorted, 90),
		P95:           percentile(sorted, 95),
		P99:           percentile(sorted, 99),
		Min:           sorted[0],
		Max:           sorted[total-1],
		StatusCodes:   statusCopy,
		Errors:        errCopy,
		Scenarios:     scenCopy,
		ElapsedTime:   elapsed,
	}
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := (p * len(sorted)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
