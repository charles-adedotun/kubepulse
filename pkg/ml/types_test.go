package ml

import (
	"testing"
	"time"
)

func TestMetric(t *testing.T) {
	now := time.Now()

	metric := Metric{
		Name:      "cpu_usage",
		Value:     75.5,
		Unit:      "percent",
		Labels:    map[string]string{"node": "worker-1"},
		Timestamp: now,
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

	if metric.Labels["node"] != "worker-1" {
		t.Errorf("expected label node='worker-1', got %s", metric.Labels["node"])
	}

	if !metric.Timestamp.Equal(now) {
		t.Errorf("expected timestamp to match")
	}
}

func TestMetric_EmptyLabels(t *testing.T) {
	metric := Metric{
		Name:      "memory_usage",
		Value:     1024.0,
		Unit:      "MB",
		Timestamp: time.Now(),
	}

	if metric.Labels != nil {
		t.Errorf("expected nil labels, got %v", metric.Labels)
	}
}

func TestPrediction(t *testing.T) {
	now := time.Now()

	prediction := Prediction{
		Timestamp:   now,
		Status:      "degraded",
		Probability: 0.85,
		Reason:      "CPU usage spike detected",
	}

	if !prediction.Timestamp.Equal(now) {
		t.Errorf("expected timestamp to match")
	}

	if prediction.Status != "degraded" {
		t.Errorf("expected status 'degraded', got %s", prediction.Status)
	}

	if prediction.Probability != 0.85 {
		t.Errorf("expected probability 0.85, got %f", prediction.Probability)
	}

	if prediction.Reason != "CPU usage spike detected" {
		t.Errorf("expected reason 'CPU usage spike detected', got %s", prediction.Reason)
	}
}

func TestPrediction_DefaultValues(t *testing.T) {
	prediction := Prediction{}

	if !prediction.Timestamp.IsZero() {
		t.Errorf("expected zero timestamp")
	}

	if prediction.Status != "" {
		t.Errorf("expected empty status, got %s", prediction.Status)
	}

	if prediction.Probability != 0.0 {
		t.Errorf("expected zero probability, got %f", prediction.Probability)
	}

	if prediction.Reason != "" {
		t.Errorf("expected empty reason, got %s", prediction.Reason)
	}
}
