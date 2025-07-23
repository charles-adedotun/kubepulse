package slo

import (
	"math"
	"sync"
	"time"
)

// Tracker manages SLO tracking and error budget calculation
type Tracker struct {
	slos    map[string]SLO
	status  map[string]*SLOStatus
	metrics map[string][]Metric
	mu      sync.RWMutex
}

// NewTracker creates a new SLO tracker
func NewTracker() *Tracker {
	return &Tracker{
		slos:    make(map[string]SLO),
		status:  make(map[string]*SLOStatus),
		metrics: make(map[string][]Metric),
	}
}

// AddSLO adds a new SLO to track
func (t *Tracker) AddSLO(slo SLO) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.slos[slo.Name] = slo
	t.status[slo.Name] = &SLOStatus{
		SLO:          slo,
		CurrentValue: 100.0,
		ErrorBudget:  100.0,
		BurnRate:     0.0,
		IsViolated:   false,
	}
}

// UpdateMetrics updates metrics for SLO calculation
func (t *Tracker) UpdateMetrics(sloName string, metrics []Metric) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.metrics[sloName] = append(t.metrics[sloName], metrics...)

	// Keep only recent metrics (within SLO window)
	if len(t.metrics[sloName]) > 1000 {
		t.metrics[sloName] = t.metrics[sloName][len(t.metrics[sloName])-1000:]
	}

	t.calculateSLOStatus(sloName)
}

// GetSLOStatus returns current SLO status
func (t *Tracker) GetSLOStatus(sloName string) (*SLOStatus, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status, exists := t.status[sloName]
	return status, exists
}

// GetAllSLOs returns all SLO statuses
func (t *Tracker) GetAllSLOs() map[string]*SLOStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*SLOStatus)
	for name, status := range t.status {
		result[name] = status
	}
	return result
}

// calculateSLOStatus calculates the current SLO status
func (t *Tracker) calculateSLOStatus(sloName string) {
	slo, exists := t.slos[sloName]
	if !exists {
		return
	}

	status := t.status[sloName]
	metrics := t.metrics[sloName]

	if len(metrics) == 0 {
		return
	}

	// Calculate current value based on SLI type
	switch slo.SLI {
	case "availability":
		status.CurrentValue = t.calculateAvailability(metrics)
	case "latency":
		status.CurrentValue = t.calculateLatency(metrics)
	case "error_rate":
		status.CurrentValue = t.calculateErrorRate(metrics)
	default:
		status.CurrentValue = t.calculateGeneric(metrics)
	}

	// Calculate error budget
	if status.CurrentValue < slo.Target {
		deficit := slo.Target - status.CurrentValue
		status.ErrorBudget = math.Max(0, 100.0-deficit*10) // Simplified calculation
	} else {
		status.ErrorBudget = 100.0
	}

	// Calculate burn rate (errors per time unit)
	status.BurnRate = t.calculateBurnRate(metrics)

	// Check if SLO is violated
	status.IsViolated = status.CurrentValue < slo.Target

	// Calculate time to exhaust error budget
	if status.BurnRate > 0 {
		hoursLeft := status.ErrorBudget / status.BurnRate
		if hoursLeft < 168 { // Less than a week
			status.TimeToExhaust = time.Duration(hoursLeft * float64(time.Hour)).String()
		}
	}
}

// calculateAvailability calculates availability percentage
func (t *Tracker) calculateAvailability(metrics []Metric) float64 {
	if len(metrics) == 0 {
		return 100.0
	}

	var totalRequests, successfulRequests float64

	for _, metric := range metrics {
		if metric.Name == "request_total" {
			totalRequests += metric.Value
		} else if metric.Name == "request_success" {
			successfulRequests += metric.Value
		}
	}

	if totalRequests == 0 {
		return 100.0
	}

	return (successfulRequests / totalRequests) * 100.0
}

// calculateLatency calculates latency percentile
func (t *Tracker) calculateLatency(metrics []Metric) float64 {
	latencies := make([]float64, 0)

	for _, metric := range metrics {
		if metric.Name == "request_duration" {
			latencies = append(latencies, metric.Value)
		}
	}

	if len(latencies) == 0 {
		return 0.0
	}

	// Simple p95 calculation
	return t.percentile(latencies, 95)
}

// calculateErrorRate calculates error rate percentage
func (t *Tracker) calculateErrorRate(metrics []Metric) float64 {
	var totalRequests, errorRequests float64

	for _, metric := range metrics {
		if metric.Name == "request_total" {
			totalRequests += metric.Value
		} else if metric.Name == "request_errors" {
			errorRequests += metric.Value
		}
	}

	if totalRequests == 0 {
		return 0.0
	}

	return 100.0 - (errorRequests/totalRequests)*100.0 // Return success rate
}

// calculateGeneric calculates generic metric average
func (t *Tracker) calculateGeneric(metrics []Metric) float64 {
	if len(metrics) == 0 {
		return 0.0
	}

	var sum float64
	for _, metric := range metrics {
		sum += metric.Value
	}

	return sum / float64(len(metrics))
}

// calculateBurnRate calculates error budget burn rate
func (t *Tracker) calculateBurnRate(metrics []Metric) float64 {
	if len(metrics) < 2 {
		return 0.0
	}

	// Simple burn rate calculation based on recent trend
	recent := metrics[len(metrics)-10:]
	if len(recent) < 2 {
		return 0.0
	}

	// Calculate trend
	var sum float64
	for i := 1; i < len(recent); i++ {
		if recent[i].Value < recent[i-1].Value {
			sum += recent[i-1].Value - recent[i].Value
		}
	}

	return sum / float64(len(recent)-1)
}

// percentile calculates the given percentile of values
func (t *Tracker) percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// Simple percentile calculation (should use proper sorting in production)
	index := int(p/100.0*float64(len(values)-1) + 0.5)
	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}
