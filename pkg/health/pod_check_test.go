package health

import (
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
)

func TestNewPodHealthCheck(t *testing.T) {
	check := NewPodHealthCheck()

	if check == nil {
		t.Fatal("expected non-nil PodHealthCheck")
	}

	if check.namespace != "" {
		t.Errorf("expected empty namespace, got %s", check.namespace)
	}

	if check.restartThreshold != 5 {
		t.Errorf("expected restart threshold 5, got %d", check.restartThreshold)
	}

	if check.interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", check.interval)
	}

	expectedExcludes := []string{"kube-system", "kube-public"}
	if len(check.excludeNamespaces) != len(expectedExcludes) {
		t.Errorf("expected %d excluded namespaces, got %d", len(expectedExcludes), len(check.excludeNamespaces))
	}

	for i, ns := range expectedExcludes {
		if check.excludeNamespaces[i] != ns {
			t.Errorf("expected excluded namespace %s, got %s", ns, check.excludeNamespaces[i])
		}
	}
}

func TestPodHealthCheck_Name(t *testing.T) {
	check := NewPodHealthCheck()

	name := check.Name()
	if name != "pod-health" {
		t.Errorf("expected name 'pod-health', got %s", name)
	}
}

func TestPodHealthCheck_Description(t *testing.T) {
	check := NewPodHealthCheck()

	description := check.Description()
	expected := "Monitors pod status, restarts, and container readiness"
	if description != expected {
		t.Errorf("expected description '%s', got '%s'", expected, description)
	}
}

func TestPodHealthCheck_Interval(t *testing.T) {
	check := NewPodHealthCheck()

	interval := check.Interval()
	if interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", interval)
	}
}

func TestPodHealthCheck_Criticality(t *testing.T) {
	check := NewPodHealthCheck()

	criticality := check.Criticality()
	if criticality != core.CriticalityHigh {
		t.Errorf("expected criticality high, got %v", criticality)
	}
}

func TestPodHealthCheck_Configure(t *testing.T) {
	check := NewPodHealthCheck()

	config := map[string]interface{}{
		"namespace":               "test-namespace",
		"restart_threshold":       10,
		"exclude_namespaces":      []string{"exclude-me"},
		"include_only_namespaces": []string{"include-me"},
	}

	err := check.Configure(config)
	if err != nil {
		t.Errorf("unexpected error configuring: %v", err)
	}

	if check.namespace != "test-namespace" {
		t.Errorf("expected namespace 'test-namespace', got %s", check.namespace)
	}

	if check.restartThreshold != 10 {
		t.Errorf("expected restart threshold 10, got %d", check.restartThreshold)
	}

	if len(check.excludeNamespaces) != 1 || check.excludeNamespaces[0] != "exclude-me" {
		t.Errorf("expected exclude namespaces ['exclude-me'], got %v", check.excludeNamespaces)
	}

	if len(check.includeOnlyNamespaces) != 1 || check.includeOnlyNamespaces[0] != "include-me" {
		t.Errorf("expected include only namespaces ['include-me'], got %v", check.includeOnlyNamespaces)
	}
}

func TestPodHealthCheck_Configure_InvalidTypes(t *testing.T) {
	check := NewPodHealthCheck()
	originalThreshold := check.restartThreshold
	originalNamespace := check.namespace

	// Test with invalid types - should not panic or change values
	config := map[string]interface{}{
		"namespace":         123,     // wrong type
		"restart_threshold": "wrong", // wrong type
	}

	err := check.Configure(config)
	if err != nil {
		t.Errorf("unexpected error configuring with invalid types: %v", err)
	}

	// Values should remain unchanged
	if check.restartThreshold != originalThreshold {
		t.Errorf("restart threshold should not change with invalid type")
	}

	if check.namespace != originalNamespace {
		t.Errorf("namespace should not change with invalid type")
	}
}
