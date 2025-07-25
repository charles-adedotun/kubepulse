package core

import (
	"testing"
	"time"
)

func TestHealthStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected string
	}{
		{"healthy status", HealthStatusHealthy, "healthy"},
		{"degraded status", HealthStatusDegraded, "degraded"},
		{"unhealthy status", HealthStatusUnhealthy, "unhealthy"},
		{"unknown status", HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}

func TestCheckResult(t *testing.T) {
	now := time.Now()
	duration := 100 * time.Millisecond

	result := CheckResult{
		Name:       "test-check",
		Status:     HealthStatusHealthy,
		Message:    "All systems operational",
		Timestamp:  now,
		Duration:   duration,
		Confidence: 0.95,
	}

	if result.Name != "test-check" {
		t.Errorf("expected name 'test-check', got %s", result.Name)
	}

	if result.Status != HealthStatusHealthy {
		t.Errorf("expected status %s, got %s", HealthStatusHealthy, result.Status)
	}

	if result.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", result.Confidence)
	}
}

func TestMetric(t *testing.T) {
	metric := Metric{
		Name:      "cpu_usage",
		Value:     75.5,
		Unit:      "percent",
		Timestamp: time.Now(),
		Labels:    map[string]string{"node": "worker-1"},
	}

	if metric.Name != "cpu_usage" {
		t.Errorf("expected name 'cpu_usage', got %s", metric.Name)
	}

	if metric.Value != 75.5 {
		t.Errorf("expected value 75.5, got %f", metric.Value)
	}

	if metric.Unit != "percent" {
		t.Errorf("expected unit 'percent', got %s", metric.Unit)
	}
}
