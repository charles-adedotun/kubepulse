package slo

import (
	"testing"
	"time"
)

func TestSLO(t *testing.T) {
	slo := SLO{
		Name:        "api-availability",
		Description: "API endpoint availability",
		SLI:         "success_rate",
		Target:      99.9,
		Window:      24 * time.Hour,
		BudgetPolicy: []BudgetRule{
			{Threshold: 0.5, Action: "notify"},
			{Threshold: 0.8, Action: "alert"},
		},
	}

	if slo.Name != "api-availability" {
		t.Errorf("expected name 'api-availability', got %s", slo.Name)
	}

	if slo.Target != 99.9 {
		t.Errorf("expected target 99.9, got %f", slo.Target)
	}

	if slo.Window != 24*time.Hour {
		t.Errorf("expected window 24h, got %v", slo.Window)
	}

	if len(slo.BudgetPolicy) != 2 {
		t.Errorf("expected 2 budget rules, got %d", len(slo.BudgetPolicy))
	}
}

func TestBudgetRule(t *testing.T) {
	rule := BudgetRule{
		Threshold: 0.75,
		Action:    "page",
	}

	if rule.Threshold != 0.75 {
		t.Errorf("expected threshold 0.75, got %f", rule.Threshold)
	}

	if rule.Action != "page" {
		t.Errorf("expected action 'page', got %s", rule.Action)
	}
}

func TestSLOStatus(t *testing.T) {
	slo := SLO{
		Name:   "test-slo",
		Target: 99.0,
	}

	status := SLOStatus{
		SLO:           slo,
		CurrentValue:  98.5,
		ErrorBudget:   0.15,
		BurnRate:      2.0,
		IsViolated:    true,
		TimeToExhaust: "2h30m",
	}

	if status.SLO.Name != "test-slo" {
		t.Errorf("expected SLO name 'test-slo', got %s", status.SLO.Name)
	}

	if status.CurrentValue != 98.5 {
		t.Errorf("expected current value 98.5, got %f", status.CurrentValue)
	}

	if status.ErrorBudget != 0.15 {
		t.Errorf("expected error budget 0.15, got %f", status.ErrorBudget)
	}

	if !status.IsViolated {
		t.Error("expected SLO to be violated")
	}

	if status.TimeToExhaust != "2h30m" {
		t.Errorf("expected time to exhaust '2h30m', got %s", status.TimeToExhaust)
	}
}

func TestMetric(t *testing.T) {
	now := time.Now()
	
	metric := Metric{
		Name:      "response_time",
		Value:     125.5,
		Timestamp: now,
	}

	if metric.Name != "response_time" {
		t.Errorf("expected name 'response_time', got %s", metric.Name)
	}

	if metric.Value != 125.5 {
		t.Errorf("expected value 125.5, got %f", metric.Value)
	}

	if !metric.Timestamp.Equal(now) {
		t.Errorf("expected timestamp to match")
	}
}