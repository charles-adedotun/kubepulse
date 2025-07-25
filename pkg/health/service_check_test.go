package health

import (
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
)

func TestNewServiceHealthCheck(t *testing.T) {
	check := NewServiceHealthCheck()

	if check == nil {
		t.Fatal("expected non-nil ServiceHealthCheck")
	}

	if check.namespace != "" {
		t.Errorf("expected empty namespace, got %s", check.namespace)
	}

	if check.interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", check.interval)
	}
}

func TestServiceHealthCheck_Name(t *testing.T) {
	check := NewServiceHealthCheck()

	name := check.Name()
	if name != "service-health" {
		t.Errorf("expected name 'service-health', got %s", name)
	}
}

func TestServiceHealthCheck_Description(t *testing.T) {
	check := NewServiceHealthCheck()

	description := check.Description()
	expected := "Monitors service endpoints, selectors, and connectivity"
	if description != expected {
		t.Errorf("expected description '%s', got '%s'", expected, description)
	}
}

func TestServiceHealthCheck_Interval(t *testing.T) {
	check := NewServiceHealthCheck()

	interval := check.Interval()
	if interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", interval)
	}
}

func TestServiceHealthCheck_Criticality(t *testing.T) {
	check := NewServiceHealthCheck()

	criticality := check.Criticality()
	if criticality != core.CriticalityMedium {
		t.Errorf("expected criticality medium, got %v", criticality)
	}
}

func TestServiceHealthCheck_Configure(t *testing.T) {
	check := NewServiceHealthCheck()

	config := map[string]interface{}{
		"namespace": "test-namespace",
	}

	err := check.Configure(config)
	if err != nil {
		t.Errorf("unexpected error configuring: %v", err)
	}

	if check.namespace != "test-namespace" {
		t.Errorf("expected namespace 'test-namespace', got %s", check.namespace)
	}
}

func TestServiceHealthCheck_Configure_InvalidType(t *testing.T) {
	check := NewServiceHealthCheck()
	originalNamespace := check.namespace

	// Test with invalid type - should not panic or change value
	config := map[string]interface{}{
		"namespace": 123, // wrong type
	}

	err := check.Configure(config)
	if err != nil {
		t.Errorf("unexpected error configuring with invalid type: %v", err)
	}

	// Value should remain unchanged
	if check.namespace != originalNamespace {
		t.Errorf("namespace should not change with invalid type")
	}
}
