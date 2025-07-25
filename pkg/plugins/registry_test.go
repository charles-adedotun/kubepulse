package plugins

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	"k8s.io/client-go/kubernetes"
)

// mockHealthCheck implements core.HealthCheck for testing
type mockHealthCheck struct {
	name        string
	description string
	interval    time.Duration
	criticality core.Criticality
}

func (m *mockHealthCheck) Name() string {
	return m.name
}

func (m *mockHealthCheck) Description() string {
	return m.description
}

func (m *mockHealthCheck) Check(ctx context.Context, client kubernetes.Interface) (core.CheckResult, error) {
	return core.CheckResult{
		Name:      m.name,
		Status:    core.HealthStatusHealthy,
		Timestamp: time.Now(),
		Message:   "Mock check passed",
	}, nil
}

func (m *mockHealthCheck) Configure(config map[string]interface{}) error {
	return nil
}

func (m *mockHealthCheck) Interval() time.Duration {
	return m.interval
}

func (m *mockHealthCheck) Criticality() core.Criticality {
	return m.criticality
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.checks == nil {
		t.Fatal("expected initialized checks map")
	}

	if len(registry.checks) != 0 {
		t.Errorf("expected empty registry, got %d checks", len(registry.checks))
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	check := &mockHealthCheck{
		name:        "test-check",
		description: "Test health check",
		interval:    30 * time.Second,
		criticality: core.CriticalityMedium,
	}

	// Test successful registration
	err := registry.Register(check)
	if err != nil {
		t.Errorf("unexpected error registering check: %v", err)
	}

	// Verify check was registered
	if len(registry.checks) != 1 {
		t.Errorf("expected 1 check in registry, got %d", len(registry.checks))
	}

	// Test duplicate registration
	err = registry.Register(check)
	if err == nil {
		t.Error("expected error when registering duplicate check")
	}

	expectedMsg := "health check test-check already registered"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	check := &mockHealthCheck{
		name:        "test-check",
		description: "Test health check",
		interval:    30 * time.Second,
		criticality: core.CriticalityMedium,
	}

	// Test getting non-existent check
	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("expected error when getting non-existent check")
	}

	expectedMsg := "health check non-existent not found"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Register check and test retrieval
	err = registry.Register(check)
	if err != nil {
		t.Fatalf("failed to register check: %v", err)
	}

	retrieved, err := registry.Get("test-check")
	if err != nil {
		t.Errorf("unexpected error getting check: %v", err)
	}

	if retrieved != check {
		t.Error("retrieved check is not the same instance as registered")
	}

	if retrieved.Name() != "test-check" {
		t.Errorf("expected check name 'test-check', got %s", retrieved.Name())
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	check := &mockHealthCheck{
		name:        "test-check",
		description: "Test health check",
		interval:    30 * time.Second,
		criticality: core.CriticalityMedium,
	}

	// Test unregistering non-existent check
	err := registry.Unregister("non-existent")
	if err == nil {
		t.Error("expected error when unregistering non-existent check")
	}

	expectedMsg := "health check non-existent not found"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Register check and test unregistration
	err = registry.Register(check)
	if err != nil {
		t.Fatalf("failed to register check: %v", err)
	}

	if len(registry.checks) != 1 {
		t.Errorf("expected 1 check in registry, got %d", len(registry.checks))
	}

	err = registry.Unregister("test-check")
	if err != nil {
		t.Errorf("unexpected error unregistering check: %v", err)
	}

	if len(registry.checks) != 0 {
		t.Errorf("expected 0 checks in registry after unregister, got %d", len(registry.checks))
	}

	// Verify check is no longer retrievable
	_, err = registry.Get("test-check")
	if err == nil {
		t.Error("expected error when getting unregistered check")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Test empty registry
	checks := registry.List()
	if len(checks) != 0 {
		t.Errorf("expected 0 checks from empty registry, got %d", len(checks))
	}

	// Add multiple checks
	check1 := &mockHealthCheck{name: "check-1", criticality: core.CriticalityHigh}
	check2 := &mockHealthCheck{name: "check-2", criticality: core.CriticalityMedium}
	check3 := &mockHealthCheck{name: "check-3", criticality: core.CriticalityLow}

	_ = registry.Register(check1)
	_ = registry.Register(check2)
	_ = registry.Register(check3)

	checks = registry.List()
	if len(checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(checks))
	}

	// Verify all checks are present (order may vary due to map iteration)
	checkNames := make(map[string]bool)
	for _, check := range checks {
		checkNames[check.Name()] = true
	}

	if !checkNames["check-1"] {
		t.Error("check-1 not found in list")
	}
	if !checkNames["check-2"] {
		t.Error("check-2 not found in list")
	}
	if !checkNames["check-3"] {
		t.Error("check-3 not found in list")
	}
}

func TestRegistry_ListByName(t *testing.T) {
	registry := NewRegistry()

	// Test empty registry
	names := registry.ListByName()
	if len(names) != 0 {
		t.Errorf("expected 0 names from empty registry, got %d", len(names))
	}

	// Add multiple checks
	check1 := &mockHealthCheck{name: "check-1"}
	check2 := &mockHealthCheck{name: "check-2"}
	check3 := &mockHealthCheck{name: "check-3"}

	_ = registry.Register(check1)
	_ = registry.Register(check2)
	_ = registry.Register(check3)

	names = registry.ListByName()
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// Verify all names are present (order may vary due to map iteration)
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	if !nameSet["check-1"] {
		t.Error("check-1 not found in name list")
	}
	if !nameSet["check-2"] {
		t.Error("check-2 not found in name list")
	}
	if !nameSet["check-3"] {
		t.Error("check-3 not found in name list")
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	// Test empty registry
	count := registry.Count()
	if count != 0 {
		t.Errorf("expected count 0 for empty registry, got %d", count)
	}

	// Add checks and verify count
	check1 := &mockHealthCheck{name: "check-1"}
	check2 := &mockHealthCheck{name: "check-2"}

	_ = registry.Register(check1)
	count = registry.Count()
	if count != 1 {
		t.Errorf("expected count 1 after adding one check, got %d", count)
	}

	_ = registry.Register(check2)
	count = registry.Count()
	if count != 2 {
		t.Errorf("expected count 2 after adding two checks, got %d", count)
	}

	// Remove check and verify count
	_ = registry.Unregister("check-1")
	count = registry.Count()
	if count != 1 {
		t.Errorf("expected count 1 after removing one check, got %d", count)
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry()

	// Add multiple checks
	check1 := &mockHealthCheck{name: "check-1"}
	check2 := &mockHealthCheck{name: "check-2"}
	check3 := &mockHealthCheck{name: "check-3"}

	_ = registry.Register(check1)
	_ = registry.Register(check2)
	_ = registry.Register(check3)

	// Verify checks are present
	if registry.Count() != 3 {
		t.Errorf("expected 3 checks before clear, got %d", registry.Count())
	}

	// Clear registry
	registry.Clear()

	// Verify registry is empty
	if registry.Count() != 0 {
		t.Errorf("expected 0 checks after clear, got %d", registry.Count())
	}

	if len(registry.checks) != 0 {
		t.Errorf("expected empty checks map after clear, got %d entries", len(registry.checks))
	}

	// Verify checks are no longer retrievable
	_, err := registry.Get("check-1")
	if err == nil {
		t.Error("expected error when getting check after clear")
	}

	names := registry.ListByName()
	if len(names) != 0 {
		t.Errorf("expected 0 names after clear, got %d", len(names))
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent operations to ensure thread safety
	done := make(chan bool)
	numGoroutines := 10

	// Concurrent registrations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			check := &mockHealthCheck{
				name: fmt.Sprintf("check-%d", id),
			}
			_ = registry.Register(check)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all checks were registered
	if registry.Count() != numGoroutines {
		t.Errorf("expected %d checks after concurrent registration, got %d", numGoroutines, registry.Count())
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			_, err := registry.Get(fmt.Sprintf("check-%d", id))
			if err != nil {
				t.Errorf("unexpected error getting check-%d: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
