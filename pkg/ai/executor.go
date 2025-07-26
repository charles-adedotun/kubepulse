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

	// Parse and validate command to prevent injection
	args, err := k.parseAndValidateCommand(command)
	if err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	klog.V(2).Infof("Executing kubectl with args: %v", args)

	// Use exec.CommandContext with explicit args to prevent shell injection
	cmd := exec.CommandContext(ctx, "kubectl", args...) // #nosec G204 - args are validated and sanitized above
	output, err := cmd.CombinedOutput()

	if err != nil {
		return string(output), fmt.Errorf("command failed: %w, output: %s", err, output)
	}

	return string(output), nil
}

// parseAndValidateCommand parses a kubectl command string and validates arguments
func (k *KubectlExecutor) parseAndValidateCommand(command string) ([]string, error) {
	// Remove leading/trailing whitespace
	command = strings.TrimSpace(command)

	// Ensure command starts with kubectl
	if !strings.HasPrefix(command, "kubectl") {
		return nil, fmt.Errorf("only kubectl commands are supported")
	}

	// Split command into parts, handling quoted arguments safely
	parts := strings.Fields(command)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid kubectl command format")
	}

	// Remove "kubectl" from the parts (we'll add it back in Execute)
	args := parts[1:]

	// Validate each argument to prevent injection
	for _, arg := range args {
		if err := k.validateArgument(arg); err != nil {
			return nil, fmt.Errorf("invalid argument '%s': %w", arg, err)
		}
	}

	// Add namespace if not present and namespace is configured
	if k.namespace != "" && !k.hasNamespaceArg(args) {
		args = append([]string{"-n", k.namespace}, args...)
	}

	return args, nil
}

// validateArgument validates a single command argument
func (k *KubectlExecutor) validateArgument(arg string) error {
	// Check for shell injection patterns
	dangerousPatterns := []string{
		";", "&", "|", "$(", "`", ">", "<", "&&", "||", "\\",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(arg, pattern) {
			return fmt.Errorf("contains dangerous pattern: %s", pattern)
		}
	}

	// Ensure argument doesn't contain null bytes or control characters
	for _, char := range arg {
		if char < 32 && char != 9 && char != 10 && char != 13 { // Allow tab, LF, CR
			return fmt.Errorf("contains control character")
		}
	}

	return nil
}

// hasNamespaceArg checks if namespace is already specified in args
func (k *KubectlExecutor) hasNamespaceArg(args []string) bool {
	for _, arg := range args {
		if arg == "-n" || arg == "--namespace" {
			return true
		}
		if strings.HasPrefix(arg, "--namespace=") {
			return true
		}
		// Check for combined short flags like -ndefault
		if strings.HasPrefix(arg, "-n") && len(arg) > 2 {
			return true
		}
	}
	return false
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
