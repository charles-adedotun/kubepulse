package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// KubectlExecutor provides secure kubectl command execution
type KubectlExecutor struct {
	kubectlPath    string
	timeout        time.Duration
	allowedCommands []string
	database       *Database
}

// KubectlExecutorConfig holds configuration for the kubectl executor
type KubectlExecutorConfig struct {
	KubectlPath string
	Timeout     time.Duration
	Database    *Database
}

// NewKubectlExecutor creates a new kubectl executor
func NewKubectlExecutor(config KubectlExecutorConfig) *KubectlExecutor {
	if config.KubectlPath == "" {
		config.KubectlPath = "kubectl"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &KubectlExecutor{
		kubectlPath:     config.KubectlPath,
		timeout:         config.Timeout,
		allowedCommands: getAllowedCommands(),
		database:        config.Database,
	}
}

// Execute runs a kubectl command safely
func (e *KubectlExecutor) Execute(ctx context.Context, clusterName, command string) (*KubectlExecution, error) {
	start := time.Now()

	// Validate command is allowed
	if !e.isCommandAllowed(command) {
		return nil, fmt.Errorf("command not allowed: %s", command)
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 || parts[0] != "kubectl" {
		return nil, fmt.Errorf("invalid kubectl command: %s", command)
	}

	klog.V(3).Infof("Executing kubectl command: %s", command)

	// Execute command
	cmd := exec.CommandContext(ctx, e.kubectlPath, parts[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	executionTime := time.Since(start)

	execution := &KubectlExecution{
		ClusterName:   clusterName,
		Command:       command,
		Output:        stdout.String(),
		Success:       err == nil,
		ExecutionTime: executionTime,
		Timestamp:     start,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			execution.ErrorMessage = fmt.Sprintf("Command timed out after %v", e.timeout)
		} else {
			execution.ErrorMessage = fmt.Sprintf("Command failed: %v, stderr: %s", err, stderr.String())
		}
		klog.Errorf("kubectl command failed: %s, error: %s", command, execution.ErrorMessage)
	} else {
		klog.V(3).Infof("kubectl command completed successfully: %s (took %v)", command, executionTime)
	}

	// Store execution in database if available
	if e.database != nil {
		if err := e.storeExecution(execution); err != nil {
			klog.Errorf("Failed to store kubectl execution: %v", err)
		}
	}

	return execution, nil
}

// ExecuteBatch executes multiple kubectl commands
func (e *KubectlExecutor) ExecuteBatch(ctx context.Context, clusterName string, commands []string) (map[string]*KubectlExecution, error) {
	results := make(map[string]*KubectlExecution)

	for _, command := range commands {
		execution, err := e.Execute(ctx, clusterName, command)
		if err != nil {
			// Continue with other commands even if one fails
			execution = &KubectlExecution{
				ClusterName:   clusterName,
				Command:       command,
				Success:       false,
				ErrorMessage:  err.Error(),
				Timestamp:     time.Now(),
			}
		}
		results[command] = execution
	}

	return results, nil
}

// isCommandAllowed checks if a command is in the allowed list
func (e *KubectlExecutor) isCommandAllowed(command string) bool {
	// Normalize command for comparison
	normalized := strings.ToLower(strings.TrimSpace(command))

	for _, allowed := range e.allowedCommands {
		if strings.HasPrefix(normalized, strings.ToLower(allowed)) {
			return true
		}
	}

	return false
}

// storeExecution stores the kubectl execution in the database
func (e *KubectlExecutor) storeExecution(execution *KubectlExecution) error {
	query := `
		INSERT INTO kubectl_executions 
		(cluster_name, command, output, success, error_message, execution_time, analysis_session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := e.database.DB().Exec(query,
		execution.ClusterName,
		execution.Command,
		execution.Output,
		execution.Success,
		execution.ErrorMessage,
		execution.ExecutionTime.Milliseconds(),
		execution.AnalysisSessionID,
	)

	return err
}

// getAllowedCommands returns the list of allowed kubectl commands
// Based on the kubectl read-only runbook
func getAllowedCommands() []string {
	return []string{
		// Control-Plane Vitality
		"kubectl get --raw='/livez",
		"kubectl get --raw='/readyz",
		"kubectl get componentstatuses",
		"kubectl get cs",
		"kubectl get --raw='/metrics'",

		// Node Condition & Capacity
		"kubectl get nodes",
		"kubectl describe node",
		"kubectl top nodes",

		// Workload Health
		"kubectl get pods",
		"kubectl get deployments",
		"kubectl get replicasets",
		"kubectl get statefulsets",
		"kubectl get daemonsets",
		"kubectl describe pod",
		"kubectl describe deployment",
		"kubectl describe replicaset",
		"kubectl describe statefulset",
		"kubectl describe daemonset",
		"kubectl top pods",
		"kubectl rollout status",

		// Events & Recent Changes
		"kubectl get events",

		// Networking Snapshot
		"kubectl get services",
		"kubectl get svc",
		"kubectl get endpoints",
		"kubectl get ep",
		"kubectl get ingress",
		"kubectl get networkpolicies",
		"kubectl describe service",
		"kubectl describe svc",
		"kubectl describe ingress",

		// Storage Check-ups
		"kubectl get persistentvolumes",
		"kubectl get pv",
		"kubectl get persistentvolumeclaims",
		"kubectl get pvc",
		"kubectl describe pv",
		"kubectl describe pvc",
		"kubectl get storageclasses",
		"kubectl get sc",

		// Logs (read-only)
		"kubectl logs",

		// Configuration and Secrets (metadata only)
		"kubectl get configmaps",
		"kubectl get cm",
		"kubectl get secrets",
		"kubectl describe configmap",
		"kubectl describe cm",
		"kubectl describe secret",

		// RBAC (read-only)
		"kubectl get clusterroles",
		"kubectl get roles",
		"kubectl get clusterrolebindings",
		"kubectl get rolebindings",
		"kubectl get serviceaccounts",
		"kubectl get sa",

		// Custom Resources
		"kubectl get crd",
		"kubectl get customresourcedefinitions",

		// API Resources
		"kubectl api-resources",
		"kubectl api-versions",

		// Cluster Info
		"kubectl cluster-info",
		"kubectl version",

		// Metrics API (raw)
		"kubectl get --raw '/apis/metrics.k8s.io",
	}
}

// GetAllowedCommands returns the list of allowed commands (for documentation)
func (e *KubectlExecutor) GetAllowedCommands() []string {
	return e.allowedCommands
}

// ValidateCommand checks if a command would be allowed without executing it
func (e *KubectlExecutor) ValidateCommand(command string) error {
	if !e.isCommandAllowed(command) {
		return fmt.Errorf("command not allowed: %s", command)
	}

	parts := strings.Fields(command)
	if len(parts) == 0 || parts[0] != "kubectl" {
		return fmt.Errorf("invalid kubectl command format: %s", command)
	}

	return nil
}

// GetExecutionHistory retrieves kubectl execution history for a cluster
func (e *KubectlExecutor) GetExecutionHistory(clusterName string, since time.Time) ([]KubectlExecution, error) {
	if e.database == nil {
		return nil, fmt.Errorf("database not configured")
	}

	query := `
		SELECT cluster_name, command, output, success, error_message, 
		       execution_time, timestamp, analysis_session_id
		FROM kubectl_executions 
		WHERE cluster_name = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 100
	`

	rows, err := e.database.DB().Query(query, clusterName, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution history: %w", err)
	}
	defer rows.Close()

	var executions []KubectlExecution
	for rows.Next() {
		var execution KubectlExecution
		var executionTimeMs int64

		err := rows.Scan(
			&execution.ClusterName,
			&execution.Command,
			&execution.Output,
			&execution.Success,
			&execution.ErrorMessage,
			&executionTimeMs,
			&execution.Timestamp,
			&execution.AnalysisSessionID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		execution.ExecutionTime = time.Duration(executionTimeMs) * time.Millisecond
		executions = append(executions, execution)
	}

	return executions, nil
}