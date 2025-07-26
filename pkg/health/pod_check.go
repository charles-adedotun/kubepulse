package health

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodHealthCheck checks the health of pods in the cluster
type PodHealthCheck struct {
	namespace             string
	restartThreshold      int32
	interval              time.Duration
	excludeNamespaces     []string
	includeOnlyNamespaces []string
}

// NewPodHealthCheck creates a new pod health check
func NewPodHealthCheck() *PodHealthCheck {
	return &PodHealthCheck{
		namespace:         "",
		restartThreshold:  5,
		interval:          30 * time.Second,
		excludeNamespaces: []string{"kube-system", "kube-public"},
	}
}

// Name returns the name of the health check
func (p *PodHealthCheck) Name() string {
	return "pod-health"
}

// Description returns a description of the health check
func (p *PodHealthCheck) Description() string {
	return "Monitors pod status, restarts, and container readiness"
}

// Check performs the pod health check
func (p *PodHealthCheck) Check(ctx context.Context, client kubernetes.Interface) (core.CheckResult, error) {
	result := core.CheckResult{
		Name:      p.Name(),
		Timestamp: time.Now(),
		Status:    core.HealthStatusHealthy,
		Details:   make(map[string]interface{}),
		Metrics:   []core.Metric{},
	}

	// Get namespaces to check
	namespaces, err := p.getNamespacesToCheck(ctx, client)
	if err != nil {
		return result, fmt.Errorf("failed to get namespaces: %w", err)
	}

	var totalPods, runningPods, failedPods, pendingPods int
	var highRestartPods []string
	podsByNamespace := make(map[string]int)

	// Check pods in each namespace
	for _, ns := range namespaces {
		pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return result, fmt.Errorf("failed to list pods in namespace %s: %w", ns, err)
		}

		for _, pod := range pods.Items {
			totalPods++
			podsByNamespace[ns]++

			// Check pod status
			switch pod.Status.Phase {
			case corev1.PodRunning:
				if p.isPodReady(&pod) {
					runningPods++
				} else {
					pendingPods++
				}
			case corev1.PodFailed:
				failedPods++
			case corev1.PodPending:
				// Check if pending pod is actually problematic
				if p.isPodProblematic(&pod) {
					failedPods++ // Count problematic pending pods as failed
				} else {
					pendingPods++
				}
			}

			// Check restart count
			restarts := p.getRestartCount(&pod)
			if restarts > p.restartThreshold {
				highRestartPods = append(highRestartPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
			}
		}
	}

	// Calculate health status
	failureRate := float64(failedPods) / float64(totalPods)
	pendingRate := float64(pendingPods) / float64(totalPods)

	if failureRate > 0.1 {
		result.Status = core.HealthStatusUnhealthy
		result.Message = fmt.Sprintf("High pod failure rate: %.1f%% (%d failed)", failureRate*100, failedPods)
	} else if failureRate > 0.05 || len(highRestartPods) > 0 || pendingRate > 0.2 {
		result.Status = core.HealthStatusDegraded
		issues := []string{}
		if failedPods > 0 {
			issues = append(issues, fmt.Sprintf("%d failed", failedPods))
		}
		if len(highRestartPods) > 0 {
			issues = append(issues, fmt.Sprintf("%d high-restart", len(highRestartPods)))
		}
		if pendingPods > 0 {
			issues = append(issues, fmt.Sprintf("%d pending", pendingPods))
		}
		result.Message = fmt.Sprintf("Pod issues detected: %s", strings.Join(issues, ", "))
	} else {
		result.Message = fmt.Sprintf("All pods are healthy (%d running, %d total)", runningPods, totalPods)
	}

	// Add details
	result.Details["total_pods"] = totalPods
	result.Details["running_pods"] = runningPods
	result.Details["failed_pods"] = failedPods
	result.Details["pending_pods"] = pendingPods
	result.Details["pods_by_namespace"] = podsByNamespace
	if len(highRestartPods) > 0 {
		result.Details["high_restart_pods"] = highRestartPods
	}

	// Add metrics
	result.Metrics = append(result.Metrics,
		core.Metric{
			Name:      "pod_total",
			Value:     float64(totalPods),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
		core.Metric{
			Name:      "pod_running",
			Value:     float64(runningPods),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
		core.Metric{
			Name:      "pod_failed",
			Value:     float64(failedPods),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
		core.Metric{
			Name:      "pod_failure_rate",
			Value:     failureRate,
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
	)

	result.Confidence = 1.0 // High confidence for direct API checks

	return result, nil
}

// Configure sets up the health check with configuration
func (p *PodHealthCheck) Configure(config map[string]interface{}) error {
	if v, ok := config["namespace"].(string); ok {
		p.namespace = v
	}
	if v, ok := config["restart_threshold"].(int); ok {
		// Validate range to prevent integer overflow
		if v >= 0 && v <= int(^uint32(0)>>1) {
			p.restartThreshold = int32(v)
		}
	}
	if v, ok := config["exclude_namespaces"].([]string); ok {
		p.excludeNamespaces = v
	}
	if v, ok := config["include_only_namespaces"].([]string); ok {
		p.includeOnlyNamespaces = v
	}
	return nil
}

// Interval returns how often this check should run
func (p *PodHealthCheck) Interval() time.Duration {
	return p.interval
}

// Criticality returns the importance level of this check
func (p *PodHealthCheck) Criticality() core.Criticality {
	return core.CriticalityHigh
}

// getNamespacesToCheck returns the list of namespaces to check
func (p *PodHealthCheck) getNamespacesToCheck(ctx context.Context, client kubernetes.Interface) ([]string, error) {
	// If specific namespaces are configured, use those
	if len(p.includeOnlyNamespaces) > 0 {
		return p.includeOnlyNamespaces, nil
	}

	// If a single namespace is specified, use it
	if p.namespace != "" {
		return []string{p.namespace}, nil
	}

	// Otherwise, get all namespaces and apply exclusions
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []string
	excludeMap := make(map[string]bool)
	for _, ns := range p.excludeNamespaces {
		excludeMap[ns] = true
	}

	for _, ns := range namespaces.Items {
		if !excludeMap[ns.Name] {
			result = append(result, ns.Name)
		}
	}

	return result, nil
}

// isPodReady checks if all containers in a pod are ready
func (p *PodHealthCheck) isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// getRestartCount returns the total restart count for all containers in a pod
func (p *PodHealthCheck) getRestartCount(pod *corev1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

// isPodProblematic checks if a pending pod has problematic conditions
func (p *PodHealthCheck) isPodProblematic(pod *corev1.Pod) bool {
	// Check container statuses for problems
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			reason := containerStatus.State.Waiting.Reason
			// These reasons indicate problematic pods
			switch reason {
			case "ImagePullBackOff", "ErrImagePull", "InvalidImageName",
				"CrashLoopBackOff", "RunContainerError", "CreateContainerError":
				return true
			}
		}
	}

	// Check init container statuses
	for _, initStatus := range pod.Status.InitContainerStatuses {
		if initStatus.State.Waiting != nil {
			reason := initStatus.State.Waiting.Reason
			switch reason {
			case "ImagePullBackOff", "ErrImagePull", "InvalidImageName",
				"CrashLoopBackOff", "RunContainerError", "CreateContainerError":
				return true
			}
		}
	}

	// Check pod conditions for problems
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
			// Pod can't be scheduled - could indicate resource constraints
			if condition.Reason == "Unschedulable" {
				return true
			}
		}
	}

	return false
}
