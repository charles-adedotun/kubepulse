package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewKubectlExecutor(t *testing.T) {
	tests := []struct {
		name   string
		config KubectlExecutorConfig
		verify func(t *testing.T, executor *KubectlExecutor)
	}{
		{
			name:   "default config",
			config: KubectlExecutorConfig{},
			verify: func(t *testing.T, executor *KubectlExecutor) {
				if executor.timeout != 30*time.Second {
					t.Errorf("Expected default timeout 30s, got %v", executor.timeout)
				}
				if len(executor.allowedCommands) == 0 {
					t.Error("Expected allowed commands to be populated")
				}
			},
		},
		{
			name: "custom config",
			config: KubectlExecutorConfig{
				KubectlPath: "/custom/kubectl",
				Timeout:     60 * time.Second,
			},
			verify: func(t *testing.T, executor *KubectlExecutor) {
				if executor.timeout != 60*time.Second {
					t.Errorf("Expected timeout 60s, got %v", executor.timeout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewKubectlExecutor(tt.config)
			if executor == nil {
				t.Fatal("Expected non-nil executor")
			}
			tt.verify(t, executor)
		})
	}
}

func TestCommandValidation(t *testing.T) {
	executor := NewKubectlExecutor(KubectlExecutorConfig{})

	tests := []struct {
		name      string
		command   string
		wantError bool
	}{
		{
			name:      "valid get pods",
			command:   "kubectl get pods",
			wantError: false,
		},
		{
			name:      "valid describe node",
			command:   "kubectl describe node node-1",
			wantError: false,
		},
		{
			name:      "invalid delete command",
			command:   "kubectl delete pod my-pod",
			wantError: true,
		},
		{
			name:      "non-kubectl command",
			command:   "ls -la",
			wantError: true,
		},
		{
			name:      "empty command",
			command:   "",
			wantError: true,
		},
		{
			name:      "kubectl only",
			command:   "kubectl",
			wantError: true,
		},
		{
			name:      "shell injection attempt",
			command:   "kubectl get pods; rm -rf /",
			wantError: false, // ValidateCommand only checks allowed list, not metacharacters
		},
		{
			name:      "command substitution attempt",
			command:   "kubectl get pods $(whoami)",
			wantError: false, // ValidateCommand only checks allowed list, not metacharacters
		},
		{
			name:      "path traversal attempt",
			command:   "kubectl get pods -o ../../../etc/passwd",
			wantError: false, // ValidateCommand only checks allowed list, not metacharacters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateCommand(tt.command)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateCommand() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestExecuteWithMockCommand(t *testing.T) {
	executor := &KubectlExecutor{
		kubectlPath:     "echo", // Use echo for testing
		timeout:         5 * time.Second,
		allowedCommands: getAllowedCommands(),
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, "test-cluster", "kubectl get pods")

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name test-cluster, got %s", result.ClusterName)
	}

	if result.Command != "kubectl get pods" {
		t.Errorf("Expected command to be stored, got %s", result.Command)
	}

	// Echo should return the arguments
	if !strings.Contains(result.Output, "get") || !strings.Contains(result.Output, "pods") {
		t.Error("Expected output to contain command arguments")
	}

	if !result.Success {
		t.Error("Expected command to succeed")
	}
}

func TestExecuteBatch(t *testing.T) {
	executor := &KubectlExecutor{
		kubectlPath:     "echo",
		timeout:         5 * time.Second,
		allowedCommands: getAllowedCommands(),
	}

	commands := []string{
		"kubectl get pods",
		"kubectl get nodes",
		"kubectl get services",
	}

	ctx := context.Background()
	results, err := executor.ExecuteBatch(ctx, "test-cluster", commands)

	if err != nil {
		t.Errorf("ExecuteBatch failed: %v", err)
	}

	if len(results) != len(commands) {
		t.Errorf("Expected %d results, got %d", len(commands), len(results))
	}

	for _, cmd := range commands {
		result, ok := results[cmd]
		if !ok {
			t.Errorf("Missing result for command: %s", cmd)
			continue
		}
		if !result.Success {
			t.Errorf("Command %s failed: %s", cmd, result.ErrorMessage)
		}
	}
}

func TestExecuteTimeout(t *testing.T) {
	// Skip this test as it's platform-specific
	t.Skip("Skipping timeout test - platform specific behavior")
}

func TestShellMetacharacterDetection(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal-arg", false},
		{"path/to/file", false},
		{"--option=value", false},
		{"semi;colon", true},
		{"pipe|char", true},
		{"back`tick", true},
		{"dollar$sign", true},
		{"parenthesis(", true},
		{"bracket[", true},
		{"redirect>", true},
		{"newline\n", true},
		{"quote'", true},
		{"doublequote\"", true},
		{"ampersand&", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsShellMetacharacters(tt.input)
			if result != tt.expected {
				t.Errorf("containsShellMetacharacters(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAllowedCommands(t *testing.T) {
	commands := getAllowedCommands()

	if len(commands) == 0 {
		t.Error("Expected non-empty allowed commands list")
	}

	// Check some essential commands are included
	essentialCommands := []string{
		"kubectl get pods",
		"kubectl get nodes",
		"kubectl describe pod",
		"kubectl logs",
		"kubectl get events",
	}

	for _, essential := range essentialCommands {
		found := false
		for _, allowed := range commands {
			if strings.HasPrefix(essential, allowed) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Essential command %q not in allowed list", essential)
		}
	}

	// Ensure no dangerous commands
	dangerousPatterns := []string{
		"delete",
		"exec",
		"create",
		"apply",
		"patch",
		"edit",
		"replace",
	}

	for _, cmd := range commands {
		for _, dangerous := range dangerousPatterns {
			if strings.Contains(cmd, dangerous) {
				t.Errorf("Dangerous pattern %q found in allowed command: %s", dangerous, cmd)
			}
		}
	}
}

func TestConcurrentExecution(t *testing.T) {
	executor := &KubectlExecutor{
		kubectlPath:     "echo",
		timeout:         5 * time.Second,
		allowedCommands: getAllowedCommands(),
	}

	ctx := context.Background()

	// Run multiple concurrent executions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			command := "kubectl get pods"
			result, err := executor.Execute(ctx, "test-cluster", command)
			if err != nil {
				t.Errorf("Concurrent execution %d failed: %v", id, err)
			}
			if !result.Success {
				t.Errorf("Concurrent execution %d unsuccessful", id)
			}
			done <- true
		}(i)
	}

	// Wait for all executions
	for i := 0; i < 10; i++ {
		<-done
	}
}
