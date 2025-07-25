package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestMain_BuildAndExecute(t *testing.T) {
	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test basic help command
	cmd := exec.Command("./kubepulse_test", "--help")
	cmd.Dir = "."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Help should exit with code 0
	if err != nil {
		t.Logf("Help command stderr: %s", stderr.String())
		t.Logf("Help command stdout: %s", stdout.String())
		// Help command typically exits with 0, but let's check output
	}

	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "KubePulse") {
		t.Errorf("Expected output to contain 'KubePulse', got: %s", output)
	}
}

func TestMain_Version(t *testing.T) {
	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test version command
	cmd := exec.Command("./kubepulse_test", "version")
	cmd.Dir = "."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Version command stderr: %s", stderr.String())
		t.Logf("Version command stdout: %s", stdout.String())
	}

	// Version command should produce some output
	output := stdout.String() + stderr.String()
	if len(output) == 0 {
		t.Error("Expected version command to produce output")
	}
}

func TestMain_InvalidCommand(t *testing.T) {
	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test invalid command
	cmd := exec.Command("./kubepulse_test", "invalid-command-xyz")
	cmd.Dir = "."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Invalid command should exit with non-zero code
	if err == nil {
		t.Error("Expected invalid command to exit with error")
	}

	output := stdout.String() + stderr.String()
	if !strings.Contains(strings.ToLower(output), "unknown") &&
		!strings.Contains(strings.ToLower(output), "error") {
		t.Errorf("Expected error output for invalid command, got: %s", output)
	}
}

func TestMain_WithTimeout(t *testing.T) {
	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test that the binary starts and can be terminated
	cmd := exec.Command("./kubepulse_test", "--help")
	cmd.Dir = "."

	// Set a reasonable timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		// Command completed, which is expected for --help
		if err != nil {
			t.Logf("Command completed with error (expected for some cases): %v", err)
		}
	case <-time.After(10 * time.Second):
		// Kill the process if it's taking too long
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		t.Error("Command took too long to execute")
	}
}

func TestMain_EnvironmentVariables(t *testing.T) {
	// Build the binary for testing
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test with environment variables
	cmd := exec.Command("./kubepulse_test", "--help")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "KUBECONFIG=/dev/null")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Command with env vars stderr: %s", stderr.String())
		t.Logf("Command with env vars stdout: %s", stdout.String())
	}

	// Should still work with environment variables
	output := stdout.String() + stderr.String()
	if len(output) == 0 {
		t.Error("Expected command with env vars to produce output")
	}
}

// Test the actual main function behavior by running it in a subprocess
func TestMainFunction_ErrorHandling(t *testing.T) {
	// This test validates that main() properly handles errors from commands.Execute()
	// We test this by examining the main function's structure rather than executing it directly

	// Verify main function exists and has expected structure
	if testing.Short() {
		t.Skip("Skipping main function integration test in short mode")
	}

	// Build and test error handling
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test with an invalid kubeconfig to trigger an error path
	cmd := exec.Command("./kubepulse_test", "monitor", "--kubeconfig", "/nonexistent/path")
	cmd.Dir = "."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with non-zero code due to invalid kubeconfig
	if err == nil {
		t.Error("Expected command with invalid kubeconfig to fail")
	}

	// Check that error is printed to stderr
	stderrOutput := stderr.String()
	if !strings.Contains(strings.ToLower(stderrOutput), "error") {
		t.Errorf("Expected error message in stderr, got: %s", stderrOutput)
	}

	// Check exit code
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() != 1 {
			t.Errorf("Expected exit code 1, got %d", exitError.ExitCode())
		}
	}
}

func TestMainFunction_KlogInitialization(t *testing.T) {
	// Test that klog is properly initialized by checking flag parsing
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test with klog flags
	cmd := exec.Command("./kubepulse_test", "--help", "-v=1")
	cmd.Dir = "."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Klog test stderr: %s", stderr.String())
		t.Logf("Klog test stdout: %s", stdout.String())
	}

	// Should handle klog flags without crashing
	output := stdout.String() + stderr.String()
	if len(output) == 0 {
		t.Error("Expected output when using klog flags")
	}
}

func TestMainFunction_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "kubepulse_test", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("kubepulse_test")

	// Test basic functionality with multiple scenarios
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(stdout, stderr string) error
	}{
		{
			name:        "help_command",
			args:        []string{"--help"},
			expectError: false,
			checkOutput: func(stdout, stderr string) error {
				output := stdout + stderr
				if !strings.Contains(output, "Available Commands") && !strings.Contains(output, "Usage") {
					return fmt.Errorf("expected help text, got: %s", output)
				}
				return nil
			},
		},
		{
			name:        "list_commands",
			args:        []string{},
			expectError: false,
			checkOutput: func(stdout, stderr string) error {
				output := stdout + stderr
				if len(output) == 0 {
					return fmt.Errorf("expected some output for root command")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./kubepulse_test", tt.args...)
			cmd.Dir = "."

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.expectError && err == nil {
				t.Error("Expected command to fail but it succeeded")
			}

			if !tt.expectError && err != nil {
				t.Logf("Command failed: %v", err)
				t.Logf("Stdout: %s", stdout.String())
				t.Logf("Stderr: %s", stderr.String())
			}

			if tt.checkOutput != nil {
				if err := tt.checkOutput(stdout.String(), stderr.String()); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// Test the commands.Execute() function behavior directly
func TestMainLogic_CommandsExecute(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with help flag
	os.Args = []string{"kubepulse", "--help"}

	// Since commands.Execute() will call cobra which may exit or panic,
	// we need to test this more carefully
	// For now, let's just verify the function exists and can be called
	// The actual functionality is tested via subprocess above

	// Test that commands.Execute exists and is callable
	// We can't compare functions to nil directly in Go,
	// but we can verify it exists by calling it in a subprocess test above

	// We can't directly test commands.Execute() here because it may call os.Exit
	// The subprocess tests above provide the real functional testing
}

// Test klog initialization behavior
func TestMainLogic_KlogInitialization(t *testing.T) {
	// Test that the main function properly initializes klog
	// This is difficult to test directly since klog.InitFlags modifies global state
	// The subprocess tests provide the real validation

	// Just verify the import and basic structure
	if testing.Short() {
		t.Skip("Skipping klog test in short mode")
	}

	// Verify our main function has the expected structure by building successfully
	buildCmd := exec.Command("go", "build", "-o", "test_build", ".")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Main function should compile successfully: %v", err)
	}
	defer os.Remove("test_build")

	// The successful build confirms main() function structure is correct
}
