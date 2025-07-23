package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/health"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestPodHealthCheck_AllHealthy(t *testing.T) {
	// Create fake client with healthy pods
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "healthy-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	)

	check := health.NewPodHealthCheck()
	check.Configure(map[string]interface{}{
		"namespace": "default",
	})

	result, err := check.Check(context.Background(), client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != core.HealthStatusHealthy {
		t.Errorf("Expected status %v, got %v", core.HealthStatusHealthy, result.Status)
	}

	if result.Name != "pod-health" {
		t.Errorf("Expected name 'pod-health', got %s", result.Name)
	}
}

func TestPodHealthCheck_FailedPods(t *testing.T) {
	// Create fake client with failed pods
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failed-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		},
	)

	check := health.NewPodHealthCheck()
	check.Configure(map[string]interface{}{
		"namespace": "default",
	})

	result, err := check.Check(context.Background(), client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != core.HealthStatusUnhealthy {
		t.Errorf("Expected status %v, got %v", core.HealthStatusUnhealthy, result.Status)
	}
}

func TestPodHealthCheck_ProblematicPending(t *testing.T) {
	// Create fake client with problematic pending pod
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "problematic-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ImagePullBackOff",
							},
						},
					},
				},
			},
		},
	)

	check := health.NewPodHealthCheck()
	check.Configure(map[string]interface{}{
		"namespace": "default",
	})

	result, err := check.Check(context.Background(), client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != core.HealthStatusUnhealthy {
		t.Errorf("Expected status %v, got %v", core.HealthStatusUnhealthy, result.Status)
	}
}

func TestPodHealthCheck_HighRestartCount(t *testing.T) {
	// Create fake client with high restart pod
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						RestartCount: 10, // Above default threshold of 5
					},
				},
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	)

	check := health.NewPodHealthCheck()
	check.Configure(map[string]interface{}{
		"namespace": "default",
	})

	result, err := check.Check(context.Background(), client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.Status != core.HealthStatusDegraded {
		t.Errorf("Expected status %v, got %v", core.HealthStatusDegraded, result.Status)
	}
}

func TestPodHealthCheck_Configuration(t *testing.T) {
	check := health.NewPodHealthCheck()

	config := map[string]interface{}{
		"restart_threshold":  3,
		"exclude_namespaces": []string{"kube-system"},
	}

	err := check.Configure(config)
	if err != nil {
		t.Fatalf("Expected no error configuring check, got %v", err)
	}

	if check.Interval() != 30*time.Second {
		t.Errorf("Expected interval 30s, got %v", check.Interval())
	}

	if check.Criticality() != core.CriticalityHigh {
		t.Errorf("Expected high criticality, got %v", check.Criticality())
	}
}

func TestPodHealthCheck_Metrics(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	)

	check := health.NewPodHealthCheck()
	check.Configure(map[string]interface{}{
		"namespace": "default",
	})

	result, err := check.Check(context.Background(), client)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that metrics are generated
	if len(result.Metrics) == 0 {
		t.Error("Expected metrics to be generated")
	}

	// Verify specific metrics exist
	metricNames := make(map[string]bool)
	for _, metric := range result.Metrics {
		metricNames[metric.Name] = true
	}

	expectedMetrics := []string{"pod_total", "pod_running", "pod_failed", "pod_failure_rate"}
	for _, expected := range expectedMetrics {
		if !metricNames[expected] {
			t.Errorf("Expected metric %s not found", expected)
		}
	}
}

func TestPodHealthCheck_APIError(t *testing.T) {
	client := fake.NewSimpleClientset()

	// Add a reactor to simulate API error
	client.PrependReactor("list", "pods", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("API error")
	})

	check := health.NewPodHealthCheck()
	check.Configure(map[string]interface{}{
		"namespace": "default",
	})

	_, err := check.Check(context.Background(), client)

	if err == nil {
		t.Fatal("Expected error due to API failure, got none")
	}
}
