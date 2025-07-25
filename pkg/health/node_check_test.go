package health

import (
	"context"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
)

func TestNewNodeHealthCheck(t *testing.T) {
	check := NewNodeHealthCheck()

	if check == nil {
		t.Fatal("expected non-nil NodeHealthCheck")
	}

	if check.cpuThreshold != 80.0 {
		t.Errorf("expected CPU threshold 80.0, got %f", check.cpuThreshold)
	}

	if check.memoryThreshold != 85.0 {
		t.Errorf("expected memory threshold 85.0, got %f", check.memoryThreshold)
	}

	if check.diskThreshold != 90.0 {
		t.Errorf("expected disk threshold 90.0, got %f", check.diskThreshold)
	}

	if check.interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", check.interval)
	}
}

func TestNodeHealthCheck_Name(t *testing.T) {
	check := NewNodeHealthCheck()

	name := check.Name()
	if name != "node-health" {
		t.Errorf("expected name 'node-health', got %s", name)
	}
}

func TestNodeHealthCheck_Description(t *testing.T) {
	check := NewNodeHealthCheck()

	description := check.Description()
	expected := "Monitors node conditions, resource usage, and readiness"
	if description != expected {
		t.Errorf("expected description '%s', got '%s'", expected, description)
	}
}

func TestNodeHealthCheck_Interval(t *testing.T) {
	check := NewNodeHealthCheck()

	interval := check.Interval()
	if interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", interval)
	}
}

func TestNodeHealthCheck_Criticality(t *testing.T) {
	check := NewNodeHealthCheck()

	criticality := check.Criticality()
	if criticality != core.CriticalityCritical {
		t.Errorf("expected criticality critical, got %v", criticality)
	}
}

func TestNodeHealthCheck_Configure(t *testing.T) {
	check := NewNodeHealthCheck()

	config := map[string]interface{}{
		"cpu_threshold":    75.0,
		"memory_threshold": 80.0,
		"disk_threshold":   85.0,
	}

	err := check.Configure(config)
	if err != nil {
		t.Errorf("unexpected error configuring: %v", err)
	}

	if check.cpuThreshold != 75.0 {
		t.Errorf("expected CPU threshold 75.0, got %f", check.cpuThreshold)
	}

	if check.memoryThreshold != 80.0 {
		t.Errorf("expected memory threshold 80.0, got %f", check.memoryThreshold)
	}

	if check.diskThreshold != 85.0 {
		t.Errorf("expected disk threshold 85.0, got %f", check.diskThreshold)
	}
}

func TestNodeHealthCheck_Configure_InvalidTypes(t *testing.T) {
	check := NewNodeHealthCheck()
	originalCPU := check.cpuThreshold
	originalMemory := check.memoryThreshold
	originalDisk := check.diskThreshold

	// Test with invalid types - should not panic or change values
	config := map[string]interface{}{
		"cpu_threshold":    "invalid",
		"memory_threshold": 123,  // int instead of float64
		"disk_threshold":   true, // bool instead of float64
	}

	err := check.Configure(config)
	if err != nil {
		t.Errorf("unexpected error configuring with invalid types: %v", err)
	}

	// Values should remain unchanged
	if check.cpuThreshold != originalCPU {
		t.Errorf("CPU threshold should not change with invalid type")
	}

	if check.memoryThreshold != originalMemory {
		t.Errorf("memory threshold should not change with invalid type")
	}

	if check.diskThreshold != originalDisk {
		t.Errorf("disk threshold should not change with invalid type")
	}
}

func TestNodeHealthCheck_GetNodeResourceUsage(t *testing.T) {
	check := NewNodeHealthCheck()

	// Test the placeholder implementation
	cpuMillis, memoryBytes := check.getNodeResourceUsage(context.TODO(), nil, "test-node")

	expectedCPU := int64(2000)
	expectedMemory := int64(4 * 1024 * 1024 * 1024) // 4GB

	if cpuMillis != expectedCPU {
		t.Errorf("expected CPU millis %d, got %d", expectedCPU, cpuMillis)
	}

	if memoryBytes != expectedMemory {
		t.Errorf("expected memory bytes %d, got %d", expectedMemory, memoryBytes)
	}
}
