package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// KubectlTool interface for kubectl-based analysis modules
type KubectlTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, clusterName string) (*KubectlToolResult, error)
	GetCommands() []string
	RequiredPermissions() []string
	Category() string
	Priority() int // Lower number = higher priority
}

// ControlPlaneAnalyzer analyzes Kubernetes control plane health
type ControlPlaneAnalyzer struct {
	executor *KubectlExecutor
	database *Database
}

// NewControlPlaneAnalyzer creates a new control plane analyzer
func NewControlPlaneAnalyzer(executor *KubectlExecutor, database *Database) *ControlPlaneAnalyzer {
	return &ControlPlaneAnalyzer{
		executor: executor,
		database: database,
	}
}

func (c *ControlPlaneAnalyzer) Name() string {
	return "control-plane"
}

func (c *ControlPlaneAnalyzer) Description() string {
	return "Analyzes Kubernetes control plane components including API server, etcd, scheduler, and controller manager health"
}

func (c *ControlPlaneAnalyzer) Category() string {
	return "infrastructure"
}

func (c *ControlPlaneAnalyzer) Priority() int {
	return 1 // Highest priority - control plane is critical
}

func (c *ControlPlaneAnalyzer) GetCommands() []string {
	return []string{
		"kubectl get --raw='/livez?verbose'",
		"kubectl get --raw='/readyz?verbose'",
		"kubectl get componentstatuses",
		"kubectl get --raw='/metrics' | head -200",
		"kubectl cluster-info",
		"kubectl version --short",
	}
}

func (c *ControlPlaneAnalyzer) RequiredPermissions() []string {
	return []string{
		"get /livez",
		"get /readyz",
		"get componentstatuses",
		"get /metrics",
		"cluster-info",
	}
}

func (c *ControlPlaneAnalyzer) Execute(ctx context.Context, clusterName string) (*KubectlToolResult, error) {
	start := time.Now()

	klog.V(2).Infof("Executing control plane analysis for cluster: %s", clusterName)

	commands := c.GetCommands()
	executions, err := c.executor.ExecuteBatch(ctx, clusterName, commands)
	if err != nil {
		return nil, fmt.Errorf("failed to execute control plane commands: %w", err)
	}

	result := &KubectlToolResult{
		ToolName:      c.Name(),
		Commands:      commands,
		Outputs:       make(map[string]string),
		Errors:        make(map[string]string),
		ExecutionTime: time.Since(start),
		Success:       true,
		Metadata:      make(map[string]interface{}),
	}

	// Process execution results
	successCount := 0
	for cmd, execution := range executions {
		if execution.Success {
			result.Outputs[cmd] = execution.Output
			successCount++
		} else {
			result.Errors[cmd] = execution.ErrorMessage
			result.Success = false // Mark overall as failed if any command fails
		}
	}

	// Analyze outputs and create summary
	result.Summary = c.generateSummary(result.Outputs, result.Errors)
	result.Metadata["success_rate"] = float64(successCount) / float64(len(commands))
	result.Metadata["cluster_name"] = clusterName
	result.Metadata["analysis_time"] = time.Now().Format(time.RFC3339)

	// Extract key metrics
	c.extractMetrics(result)

	klog.V(2).Infof("Control plane analysis completed for %s: success_rate=%.2f, duration=%v",
		clusterName, result.Metadata["success_rate"], result.ExecutionTime)

	return result, nil
}

// generateSummary creates a human-readable summary of control plane health
func (c *ControlPlaneAnalyzer) generateSummary(outputs, errors map[string]string) string {
	var summary strings.Builder

	summary.WriteString("Control Plane Health Analysis:\n")

	// Check API server liveness
	if livez, ok := outputs["kubectl get --raw='/livez?verbose'"]; ok {
		if strings.Contains(livez, "livez check passed") || strings.Contains(livez, "ok") {
			summary.WriteString("✓ API Server is alive and responding\n")
		} else {
			summary.WriteString("✗ API Server liveness check failed\n")
		}
	} else {
		summary.WriteString("⚠ API Server liveness check could not be performed\n")
	}

	// Check API server readiness
	if readyz, ok := outputs["kubectl get --raw='/readyz?verbose'"]; ok {
		if strings.Contains(readyz, "readyz check passed") || strings.Contains(readyz, "ok") {
			summary.WriteString("✓ API Server is ready to serve requests\n")
		} else {
			summary.WriteString("✗ API Server readiness check failed\n")
		}
	} else {
		summary.WriteString("⚠ API Server readiness check could not be performed\n")
	}

	// Check component status
	if cs, ok := outputs["kubectl get componentstatuses"]; ok {
		healthyComponents := strings.Count(cs, "Healthy")
		if healthyComponents > 0 {
			summary.WriteString(fmt.Sprintf("✓ %d core components are healthy\n", healthyComponents))
		}
		if strings.Contains(cs, "Unhealthy") {
			summary.WriteString("✗ Some core components are unhealthy\n")
		}
	} else {
		summary.WriteString("⚠ Component status check could not be performed\n")
	}

	// Check for errors
	if len(errors) > 0 {
		summary.WriteString(fmt.Sprintf("⚠ %d commands failed to execute\n", len(errors)))
	}

	return summary.String()
}

// extractMetrics extracts key metrics from the outputs
func (c *ControlPlaneAnalyzer) extractMetrics(result *KubectlToolResult) {
	metrics := make(map[string]interface{})

	// Parse component status
	if cs, ok := result.Outputs["kubectl get componentstatuses"]; ok {
		metrics["healthy_components"] = strings.Count(cs, "Healthy")
		metrics["unhealthy_components"] = strings.Count(cs, "Unhealthy")
	}

	// Parse version information
	if version, ok := result.Outputs["kubectl version --short"]; ok {
		lines := strings.Split(version, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Server Version") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					metrics["kubernetes_version"] = strings.TrimSpace(parts[1])
				}
			}
		}
	}

	// Check API server responsiveness
	metrics["api_server_responsive"] = result.Outputs["kubectl get --raw='/livez?verbose'"] != ""

	// Basic metrics parsing from /metrics endpoint
	if metricsOutput, ok := result.Outputs["kubectl get --raw='/metrics' | head -200"]; ok {
		// Count HTTP requests
		metrics["metrics_available"] = len(strings.Split(metricsOutput, "\n")) > 10
	}

	result.Metadata["extracted_metrics"] = metrics
}

// NodeAnalyzer analyzes Kubernetes node health and capacity
type NodeAnalyzer struct {
	executor *KubectlExecutor
	database *Database
}

// NewNodeAnalyzer creates a new node analyzer
func NewNodeAnalyzer(executor *KubectlExecutor, database *Database) *NodeAnalyzer {
	return &NodeAnalyzer{
		executor: executor,
		database: database,
	}
}

func (n *NodeAnalyzer) Name() string {
	return "nodes"
}

func (n *NodeAnalyzer) Description() string {
	return "Analyzes Kubernetes node health, capacity, resource usage, and conditions"
}

func (n *NodeAnalyzer) Category() string {
	return "infrastructure"
}

func (n *NodeAnalyzer) Priority() int {
	return 2 // High priority - nodes are critical
}

func (n *NodeAnalyzer) GetCommands() []string {
	return []string{
		"kubectl get nodes -o wide",
		"kubectl top nodes",
		"kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{\"\\t\"}{.status.conditions[?(@.type==\"Ready\")].status}{\"\\n\"}{end}'",
		"kubectl describe nodes",
	}
}

func (n *NodeAnalyzer) RequiredPermissions() []string {
	return []string{
		"get nodes",
		"list nodes",
		"describe nodes",
	}
}

func (n *NodeAnalyzer) Execute(ctx context.Context, clusterName string) (*KubectlToolResult, error) {
	start := time.Now()

	klog.V(2).Infof("Executing node analysis for cluster: %s", clusterName)

	commands := n.GetCommands()
	executions, err := n.executor.ExecuteBatch(ctx, clusterName, commands)
	if err != nil {
		return nil, fmt.Errorf("failed to execute node commands: %w", err)
	}

	result := &KubectlToolResult{
		ToolName:      n.Name(),
		Commands:      commands,
		Outputs:       make(map[string]string),
		Errors:        make(map[string]string),
		ExecutionTime: time.Since(start),
		Success:       true,
		Metadata:      make(map[string]interface{}),
	}

	// Process execution results
	successCount := 0
	for cmd, execution := range executions {
		if execution.Success {
			result.Outputs[cmd] = execution.Output
			successCount++
		} else {
			result.Errors[cmd] = execution.ErrorMessage
			if !strings.Contains(execution.ErrorMessage, "metrics-server") {
				// Don't fail overall analysis if only metrics-server is missing
				result.Success = false
			}
		}
	}

	result.Summary = n.generateSummary(result.Outputs, result.Errors)
	result.Metadata["success_rate"] = float64(successCount) / float64(len(commands))
	result.Metadata["cluster_name"] = clusterName

	n.extractMetrics(result)

	klog.V(2).Infof("Node analysis completed for %s: success_rate=%.2f, duration=%v",
		clusterName, result.Metadata["success_rate"], result.ExecutionTime)

	return result, nil
}

func (n *NodeAnalyzer) generateSummary(outputs, errors map[string]string) string {
	var summary strings.Builder

	summary.WriteString("Node Health Analysis:\n")

	// Analyze node list
	if nodes, ok := outputs["kubectl get nodes -o wide"]; ok {
		lines := strings.Split(nodes, "\n")
		if len(lines) > 1 { // Skip header
			nodeCount := len(lines) - 1
			readyCount := strings.Count(nodes, " Ready ")
			summary.WriteString(fmt.Sprintf("✓ Total nodes: %d\n", nodeCount))
			summary.WriteString(fmt.Sprintf("✓ Ready nodes: %d\n", readyCount))

			if notReadyCount := strings.Count(nodes, " NotReady "); notReadyCount > 0 {
				summary.WriteString(fmt.Sprintf("✗ NotReady nodes: %d\n", notReadyCount))
			}
		}
	}

	// Analyze resource usage if available
	if top, ok := outputs["kubectl top nodes"]; ok {
		if !strings.Contains(top, "error") && !strings.Contains(top, "not found") {
			summary.WriteString("✓ Node resource usage metrics available\n")
			// Parse CPU and memory usage
			lines := strings.Split(top, "\n")
			if len(lines) > 1 {
				summary.WriteString(fmt.Sprintf("✓ Resource metrics for %d nodes\n", len(lines)-1))
			}
		} else {
			summary.WriteString("⚠ Node resource usage metrics not available\n")
		}
	}

	if len(errors) > 0 {
		summary.WriteString(fmt.Sprintf("⚠ %d commands had issues\n", len(errors)))
	}

	return summary.String()
}

func (n *NodeAnalyzer) extractMetrics(result *KubectlToolResult) {
	metrics := make(map[string]interface{})

	// Count nodes by status
	if nodes, ok := result.Outputs["kubectl get nodes -o wide"]; ok {
		lines := strings.Split(nodes, "\n")
		if len(lines) > 1 {
			metrics["total_nodes"] = len(lines) - 1
			metrics["ready_nodes"] = strings.Count(nodes, " Ready ")
			metrics["not_ready_nodes"] = strings.Count(nodes, " NotReady ")
		}
	}

	// Parse resource usage
	if top, ok := result.Outputs["kubectl top nodes"]; ok {
		if !strings.Contains(top, "error") {
			lines := strings.Split(top, "\n")
			if len(lines) > 1 {
				metrics["metrics_available"] = true
				metrics["nodes_with_metrics"] = len(lines) - 1
			}
		} else {
			metrics["metrics_available"] = false
		}
	}

	result.Metadata["extracted_metrics"] = metrics
}

// WorkloadAnalyzer analyzes Kubernetes workload health
type WorkloadAnalyzer struct {
	executor *KubectlExecutor
	database *Database
}

// NewWorkloadAnalyzer creates a new workload analyzer
func NewWorkloadAnalyzer(executor *KubectlExecutor, database *Database) *WorkloadAnalyzer {
	return &WorkloadAnalyzer{
		executor: executor,
		database: database,
	}
}

func (w *WorkloadAnalyzer) Name() string {
	return "workloads"
}

func (w *WorkloadAnalyzer) Description() string {
	return "Analyzes Kubernetes workload health including pods, deployments, services, and resource usage"
}

func (w *WorkloadAnalyzer) Category() string {
	return "workloads"
}

func (w *WorkloadAnalyzer) Priority() int {
	return 3 // Medium priority - after infrastructure
}

func (w *WorkloadAnalyzer) GetCommands() []string {
	return []string{
		"kubectl get pods -A -o wide",
		"kubectl get deployments -A",
		"kubectl get replicasets -A",
		"kubectl get services -A",
		"kubectl get events -A --sort-by=.lastTimestamp | tail -20",
		"kubectl top pods -A --sort-by=cpu | head -20",
		"kubectl get pods -A --field-selector status.phase!=Running",
	}
}

func (w *WorkloadAnalyzer) RequiredPermissions() []string {
	return []string{
		"get pods",
		"list pods",
		"get deployments",
		"list deployments",
		"get replicasets",
		"list replicasets",
		"get services",
		"list services",
		"get events",
		"list events",
	}
}

func (w *WorkloadAnalyzer) Execute(ctx context.Context, clusterName string) (*KubectlToolResult, error) {
	start := time.Now()

	klog.V(2).Infof("Executing workload analysis for cluster: %s", clusterName)

	commands := w.GetCommands()
	executions, err := w.executor.ExecuteBatch(ctx, clusterName, commands)
	if err != nil {
		return nil, fmt.Errorf("failed to execute workload commands: %w", err)
	}

	result := &KubectlToolResult{
		ToolName:      w.Name(),
		Commands:      commands,
		Outputs:       make(map[string]string),
		Errors:        make(map[string]string),
		ExecutionTime: time.Since(start),
		Success:       true,
		Metadata:      make(map[string]interface{}),
	}

	// Process execution results
	successCount := 0
	for cmd, execution := range executions {
		if execution.Success {
			result.Outputs[cmd] = execution.Output
			successCount++
		} else {
			result.Errors[cmd] = execution.ErrorMessage
			// Don't fail overall if only metrics-server commands fail
			if !strings.Contains(execution.ErrorMessage, "metrics-server") &&
				!strings.Contains(cmd, "top pods") {
				result.Success = false
			}
		}
	}

	result.Summary = w.generateSummary(result.Outputs, result.Errors)
	result.Metadata["success_rate"] = float64(successCount) / float64(len(commands))
	result.Metadata["cluster_name"] = clusterName

	w.extractMetrics(result)

	klog.V(2).Infof("Workload analysis completed for %s: success_rate=%.2f, duration=%v",
		clusterName, result.Metadata["success_rate"], result.ExecutionTime)

	return result, nil
}

func (w *WorkloadAnalyzer) generateSummary(outputs, errors map[string]string) string {
	var summary strings.Builder

	summary.WriteString("Workload Health Analysis:\n")

	// Analyze pods
	if pods, ok := outputs["kubectl get pods -A -o wide"]; ok {
		lines := strings.Split(pods, "\n")
		if len(lines) > 1 {
			podCount := len(lines) - 1
			runningCount := strings.Count(pods, " Running ")
			summary.WriteString(fmt.Sprintf("✓ Total pods: %d\n", podCount))
			summary.WriteString(fmt.Sprintf("✓ Running pods: %d\n", runningCount))

			// Check for problematic pods
			if problemPods, ok := outputs["kubectl get pods -A --field-selector status.phase!=Running"]; ok {
				problemLines := strings.Split(problemPods, "\n")
				if len(problemLines) > 1 && problemLines[1] != "" {
					summary.WriteString(fmt.Sprintf("⚠ Non-running pods: %d\n", len(problemLines)-1))
				}
			}
		}
	}

	// Analyze deployments
	if deployments, ok := outputs["kubectl get deployments -A"]; ok {
		lines := strings.Split(deployments, "\n")
		if len(lines) > 1 {
			deploymentCount := len(lines) - 1
			summary.WriteString(fmt.Sprintf("✓ Total deployments: %d\n", deploymentCount))
		}
	}

	// Analyze services
	if services, ok := outputs["kubectl get services -A"]; ok {
		lines := strings.Split(services, "\n")
		if len(lines) > 1 {
			serviceCount := len(lines) - 1
			summary.WriteString(fmt.Sprintf("✓ Total services: %d\n", serviceCount))
		}
	}

	// Check resource usage if available
	if top, ok := outputs["kubectl top pods -A --sort-by=cpu | head -20"]; ok {
		if !strings.Contains(top, "error") && !strings.Contains(top, "not found") {
			summary.WriteString("✓ Pod resource usage metrics available\n")
		} else {
			summary.WriteString("⚠ Pod resource usage metrics not available\n")
		}
	}

	// Check recent events
	if events, ok := outputs["kubectl get events -A --sort-by=.lastTimestamp | tail -20"]; ok {
		if strings.Contains(events, "Warning") {
			warningCount := strings.Count(events, " Warning ")
			summary.WriteString(fmt.Sprintf("⚠ Recent warning events: %d\n", warningCount))
		}
		if strings.Contains(events, "Error") {
			errorCount := strings.Count(events, " Error ")
			summary.WriteString(fmt.Sprintf("✗ Recent error events: %d\n", errorCount))
		}
	}

	if len(errors) > 0 {
		summary.WriteString(fmt.Sprintf("⚠ %d commands had issues\n", len(errors)))
	}

	return summary.String()
}

func (w *WorkloadAnalyzer) extractMetrics(result *KubectlToolResult) {
	metrics := make(map[string]interface{})

	// Count workloads by type
	if pods, ok := result.Outputs["kubectl get pods -A -o wide"]; ok {
		lines := strings.Split(pods, "\n")
		if len(lines) > 1 {
			metrics["total_pods"] = len(lines) - 1
			metrics["running_pods"] = strings.Count(pods, " Running ")
			metrics["pending_pods"] = strings.Count(pods, " Pending ")
			metrics["failed_pods"] = strings.Count(pods, " Failed ")
		}
	}

	if deployments, ok := result.Outputs["kubectl get deployments -A"]; ok {
		lines := strings.Split(deployments, "\n")
		if len(lines) > 1 {
			metrics["total_deployments"] = len(lines) - 1
		}
	}

	if services, ok := result.Outputs["kubectl get services -A"]; ok {
		lines := strings.Split(services, "\n")
		if len(lines) > 1 {
			metrics["total_services"] = len(lines) - 1
		}
	}

	// Check for problem pods
	if problemPods, ok := result.Outputs["kubectl get pods -A --field-selector status.phase!=Running"]; ok {
		lines := strings.Split(problemPods, "\n")
		if len(lines) > 1 && lines[1] != "" {
			metrics["problem_pods"] = len(lines) - 1
		} else {
			metrics["problem_pods"] = 0
		}
	}

	// Count recent events
	if events, ok := result.Outputs["kubectl get events -A --sort-by=.lastTimestamp | tail -20"]; ok {
		metrics["recent_warning_events"] = strings.Count(events, " Warning ")
		metrics["recent_error_events"] = strings.Count(events, " Error ")
	}

	// Check metrics availability
	if top, ok := result.Outputs["kubectl top pods -A --sort-by=cpu | head -20"]; ok {
		metrics["resource_metrics_available"] = !strings.Contains(top, "error") && !strings.Contains(top, "not found")
	}

	result.Metadata["extracted_metrics"] = metrics
}

// ToolRegistry manages available kubectl tools
type ToolRegistry struct {
	tools    map[string]KubectlTool
	executor *KubectlExecutor
	database *Database
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(executor *KubectlExecutor, database *Database) *ToolRegistry {
	registry := &ToolRegistry{
		tools:    make(map[string]KubectlTool),
		executor: executor,
		database: database,
	}

	// Register built-in tools
	registry.RegisterTool(NewControlPlaneAnalyzer(executor, database))
	registry.RegisterTool(NewNodeAnalyzer(executor, database))
	registry.RegisterTool(NewWorkloadAnalyzer(executor, database))

	return registry
}

// RegisterTool registers a new kubectl tool
func (r *ToolRegistry) RegisterTool(tool KubectlTool) {
	r.tools[tool.Name()] = tool
	klog.V(2).Infof("Registered kubectl tool: %s", tool.Name())
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (KubectlTool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// ListTools returns all registered tools
func (r *ToolRegistry) ListTools() []KubectlTool {
	tools := make([]KubectlTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteAll executes all registered tools
func (r *ToolRegistry) ExecuteAll(ctx context.Context, clusterName string) (map[string]*KubectlToolResult, error) {
	results := make(map[string]*KubectlToolResult)

	for name, tool := range r.tools {
		result, err := tool.Execute(ctx, clusterName)
		if err != nil {
			klog.Errorf("Tool %s failed: %v", name, err)
			// Continue with other tools
			results[name] = &KubectlToolResult{
				ToolName:      name,
				Success:       false,
				Errors:        map[string]string{"execution": err.Error()},
				ExecutionTime: 0,
			}
		} else {
			results[name] = result
		}
	}

	return results, nil
}

// GetToolsManifest returns information about all tools
func (r *ToolRegistry) GetToolsManifest() map[string]interface{} {
	manifest := make(map[string]interface{})

	for name, tool := range r.tools {
		manifest[name] = map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"category":    tool.Category(),
			"priority":    tool.Priority(),
			"commands":    tool.GetCommands(),
			"permissions": tool.RequiredPermissions(),
		}
	}

	return manifest
}
