package health

import (
	"context"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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

	// Nil client should safely return 0, 0
	if cpu, mem := check.getNodeResourceUsage(context.Background(), nil, "test-node"); cpu != 0 || mem != 0 {
		t.Fatalf("expected 0, 0 for nil client, got %d, %d", cpu, mem)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			NodeName: "test-node",
			Containers: []corev1.Container{
				{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	client := fake.NewSimpleClientset(pod)
	cpuMillis, memoryBytes := check.getNodeResourceUsage(context.Background(), client, "test-node")

	if cpuMillis != 500 {
		t.Errorf("expected CPU millis 500, got %d", cpuMillis)
	}
	wantMem := int64(256 * 1024 * 1024)
	if memoryBytes != wantMem {
		t.Errorf("expected memory bytes %d, got %d", wantMem, memoryBytes)
	}
}
