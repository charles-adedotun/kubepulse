package ai

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// RemediationKubectlExecutor implements CommandExecutor using kubectl for remediation
type RemediationKubectlExecutor struct {
	kubectlPath string
	namespace   string
	dryRunMode  bool
}

// NewRemediationKubectlExecutor creates a new kubectl executor for remediation
func NewRemediationKubectlExecutor(namespace string) *RemediationKubectlExecutor {
	// Securely find kubectl binary
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		klog.Warningf("kubectl not found in PATH: %v, using default", err)
		kubectlPath = "/usr/local/bin/kubectl"
	}

	return &RemediationKubectlExecutor{
		kubectlPath: kubectlPath,
		namespace:   namespace,
		dryRunMode:  false,
	}
}

// Execute runs a kubectl command
func (k *RemediationKubectlExecutor) Execute(ctx context.Context, command string) (string, error) {
	if k.dryRunMode {
		return k.DryRun(ctx, command)
	}

	// Parse and validate command
	if !strings.HasPrefix(command, "kubectl") {
		return "", fmt.Errorf("only kubectl commands are supported")
	}

	// Parse command into parts
	parts := strings.Fields(command)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid kubectl command format")
	}

	// Build args slice
	args := parts[1:] // Skip "kubectl"

	// Add namespace if not present
	if k.namespace != "" && !containsArg(args, "-n", "--namespace") {
		args = append(args, "-n", k.namespace)
	}

	// Validate arguments for security
	for _, arg := range args {
		if containsShellMetacharacters(arg) {
			return "", fmt.Errorf("invalid characters in command argument: %s", arg)
		}
		if strings.Contains(arg, "..") || (strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, "/dev/")) {
			return "", fmt.Errorf("path traversal attempt detected in argument: %s", arg)
		}
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	klog.V(2).Infof("Executing kubectl with args: %v", args)

	// Use direct command execution without shell
	cmd := exec.CommandContext(ctx, k.kubectlPath, args...) // #nosec G204 -- arguments are validated above
	output, err := cmd.CombinedOutput()

	if err != nil {
		return string(output), fmt.Errorf("command failed: %w, output: %s", err, output)
	}

	return string(output), nil
}

// DryRun simulates command execution
func (k *RemediationKubectlExecutor) DryRun(ctx context.Context, command string) (string, error) {
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

// containsArg checks if an argument slice contains a specific flag
func containsArg(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag || strings.HasPrefix(arg, flag+"=") {
				return true
			}
		}
	}
	return false
}

// containsShellMetacharacters checks if a string contains shell metacharacters
// that could be used for command injection
func containsShellMetacharacters(s string) bool {
	// List of dangerous shell metacharacters
	dangerousChars := []string{
		";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]",
		"<", ">", "\\", "!", "*", "?", "~", "'", "\"", "\n", "\r",
	}

	for _, char := range dangerousChars {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}
