package ai

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
	
	"k8s.io/klog/v2"
)

// KubectlExecutor implements CommandExecutor using kubectl
type KubectlExecutor struct {
	kubectlPath string
	namespace   string
	dryRunMode  bool
}

// NewKubectlExecutor creates a new kubectl executor
func NewKubectlExecutor(namespace string) *KubectlExecutor {
	return &KubectlExecutor{
		kubectlPath: "kubectl",
		namespace:   namespace,
		dryRunMode:  false,
	}
}

// Execute runs a kubectl command
func (k *KubectlExecutor) Execute(ctx context.Context, command string) (string, error) {
	if k.dryRunMode {
		return k.DryRun(ctx, command)
	}
	
	// Parse and validate command
	if !strings.HasPrefix(command, "kubectl") {
		return "", fmt.Errorf("only kubectl commands are supported")
	}
	
	// Add namespace if not present
	if k.namespace != "" && !strings.Contains(command, "-n ") && !strings.Contains(command, "--namespace") {
		command = fmt.Sprintf("%s -n %s", command, k.namespace)
	}
	
	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	klog.V(2).Infof("Executing: %s", command)
	
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w, output: %s", err, output)
	}
	
	return string(output), nil
}

// DryRun simulates command execution
func (k *KubectlExecutor) DryRun(ctx context.Context, command string) (string, error) {
	// Add --dry-run flag
	if !strings.Contains(command, "--dry-run") {
		command = strings.Replace(command, "kubectl", "kubectl --dry-run=client", 1)
	}
	
	klog.V(2).Infof("Dry run: %s", command)
	
	// For testing, simulate some common outputs
	if strings.Contains(command, "scale") {
		return "deployment.apps/test scaled (dry run)", nil
	}
	if strings.Contains(command, "delete") {
		return "pod \"test\" deleted (dry run)", nil
	}
	if strings.Contains(command, "apply") {
		return "configmap/test configured (dry run)", nil
	}
	
	return fmt.Sprintf("Command would execute: %s", command), nil
}

// DefaultSafetyChecker implements SafetyChecker with safety rules
type DefaultSafetyChecker struct {
	allowedVerbs      []string
	allowedResources  []string
	dangerousCommands []string
}

// NewDefaultSafetyChecker creates a safety checker with default rules
func NewDefaultSafetyChecker() *DefaultSafetyChecker {
	return &DefaultSafetyChecker{
		allowedVerbs: []string{
			"get", "describe", "logs", "top", "scale", "restart", "rollout",
		},
		allowedResources: []string{
			"pods", "deployments", "services", "configmaps", "secrets",
			"replicasets", "statefulsets", "daemonsets", "jobs", "cronjobs",
		},
		dangerousCommands: []string{
			"delete namespace", "delete pv", "delete node",
			"drain", "cordon", "taint",
		},
	}
}

// IsSafe validates if an action is safe to execute
func (s *DefaultSafetyChecker) IsSafe(action *RemediationAction) (bool, string) {
	// Check risk level
	if action.Risk == RiskHigh && action.Confidence < 0.9 {
		return false, fmt.Sprintf("High risk action with insufficient confidence: %.2f", action.Confidence)
	}
	
	// Validate each command
	for _, cmd := range action.Commands {
		if err := s.ValidateCommand(cmd); err != nil {
			return false, err.Error()
		}
	}
	
	// Check for production safeguards
	if action.Type == "delete" || action.Type == "drain" {
		if !action.RequiresApproval {
			return false, "Destructive actions must require approval"
		}
	}
	
	return true, ""
}

// ValidateCommand checks if a command is safe
func (s *DefaultSafetyChecker) ValidateCommand(command string) error {
	command = strings.ToLower(command)
	
	// Check for dangerous commands
	for _, dangerous := range s.dangerousCommands {
		if strings.Contains(command, dangerous) {
			return fmt.Errorf("dangerous command pattern detected: %s", dangerous)
		}
	}
	
	// Validate kubectl commands
	if strings.HasPrefix(command, "kubectl") {
		parts := strings.Fields(command)
		if len(parts) < 3 {
			return fmt.Errorf("invalid kubectl command format")
		}
		
		verb := parts[1]
		
		// Check if verb is allowed
		allowed := false
		for _, allowedVerb := range s.allowedVerbs {
			if verb == allowedVerb {
				allowed = true
				break
			}
		}
		
		if !allowed {
			return fmt.Errorf("verb '%s' is not in allowed list", verb)
		}
		
		// Special checks for scale commands
		if verb == "scale" && strings.Contains(command, "replicas=0") {
			return fmt.Errorf("scaling to zero replicas requires manual approval")
		}
	}
	
	return nil
}