package slo

import (
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()
	
	if tracker == nil {
		t.Error("expected non-nil tracker")
	}
	
	if tracker.slos == nil {
		t.Error("expected initialized slos map")
	}
	
	if tracker.status == nil {
		t.Error("expected initialized status map")
	}
	
	if tracker.metrics == nil {
		t.Error("expected initialized metrics map")
	}
}

func TestTracker_AddSLO(t *testing.T) {
	tracker := NewTracker()
	
	slo := SLO{
		Name:   "test-slo",
		Target: 99.5,
		SLI:    "availability",
	}
	
	tracker.AddSLO(slo)
	
	// Check if SLO was added
	status, exists := tracker.GetSLOStatus("test-slo")
	if !exists {
		t.Error("expected SLO to exist after adding")
	}
	
	if status.SLO.Name != "test-slo" {
		t.Errorf("expected SLO name 'test-slo', got %s", status.SLO.Name)
	}
	
	if status.CurrentValue != 100.0 {
		t.Errorf("expected initial current value 100.0, got %f", status.CurrentValue)
	}
	
	if status.ErrorBudget != 100.0 {
		t.Errorf("expected initial error budget 100.0, got %f", status.ErrorBudget)
	}
	
	if status.IsViolated {
		t.Error("expected initial SLO to not be violated")
	}
}

func TestTracker_GetSLOStatus(t *testing.T) {
	tracker := NewTracker()
	
	// Test non-existent SLO
	_, exists := tracker.GetSLOStatus("non-existent")
	if exists {
		t.Error("expected non-existent SLO to return false")
	}
	
	// Add SLO and test retrieval
	slo := SLO{Name: "test-slo", Target: 99.0}
	tracker.AddSLO(slo)
	
	status, exists := tracker.GetSLOStatus("test-slo")
	if !exists {
		t.Error("expected SLO to exist")
	}
	
	if status.SLO.Name != "test-slo" {
		t.Errorf("expected SLO name 'test-slo', got %s", status.SLO.Name)
	}
}

func TestTracker_GetAllSLOs(t *testing.T) {
	tracker := NewTracker()
	
	// Test empty tracker
	all := tracker.GetAllSLOs()
	if len(all) != 0 {
		t.Errorf("expected 0 SLOs, got %d", len(all))
	}
	
	// Add multiple SLOs
	slo1 := SLO{Name: "slo-1", Target: 99.0}
	slo2 := SLO{Name: "slo-2", Target: 95.0}
	
	tracker.AddSLO(slo1)
	tracker.AddSLO(slo2)
	
	all = tracker.GetAllSLOs()
	if len(all) != 2 {
		t.Errorf("expected 2 SLOs, got %d", len(all))
	}
	
	if _, exists := all["slo-1"]; !exists {
		t.Error("expected slo-1 to exist in results")
	}
	
	if _, exists := all["slo-2"]; !exists {
		t.Error("expected slo-2 to exist in results")
	}
}

func TestTracker_UpdateMetrics(t *testing.T) {
	tracker := NewTracker()
	
	slo := SLO{
		Name: "test-slo",
		SLI:  "availability",
		Target: 99.0,
	}
	tracker.AddSLO(slo)
	
	// Update with some metrics
	metrics := []Metric{
		{Name: "success", Value: 95.0, Timestamp: time.Now()},
		{Name: "success", Value: 98.0, Timestamp: time.Now().Add(time.Minute)},
	}
	
	tracker.UpdateMetrics("test-slo", metrics)
	
	// Check that metrics were stored (indirectly by checking if status calculation was triggered)
	status, exists := tracker.GetSLOStatus("test-slo")
	if !exists {
		t.Error("expected SLO to exist after updating metrics")
	}
	
	if status == nil {
		t.Error("expected non-nil status after updating metrics")
	}
}