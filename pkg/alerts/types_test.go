package alerts

import (
	"testing"
	"time"
)

func TestAlertSeverityConstants(t *testing.T) {
	tests := []struct {
		name     string
		severity AlertSeverity
		expected string
	}{
		{"critical severity", AlertSeverityCritical, "critical"},
		{"warning severity", AlertSeverityWarning, "warning"},
		{"info severity", AlertSeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.severity) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.severity))
			}
		})
	}
}

func TestAlert(t *testing.T) {
	now := time.Now()

	alert := Alert{
		ID:        "alert-123",
		Name:      "high-cpu-usage",
		Severity:  AlertSeverityCritical,
		Message:   "CPU usage exceeded 90%",
		Source:    "kubepulse-monitor",
		Timestamp: now,
		Labels:    map[string]string{"node": "worker-1", "namespace": "default"},
	}

	if alert.ID != "alert-123" {
		t.Errorf("expected ID 'alert-123', got %s", alert.ID)
	}

	if alert.Name != "high-cpu-usage" {
		t.Errorf("expected name 'high-cpu-usage', got %s", alert.Name)
	}

	if alert.Severity != AlertSeverityCritical {
		t.Errorf("expected severity %s, got %s", AlertSeverityCritical, alert.Severity)
	}

	if alert.Source != "kubepulse-monitor" {
		t.Errorf("expected source 'kubepulse-monitor', got %s", alert.Source)
	}

	if len(alert.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(alert.Labels))
	}
}
