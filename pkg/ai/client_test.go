package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		verify func(*Client) error
	}{
		{
			name:   "default config",
			config: Config{TestMode: true},
			verify: func(c *Client) error {
				if c.claudePath != "claude" {
					t.Errorf("expected default claudePath 'claude', got %s", c.claudePath)
				}
				if c.maxTurns != 3 {
					t.Errorf("expected default maxTurns 3, got %d", c.maxTurns)
				}
				if c.timeout != 120*time.Second {
					t.Errorf("expected default timeout 120s, got %v", c.timeout)
				}
				if c.systemPrompt == "" {
					t.Error("expected default system prompt to be set")
				}
				if c.circuitBreaker == nil {
					t.Error("expected circuit breaker to be initialized")
				}
				if c.parser == nil {
					t.Error("expected response parser to be initialized")
				}
				return nil
			},
		},
		{
			name: "custom config",
			config: Config{
				ClaudePath:   "/custom/path/claude",
				MaxTurns:     5,
				Timeout:      60 * time.Second,
				SystemPrompt: "Custom prompt",
				TestMode:     true,
			},
			verify: func(c *Client) error {
				if c.claudePath != "/custom/path/claude" {
					t.Errorf("expected claudePath '/custom/path/claude', got %s", c.claudePath)
				}
				if c.maxTurns != 5 {
					t.Errorf("expected maxTurns 5, got %d", c.maxTurns)
				}
				if c.timeout != 60*time.Second {
					t.Errorf("expected timeout 60s, got %v", c.timeout)
				}
				if c.systemPrompt != "Custom prompt" {
					t.Errorf("expected systemPrompt 'Custom prompt', got %s", c.systemPrompt)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			if err := tt.verify(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateClaudePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		expectErr bool
	}{
		{
			name:      "claude in PATH",
			path:      "claude",
			expectErr: false,
		},
		{
			name:      "usr local bin",
			path:      "/usr/local/bin/claude",
			expectErr: false,
		},
		{
			name:      "homebrew path",
			path:      "/opt/homebrew/bin/claude",
			expectErr: false,
		},
		{
			name:      "custom absolute path",
			path:      "/some/path/to/claude",
			expectErr: false,
		},
		{
			name:      "invalid path",
			path:      "/malicious/script",
			expectErr: true,
		},
		{
			name:      "relative path",
			path:      "../claude",
			expectErr: true,
		},
		{
			name:      "empty path",
			path:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{claudePath: tt.path}
			err := client.validateClaudePath()

			if tt.expectErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestSanitizePrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean prompt",
			input:    "What is the status of my cluster?",
			expected: "What is the status of my cluster?",
		},
		{
			name:     "with null bytes",
			input:    "Test\x00prompt",
			expected: "Testprompt",
		},
		{
			name:     "with escape sequences",
			input:    "Test\x1bprompt",
			expected: "Testprompt",
		},
		{
			name:     "with command injection",
			input:    "Show pods $(rm -rf /)",
			expected: "Show pods rm -rf /)",
		},
		{
			name:     "with backticks",
			input:    "Show pods `whoami`",
			expected: "Show pods whoami",
		},
		{
			name:     "with pipes and redirects",
			input:    "kubectl get pods | grep failed > /tmp/output",
			expected: "kubectl get pods  grep failed  /tmp/output",
		},
		{
			name:     "with semicolons and ampersands",
			input:    "kubectl get pods; rm file && echo done",
			expected: "kubectl get pods rm file  echo done",
		},
		{
			name:     "with OR operators",
			input:    "cmd1 || cmd2",
			expected: "cmd1  cmd2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			result := client.sanitizePrompt(tt.input)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name     string
		request  AnalysisRequest
		contains []string
	}{
		{
			name: "diagnostic request",
			request: AnalysisRequest{
				Type:    AnalysisTypeDiagnostic,
				Context: "Pod failure analysis",
				HealthCheck: &CheckResult{
					Name:    "pod-health",
					Status:  HealthStatusUnhealthy,
					Message: "Pod is crashing",
				},
			},
			contains: []string{
				"ANALYSIS REQUEST: diagnostic",
				"CONTEXT: Pod failure analysis",
				"HEALTH CHECK DATA:",
				"pod-health",
				"crashing",
				"DIAGNOSTIC ANALYSIS INSTRUCTIONS:",
			},
		},
		{
			name: "healing request",
			request: AnalysisRequest{
				Type:    AnalysisTypeHealing,
				Context: "Auto-remediation",
				Data: map[string]interface{}{
					"priority": "high",
				},
			},
			contains: []string{
				"ANALYSIS REQUEST: healing",
				"CONTEXT: Auto-remediation",
				"ADDITIONAL CONTEXT:",
				"priority",
				"HEALING ANALYSIS INSTRUCTIONS:",
			},
		},
		{
			name: "cluster summary request",
			request: AnalysisRequest{
				Type:    AnalysisTypeSummary,
				Context: "Overall health assessment",
				ClusterInfo: &ClusterHealth{
					ClusterName: "test-cluster",
					Status:      HealthStatusHealthy,
					Score: HealthScore{
						Weighted: 85.5,
					},
				},
			},
			contains: []string{
				"ANALYSIS REQUEST: summary",
				"CONTEXT: Overall health assessment",
				"CLUSTER HEALTH DATA:",
				"test-cluster",
				"CLUSTER SUMMARY INSTRUCTIONS:",
			},
		},
		{
			name: "root cause request",
			request: AnalysisRequest{
				Type:    AnalysisTypeRootCause,
				Context: "Deep analysis needed",
			},
			contains: []string{
				"ANALYSIS REQUEST: root_cause",
				"ROOT CAUSE ANALYSIS INSTRUCTIONS:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}
			prompt, err := client.buildPrompt(tt.request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(prompt, expected) {
					t.Errorf("expected prompt to contain %q, but it didn't. Prompt: %s", expected, prompt)
				}
			}
		})
	}
}

func TestAnalyzeDiagnostic(t *testing.T) {
	client := NewClient(Config{TestMode: true})

	checkResult := &CheckResult{
		Name:    "test-check",
		Status:  HealthStatusUnhealthy,
		Message: "Test failure",
	}

	diagnosticContext := DiagnosticContext{
		ClusterName:  "test-cluster",
		ResourceType: "pod",
		ResourceName: "test-pod",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// In test mode, this should return a successful mock response
	response, err := client.AnalyzeDiagnostic(ctx, checkResult, diagnosticContext)

	// We expect success in test mode with mock responses
	if err != nil {
		t.Errorf("unexpected error in test mode: %v", err)
		return
	}

	if response == nil {
		t.Error("expected response in test mode, got nil")
		return
	}

	// Validate mock response structure
	if response.Summary == "" {
		t.Error("expected non-empty summary in mock response")
	}

	if response.Confidence <= 0 {
		t.Errorf("expected positive confidence in mock response, got %f", response.Confidence)
	}

	if response.Severity == "" {
		t.Error("expected non-empty severity in mock response")
	}
}

func TestAnalyzeHealing(t *testing.T) {
	client := NewClient(Config{TestMode: true})

	checkResult := &CheckResult{
		Name:    "test-check",
		Status:  HealthStatusUnhealthy,
		Message: "Test failure",
	}

	diagnosticContext := DiagnosticContext{
		ClusterName:  "test-cluster",
		ResourceType: "pod",
		ResourceName: "test-pod",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// In test mode, this should return a successful mock response
	response, err := client.AnalyzeHealing(ctx, checkResult, diagnosticContext)

	// We expect success in test mode with mock responses
	if err != nil {
		t.Errorf("unexpected error in test mode: %v", err)
		return
	}

	if response == nil {
		t.Error("expected response in test mode, got nil")
		return
	}

	// Validate mock response structure
	if response.Summary == "" {
		t.Error("expected non-empty summary in mock response")
	}

	if response.Confidence <= 0 {
		t.Errorf("expected positive confidence in mock response, got %f", response.Confidence)
	}

	if response.Severity == "" {
		t.Error("expected non-empty severity in mock response")
	}
}

func TestAnalyzeCluster(t *testing.T) {
	client := NewClient(Config{TestMode: true})

	clusterHealth := &ClusterHealth{
		ClusterName: "test-cluster",
		Status:      HealthStatusHealthy,
		Score: HealthScore{
			Weighted: 95.0,
		},
		Checks: []CheckResult{
			{
				Name:   "pod-health",
				Status: HealthStatusHealthy,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// In test mode, this should return a successful mock response
	response, err := client.AnalyzeCluster(ctx, clusterHealth)

	// We expect success in test mode with mock responses
	if err != nil {
		t.Errorf("unexpected error in test mode: %v", err)
		return
	}

	if response == nil {
		t.Error("expected response in test mode, got nil")
		return
	}

	// Validate mock response structure for InsightSummary
	if response.OverallHealth == "" {
		t.Error("expected non-empty overall health in mock response")
	}

	if response.AIConfidence <= 0 {
		t.Errorf("expected positive AI confidence in mock response, got %f", response.AIConfidence)
	}

	if response.HealthScore <= 0 {
		t.Errorf("expected positive health score in mock response, got %f", response.HealthScore)
	}
}

func TestCountCriticalIssues(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		health   *ClusterHealth
		expected int
	}{
		{
			name: "no critical issues",
			health: &ClusterHealth{
				Checks: []CheckResult{
					{Status: HealthStatusHealthy},
					{Status: HealthStatusDegraded},
				},
			},
			expected: 0,
		},
		{
			name: "some critical issues",
			health: &ClusterHealth{
				Checks: []CheckResult{
					{Status: HealthStatusHealthy},
					{Status: HealthStatusUnhealthy},
					{Status: HealthStatusUnhealthy},
					{Status: HealthStatusDegraded},
				},
			},
			expected: 2,
		},
		{
			name: "all critical issues",
			health: &ClusterHealth{
				Checks: []CheckResult{
					{Status: HealthStatusUnhealthy},
					{Status: HealthStatusUnhealthy},
				},
			},
			expected: 2,
		},
		{
			name: "empty checks",
			health: &ClusterHealth{
				Checks: []CheckResult{},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.countCriticalIssues(tt.health)
			if result != tt.expected {
				t.Errorf("expected %d critical issues, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetCircuitBreakerStats(t *testing.T) {
	client := NewClient(Config{TestMode: true})

	stats := client.GetCircuitBreakerStats()

	if stats == nil {
		t.Error("expected circuit breaker stats, got nil")
	}

	// Verify it returns a map
	if _, ok := stats["state"]; !ok {
		t.Error("expected stats to contain 'state' key")
	}
}

func TestResetCircuitBreaker(t *testing.T) {
	client := NewClient(Config{TestMode: true})

	// This should not panic
	client.ResetCircuitBreaker()
}

func TestGetDefaultSystemPrompt(t *testing.T) {
	prompt := getDefaultSystemPrompt()

	if prompt == "" {
		t.Error("expected non-empty system prompt")
	}

	expectedContains := []string{
		"Kubernetes",
		"DevOps",
		"troubleshooting",
		"remediation",
		"JSON format",
		"kubectl",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(prompt, expected) {
			t.Errorf("expected system prompt to contain %q", expected)
		}
	}
}

func TestInstructionTemplates(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		contains []string
	}{
		{
			name: "diagnostic instructions",
			fn:   getDiagnosticInstructions,
			contains: []string{
				"DIAGNOSTIC ANALYSIS",
				"root cause",
				"SUMMARY",
				"DIAGNOSIS",
				"CONFIDENCE",
			},
		},
		{
			name: "healing instructions",
			fn:   getHealingInstructions,
			contains: []string{
				"HEALING ANALYSIS",
				"remediation",
				"kubectl",
				"IMMEDIATE_ACTIONS",
				"AUTOMATED_ACTIONS",
			},
		},
		{
			name: "summary instructions",
			fn:   getSummaryInstructions,
			contains: []string{
				"CLUSTER SUMMARY",
				"health assessment",
				"CRITICAL_ISSUES",
				"TRENDS",
				"RECOMMENDATIONS",
			},
		},
		{
			name: "root cause instructions",
			fn:   getRootCauseInstructions,
			contains: []string{
				"ROOT CAUSE ANALYSIS",
				"fundamental cause",
				"ANALYSIS",
				"EVIDENCE",
				"SYSTEMIC_FIXES",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instructions := tt.fn()

			if instructions == "" {
				t.Error("expected non-empty instructions")
			}

			for _, expected := range tt.contains {
				if !strings.Contains(instructions, expected) {
					t.Errorf("expected instructions to contain %q", expected)
				}
			}
		})
	}
}

func TestAnalysisRequestValidation(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		request AnalysisRequest
		wantErr bool
	}{
		{
			name: "valid diagnostic request",
			request: AnalysisRequest{
				Type:    AnalysisTypeDiagnostic,
				Context: "Test context",
				HealthCheck: &CheckResult{
					Name: "test-check",
				},
			},
			wantErr: false,
		},
		{
			name: "valid cluster summary request",
			request: AnalysisRequest{
				Type:    AnalysisTypeSummary,
				Context: "Test context",
				ClusterInfo: &ClusterHealth{
					ClusterName: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with data",
			request: AnalysisRequest{
				Type:    AnalysisTypeHealing,
				Context: "Test context",
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := client.buildPrompt(tt.request)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if !tt.wantErr && prompt == "" {
				t.Error("expected non-empty prompt")
			}
		})
	}
}
