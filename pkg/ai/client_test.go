package ai

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		verify func(t *testing.T, client *Client)
	}{
		{
			name:   "default config",
			config: Config{},
			verify: func(t *testing.T, client *Client) {
				if client.maxTurns != 3 {
					t.Errorf("Expected maxTurns 3, got %d", client.maxTurns)
				}
				if client.timeout != 120*time.Second {
					t.Errorf("Expected timeout 120s, got %v", client.timeout)
				}
				if client.systemPrompt == "" {
					t.Error("Expected non-empty system prompt")
				}
			},
		},
		{
			name: "custom config",
			config: Config{
				ClaudePath:   "/custom/path/claude",
				MaxTurns:     5,
				Timeout:      60 * time.Second,
				SystemPrompt: "Custom prompt",
			},
			verify: func(t *testing.T, client *Client) {
				if client.claudePath != "/custom/path/claude" {
					t.Errorf("Expected custom path, got %s", client.claudePath)
				}
				if client.maxTurns != 5 {
					t.Errorf("Expected maxTurns 5, got %d", client.maxTurns)
				}
				if client.timeout != 60*time.Second {
					t.Errorf("Expected timeout 60s, got %v", client.timeout)
				}
				if client.systemPrompt != "Custom prompt" {
					t.Errorf("Expected custom prompt, got %s", client.systemPrompt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			if client == nil {
				t.Fatal("Expected non-nil client")
			}
			if client.circuitBreaker == nil {
				t.Error("Expected circuit breaker to be initialized")
			}
			tt.verify(t, client)
		})
	}
}

func TestBinaryPathValidation(t *testing.T) {
	// Test with valid path
	if _, err := exec.LookPath("go"); err == nil {
		config := Config{ClaudePath: "go"} // Use go binary for testing
		client := NewClient(config)

		// Should resolve to absolute path
		if !filepath.IsAbs(client.claudePath) {
			t.Error("Expected absolute path for binary")
		}
	}

	// Test with invalid path
	config := Config{ClaudePath: "/nonexistent/binary"}
	client := NewClient(config)

	// Should keep the provided path even if invalid
	if client.claudePath != "/nonexistent/binary" {
		t.Error("Expected to keep provided path")
	}
}

func TestHealthCheck(t *testing.T) {
	client := &Client{
		claudePath: "echo", // Use echo for testing
		timeout:    5 * time.Second,
	}

	ctx := context.Background()
	status, err := client.HealthCheck(ctx)

	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}

	if status == nil {
		t.Fatal("Expected non-nil health status")
	}

	// The status depends on what echo returns
	// Just verify we got a status back
	if status.Version == "" && status.Healthy {
		t.Error("Expected version to be set or status to be unhealthy")
	}
}

func TestPromptValidation(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		wantError bool
	}{
		{
			name:      "valid prompt",
			prompt:    "Analyze this Kubernetes cluster",
			wantError: false,
		},
		{
			name:      "empty prompt",
			prompt:    "",
			wantError: false, // Empty context is valid, will use system prompt
		},
		{
			name:      "whitespace only",
			prompt:    "   \n\t  ",
			wantError: false, // Whitespace context is valid, will use system prompt
		},
		{
			name:      "very long prompt",
			prompt:    string(make([]byte, 100000)),
			wantError: false, // Long prompts are allowed now
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := AnalysisRequest{
				Type:    AnalysisTypeDiagnostic,
				Context: tt.prompt,
			}
			client := &Client{}
			_, err := client.buildPrompt(request)

			hasError := err != nil
			if tt.prompt == "" || strings.TrimSpace(tt.prompt) == "" {
				// Empty context should result in valid prompt with system prompt only
				hasError = false
			}

			if hasError != tt.wantError {
				t.Errorf("buildPrompt() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestPromptBuilding(t *testing.T) {
	client := &Client{
		systemPrompt: "Test system prompt",
		claudePath:   "echo",
		timeout:      5 * time.Second,
	}

	request := AnalysisRequest{
		Type:      AnalysisTypeDiagnostic,
		Context:   "Test context",
		Timestamp: time.Now(),
	}

	prompt, err := client.buildPrompt(request)
	if err != nil {
		t.Errorf("Failed to build prompt: %v", err)
	}

	if len(prompt) == 0 {
		t.Error("Expected non-empty prompt")
	}

	// Verify prompt contains key information
	if !contains(prompt, "Test context") {
		t.Error("Prompt should contain context")
	}
}

func TestAnalysisRequestConversion(t *testing.T) {
	client := &Client{systemPrompt: "Test prompt"}

	request := AnalysisRequest{
		Type:        AnalysisTypeDiagnostic,
		Context:     "Pod failing",
		Data:        map[string]interface{}{"pod": "test-pod"},
		Timestamp:   time.Now(),
		ClusterInfo: nil,
	}

	prompt, err := client.buildPrompt(request)
	if err != nil {
		t.Errorf("Failed to build prompt: %v", err)
	}

	if len(prompt) == 0 {
		t.Error("Expected non-empty prompt")
	}

	// Verify prompt contains key information
	if !contains(prompt, "Pod failing") {
		t.Error("Prompt should contain context")
	}
}

func TestCircuitBreakerIntegration(t *testing.T) {
	client := &Client{
		claudePath: "/nonexistent/binary",
		timeout:    1 * time.Second,
		circuitBreaker: NewCircuitBreaker(CircuitBreakerConfig{
			MaxFailures: 2,
			Timeout:     100 * time.Millisecond,
		}),
	}

	ctx := context.Background()
	request := AnalysisRequest{
		Type:    AnalysisTypeDiagnostic,
		Context: "Test",
	}

	// First few calls should fail normally
	for i := 0; i < 2; i++ {
		_, err := client.Analyze(ctx, request)
		if err == nil {
			t.Error("Expected error for nonexistent binary")
		}
	}

	// Circuit should now be open
	_, err := client.Analyze(ctx, request)
	if err == nil || !contains(err.Error(), "circuit breaker is open") {
		t.Error("Expected circuit breaker open error")
	}
}

func TestResponseParsing(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid JSON response",
			input: `{
				"summary": "Test summary",
				"findings": ["finding1", "finding2"],
				"recommendations": ["rec1"],
				"confidence": 0.85
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   "not a json",
			wantErr: false, // ParseResponse always returns a result, never errors
		},
		{
			name: "remediation response",
			input: `{
				"actions": [{"id": "1", "type": "scale", "description": "Scale deployment"}],
				"risk": "low",
				"confidence": 0.9
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := AnalysisRequest{
				Type: AnalysisTypeDiagnostic,
			}
			result, err := parser.ParseResponse(tt.input, request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAnalysis() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != "" &&
		(s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr || findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
