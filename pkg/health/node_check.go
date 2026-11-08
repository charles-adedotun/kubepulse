package health

import (
	"context"
	"fmt"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NodeHealthCheck checks the health of nodes in the cluster
type NodeHealthCheck struct {
	cpuThreshold    float64
	memoryThreshold float64
	diskThreshold   float64
	interval        time.Duration
}

// NewNodeHealthCheck creates a new node health check
func NewNodeHealthCheck() *NodeHealthCheck {
	return &NodeHealthCheck{
		cpuThreshold:    80.0,
		memoryThreshold: 85.0,
		diskThreshold:   90.0,
		interval:        30 * time.Second,
	}
}

// Name returns the name of the health check
func (n *NodeHealthCheck) Name() string {
	return "node-health"
}

// Description returns a description of the health check
func (n *NodeHealthCheck) Description() string {
	return "Monitors node conditions, resource usage, and readiness"
}

// Check performs the node health check
func (n *NodeHealthCheck) Check(ctx context.Context, client kubernetes.Interface) (core.CheckResult, error) {
	result := core.CheckResult{
		Name:      n.Name(),
		Timestamp: time.Now(),
		Status:    core.HealthStatusHealthy,
		Details:   make(map[string]interface{}),
		Metrics:   []core.Metric{},
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to list nodes: %w", err)
	}

	var readyNodes, notReadyNodes int
	var nodeIssues []string
	nodeDetails := make([]map[string]interface{}, 0)

	for _, node := range nodes.Items {
		nodeInfo := map[string]interface{}{
			"name": node.Name,
		}

		// Check node conditions
		isReady := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				isReady = condition.Status == corev1.ConditionTrue
				if isReady {
					readyNodes++
				} else {
					notReadyNodes++
					nodeIssues = append(nodeIssues, fmt.Sprintf("%s: NotReady", node.Name))
				}
			}

			// Check for other problematic conditions
			if condition.Type == corev1.NodeMemoryPressure && condition.Status == corev1.ConditionTrue {
				nodeIssues = append(nodeIssues, fmt.Sprintf("%s: MemoryPressure", node.Name))
			}
			if condition.Type == corev1.NodeDiskPressure && condition.Status == corev1.ConditionTrue {
				nodeIssues = append(nodeIssues, fmt.Sprintf("%s: DiskPressure", node.Name))
			}
			if condition.Type == corev1.NodePIDPressure && condition.Status == corev1.ConditionTrue {
				nodeIssues = append(nodeIssues, fmt.Sprintf("%s: PIDPressure", node.Name))
			}
		}

		// Calculate resource usage
		allocatable := node.Status.Allocatable
		capacity := node.Status.Capacity

		cpuAllocatable := allocatable.Cpu().MilliValue()
		cpuCapacity := capacity.Cpu().MilliValue()
		memoryAllocatable := allocatable.Memory().Value()
		memoryCapacity := capacity.Memory().Value()

		// Get current resource usage from metrics
		cpuUsage, memoryUsage := n.getNodeResourceUsage(ctx, client, node.Name)

		cpuPercent := float64(cpuUsage) / float64(cpuCapacity) * 100
		memoryPercent := float64(memoryUsage) / float64(memoryCapacity) * 100

		nodeInfo["ready"] = isReady
		nodeInfo["cpu_percent"] = cpuPercent
		nodeInfo["memory_percent"] = memoryPercent
		nodeInfo["cpu_allocatable"] = cpuAllocatable
		nodeInfo["memory_allocatable"] = memoryAllocatable

		// Check thresholds
		if cpuPercent > n.cpuThreshold {
			nodeIssues = append(nodeIssues, fmt.Sprintf("%s: High CPU usage (%.1f%%)", node.Name, cpuPercent))
		}
		if memoryPercent > n.memoryThreshold {
			nodeIssues = append(nodeIssues, fmt.Sprintf("%s: High memory usage (%.1f%%)", node.Name, memoryPercent))
		}

		nodeDetails = append(nodeDetails, nodeInfo)

		// Add node-specific metrics
		result.Metrics = append(result.Metrics,
			core.Metric{
				Name:  "node_cpu_usage_percent",
				Value: cpuPercent,
				Labels: map[string]string{
					"node": node.Name,
				},
				Type:      core.MetricTypeGauge,
				Timestamp: time.Now(),
			},
			core.Metric{
				Name:  "node_memory_usage_percent",
				Value: memoryPercent,
				Labels: map[string]string{
					"node": node.Name,
				},
				Type:      core.MetricTypeGauge,
				Timestamp: time.Now(),
			},
		)
	}

	// Determine overall status
	totalNodes := len(nodes.Items)
	if notReadyNodes > 0 {
		result.Status = core.HealthStatusDegraded
		result.Message = fmt.Sprintf("%d of %d nodes are not ready", notReadyNodes, totalNodes)
	} else if len(nodeIssues) > 0 {
		result.Status = core.HealthStatusDegraded
		result.Message = "Some nodes have resource pressure"
	} else {
		result.Message = fmt.Sprintf("All %d nodes are healthy", totalNodes)
	}

	// Add summary details
	result.Details["total_nodes"] = totalNodes
	result.Details["ready_nodes"] = readyNodes
	result.Details["not_ready_nodes"] = notReadyNodes
	result.Details["nodes"] = nodeDetails
	if len(nodeIssues) > 0 {
		result.Details["issues"] = nodeIssues
	}

	// Add summary metrics
	result.Metrics = append(result.Metrics,
		core.Metric{
			Name:      "node_total",
			Value:     float64(totalNodes),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
		core.Metric{
			Name:      "node_ready",
			Value:     float64(readyNodes),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
	)

	result.Confidence = 1.0

	return result, nil
}

// Configure sets up the health check with configuration
func (n *NodeHealthCheck) Configure(config map[string]interface{}) error {
	if v, ok := config["cpu_threshold"].(float64); ok {
		n.cpuThreshold = v
	}
	if v, ok := config["memory_threshold"].(float64); ok {
		n.memoryThreshold = v
	}
	if v, ok := config["disk_threshold"].(float64); ok {
		n.diskThreshold = v
	}
	return nil
}

// Interval returns how often this check should run
func (n *NodeHealthCheck) Interval() time.Duration {
	return n.interval
}

// Criticality returns the importance level of this check
func (n *NodeHealthCheck) Criticality() core.Criticality {
	return core.CriticalityCritical
}

// getNodeResourceUsage aggregates active pod resource requests scheduled on nodeName.
func (n *NodeHealthCheck) getNodeResourceUsage(ctx context.Context, client kubernetes.Interface, nodeName string) (cpuMillis int64, memoryBytes int64) {
	if client == nil || nodeName == "" {
		return 0, 0
	}

	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return 0, 0
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		for _, container := range pod.Spec.Containers {
			if cpuReq, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				cpuMillis += cpuReq.MilliValue()
			}
			if memReq, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				memoryBytes += memReq.Value()
			}
		}
	}

	return cpuMillis, memoryBytes
}

// Helper function to convert resource quantity to float64
// quantityToFloat64 is unused but kept for future resource calculations
// func quantityToFloat64(q *resource.Quantity) float64 {
// 	return float64(q.MilliValue()) / 1000.0
// }
