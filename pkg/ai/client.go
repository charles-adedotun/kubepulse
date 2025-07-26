package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// Client provides AI analysis capabilities using Claude Code CLI
type Client struct {
	claudePath     string
	maxTurns       int
	timeout        time.Duration
	systemPrompt   string
	testMode       bool
	circuitBreaker *CircuitBreaker
	parser         *ResponseParser
}

// Config holds configuration for the AI client
type Config struct {
	ClaudePath   string
	MaxTurns     int
	Timeout      time.Duration
	SystemPrompt string
	TestMode     bool // When true, returns mock responses instead of executing Claude CLI
}

// NewClient creates a new AI client
func NewClient(config Config) *Client {
	if config.ClaudePath == "" {
		config.ClaudePath = "claude" // Assume claude is in PATH
	}
	if config.MaxTurns == 0 {
		config.MaxTurns = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 120 * time.Second
	}
	if config.SystemPrompt == "" {
		config.SystemPrompt = getDefaultSystemPrompt()
	}

	// Initialize circuit breaker for AI calls
	circuitBreaker := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  3,                // Allow 3 failures before opening
		Timeout:      5 * time.Minute,  // Wait 5 minutes before retry
		ResetTimeout: 30 * time.Second, // Stay in half-open for 30 seconds
		OnStateChange: func(from, to CircuitState) {
			klog.Infof("AI Circuit breaker state changed: %s -> %s", from, to)
		},
	})

	return &Client{
		claudePath:     config.ClaudePath,
		maxTurns:       config.MaxTurns,
		timeout:        config.Timeout,
		systemPrompt:   config.SystemPrompt,
		testMode:       config.TestMode,
		circuitBreaker: circuitBreaker,
		parser:         NewResponseParser(),
	}
}

// Analyze performs AI analysis on the given request
func (c *Client) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	start := time.Now()

	prompt, err := c.buildPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	klog.V(2).Infof("AI Analysis: Running Claude Code CLI analysis for type=%s", request.Type)
	klog.V(3).Infof("AI Analysis prompt preview (first 200 chars): %s", func() string {
		if len(prompt) > 200 {
			return prompt[:200] + "..."
		}
		return prompt
	}())

	var result string
	err = c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		result, execErr = c.runClaude(ctx, prompt)
		return execErr
	})

	if err != nil {
		return nil, fmt.Errorf("claude analysis failed: %w", err)
	}

	response, err := c.parser.ParseResponse(result, request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	response.ID = fmt.Sprintf("%s-%d", request.Type, time.Now().Unix())
	response.Type = request.Type
	response.Timestamp = time.Now()
	response.Duration = time.Since(start)

	klog.V(2).Infof("AI Analysis completed: type=%s, confidence=%.2f, duration=%v",
		request.Type, response.Confidence, response.Duration)

	return response, nil
}

// AnalyzeDiagnostic performs diagnostic analysis on health check failures
func (c *Client) AnalyzeDiagnostic(ctx context.Context, checkResult *CheckResult, context DiagnosticContext) (*AnalysisResponse, error) {
	request := AnalysisRequest{
		Type:        AnalysisTypeDiagnostic,
		Context:     "Kubernetes health check failure requiring diagnostic analysis",
		HealthCheck: checkResult,
		Data: map[string]interface{}{
			"diagnostic_context": context,
		},
		Timestamp: time.Now(),
	}

	return c.Analyze(ctx, request)
}

// AnalyzeHealing suggests self-healing actions
func (c *Client) AnalyzeHealing(ctx context.Context, checkResult *CheckResult, context DiagnosticContext) (*AnalysisResponse, error) {
	request := AnalysisRequest{
		Type:        AnalysisTypeHealing,
		Context:     "Generate automated healing suggestions for Kubernetes issues",
		HealthCheck: checkResult,
		Data: map[string]interface{}{
			"diagnostic_context": context,
		},
		Timestamp: time.Now(),
	}

	return c.Analyze(ctx, request)
}

// AnalyzeCluster performs comprehensive cluster analysis
func (c *Client) AnalyzeCluster(ctx context.Context, clusterHealth *ClusterHealth) (*InsightSummary, error) {
	request := AnalysisRequest{
		Type:        AnalysisTypeSummary,
		Context:     "Comprehensive Kubernetes cluster health analysis and insights",
		ClusterInfo: clusterHealth,
		Timestamp:   time.Now(),
	}

	// Add timeout to prevent hanging (increased to 60s for AI analysis)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	response, err := c.Analyze(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Convert to InsightSummary
	summary := &InsightSummary{
		OverallHealth:   response.Summary,
		CriticalIssues:  c.countCriticalIssues(clusterHealth),
		Recommendations: response.Recommendations,
		TrendAnalysis:   response.Diagnosis,
		HealthScore:     clusterHealth.Score.Weighted,
		AIConfidence:    response.Confidence,
		LastAnalyzed:    response.Timestamp,
		Context:         response.Context,
	}

	return summary, nil
}

// runClaude executes the Claude Code CLI with the given prompt
func (c *Client) runClaude(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Return mock response in test mode
	if c.testMode {
		klog.Infof("AI: Test mode enabled, returning mock response")
		return `{
			"analysis": "Mock analysis response for testing",
			"confidence": 0.8,
			"severity": "medium",
			"recommendations": ["Mock recommendation 1", "Mock recommendation 2"],
			"actions": ["Mock action 1"],
			"commands": [],
			"references": [],
			"followup_questions": []
		}`, nil
	}

	// Validate Claude path for security
	if err := c.validateClaudePath(); err != nil {
		return "", fmt.Errorf("invalid claude path: %w", err)
	}

	// Sanitize prompt to prevent injection attacks
	sanitizedPrompt := c.sanitizePrompt(prompt)
	if len(sanitizedPrompt) > 100000 { // Reasonable limit
		return "", fmt.Errorf("prompt too long: %d characters", len(sanitizedPrompt))
	}

	args := []string{
		"-p", sanitizedPrompt,
		"--max-turns", "1",
		"--system-prompt", c.systemPrompt,
		"--permission-mode", "bypassPermissions",
	}

	// Use shell execution to inherit proper environment (like kubectl executor)
	// This ensures Node.js/NVM environment is available for Claude CLI
	escapedArgs := make([]string, len(args))
	for i, arg := range args {
		escapedArgs[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(arg, "'", "'\"'\"'"))
	}
	cmdLine := fmt.Sprintf("%s %s", c.claudePath, strings.Join(escapedArgs, " "))
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdLine)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Ensure proper environment inheritance for Node.js/Claude CLI
	cmd.Env = os.Environ() // Inherit full environment from parent process

	klog.Infof("AI: Executing Claude CLI analysis (prompt length: %d chars)", len(sanitizedPrompt))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			klog.Errorf("Claude CLI timed out after %v", c.timeout)
			return "", fmt.Errorf("claude CLI timed out after %v", c.timeout)
		}
		klog.Errorf("Claude command failed: %v, stderr: %s, stdout: %s", err, stderr.String(), stdout.String())
		return "", fmt.Errorf("claude command failed: %w, stderr: %s", err, stderr.String())
	}

	klog.Infof("AI: Claude CLI completed successfully (output length: %d chars)", len(stdout.String()))

	return stdout.String(), nil
}

// validateClaudePath ensures the Claude CLI path is safe to execute
func (c *Client) validateClaudePath() error {
	// Only allow known safe paths for Claude CLI
	allowedPaths := []string{
		"claude", // From PATH
		"/usr/local/bin/claude",
		"/opt/homebrew/bin/claude",
	}

	for _, allowed := range allowedPaths {
		if c.claudePath == allowed {
			return nil
		}
	}

	// Check if it's an absolute path pointing to a 'claude' binary
	if strings.HasPrefix(c.claudePath, "/") && strings.HasSuffix(c.claudePath, "/claude") {
		return nil
	}

	return fmt.Errorf("claude path not in allowlist: %s", c.claudePath)
}

// sanitizePrompt removes potentially dangerous content from prompts
func (c *Client) sanitizePrompt(prompt string) string {
	// Remove shell escape sequences and control characters
	sanitized := strings.ReplaceAll(prompt, "\x00", "")
	sanitized = strings.ReplaceAll(sanitized, "\x1b", "")

	// Remove potential command injection patterns
	dangerous := []string{
		"$(", "`", ";", "&", "|", ">", "<", "&&", "||",
	}

	for _, pattern := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, pattern, "")
	}

	return sanitized
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (c *Client) GetCircuitBreakerStats() map[string]interface{} {
	return c.circuitBreaker.GetStats()
}

// ResetCircuitBreaker manually resets the circuit breaker
func (c *Client) ResetCircuitBreaker() {
	c.circuitBreaker.Reset()
}

// buildPrompt constructs the AI prompt based on the analysis request
func (c *Client) buildPrompt(request AnalysisRequest) (string, error) {
	var prompt strings.Builder

	// Add context
	prompt.WriteString(fmt.Sprintf("ANALYSIS REQUEST: %s\n\n", request.Type))
	prompt.WriteString(fmt.Sprintf("CONTEXT: %s\n\n", request.Context))

	// Add health check data if available
	if request.HealthCheck != nil {
		healthData, _ := json.MarshalIndent(request.HealthCheck, "", "  ")
		prompt.WriteString("HEALTH CHECK DATA:\n")
		prompt.WriteString(string(healthData))
		prompt.WriteString("\n\n")
	}

	// Add cluster info if available
	if request.ClusterInfo != nil {
		clusterData, _ := json.MarshalIndent(request.ClusterInfo, "", "  ")
		prompt.WriteString("CLUSTER HEALTH DATA:\n")
		prompt.WriteString(string(clusterData))
		prompt.WriteString("\n\n")
	}

	// Add additional data
	if len(request.Data) > 0 {
		additionalData, _ := json.MarshalIndent(request.Data, "", "  ")
		prompt.WriteString("ADDITIONAL CONTEXT:\n")
		prompt.WriteString(string(additionalData))
		prompt.WriteString("\n\n")
	}

	// Add specific instructions based on analysis type
	switch request.Type {
	case AnalysisTypeDiagnostic:
		prompt.WriteString(getDiagnosticInstructions())
	case AnalysisTypeHealing:
		prompt.WriteString(getHealingInstructions())
	case AnalysisTypeSummary:
		prompt.WriteString(getSummaryInstructions())
	case AnalysisTypeRootCause:
		prompt.WriteString(getRootCauseInstructions())
	}

	return prompt.String(), nil
}

// parseResponse is deprecated - replaced by ResponseParser

// countCriticalIssues counts critical issues in cluster health
func (c *Client) countCriticalIssues(clusterHealth *ClusterHealth) int {
	count := 0
	for _, check := range clusterHealth.Checks {
		if check.Status == HealthStatusUnhealthy {
			count++
		}
	}
	return count
}

// getDefaultSystemPrompt returns the default system prompt for AI analysis
func getDefaultSystemPrompt() string {
	return `You are an expert Kubernetes Site Reliability Engineer and DevOps specialist with deep knowledge of:
- Kubernetes cluster operations and troubleshooting
- Container orchestration and microservices
- Infrastructure monitoring and observability
- Incident response and root cause analysis
- Automated remediation strategies

Your role is to analyze the provided Kubernetes health data (JSON format) and provide:
1. Clear, actionable diagnostic insights based on the data provided
2. Specific remediation recommendations
3. Preventive measures for future issues
4. Automated healing suggestions when appropriate

IMPORTANT: You will be provided with complete Kubernetes cluster data in JSON format. Analyze ONLY this provided data. Do not attempt to run any commands or access external systems.

Always provide responses in a structured format with:
- Clear summary of the situation
- Confidence level in your analysis
- Severity assessment
- Prioritized recommendations
- Suggested kubectl commands (for manual execution by operators)
- References to Kubernetes best practices

Focus on practical, implementable solutions that minimize downtime and prevent future incidents.`
}

// Legacy extraction functions - replaced by ResponseParser
// These functions are deprecated and will be removed in future versions

// Instruction templates for different analysis types
func getDiagnosticInstructions() string {
	return `
DIAGNOSTIC ANALYSIS INSTRUCTIONS:
1. Analyze the health check failure and identify the root cause
2. Examine error messages, logs, and metrics
3. Consider common Kubernetes issues (resource constraints, networking, configuration)
4. Provide a clear diagnosis with confidence level
5. Include specific technical details and evidence
6. Suggest investigation commands if more data is needed

Please structure your response with:
- SUMMARY: Brief overview of the issue
- DIAGNOSIS: Detailed technical analysis
- CONFIDENCE: Your confidence level in this diagnosis
- EVIDENCE: Supporting data from the health check
- NEXT_STEPS: Immediate actions to take
`
}

func getHealingInstructions() string {
	return `
HEALING ANALYSIS INSTRUCTIONS:
1. Based on the diagnostic data, suggest automated remediation actions
2. Prioritize actions by safety and effectiveness
3. Include both immediate fixes and preventive measures
4. Specify which actions can be automated safely
5. Provide exact kubectl commands where applicable
6. Consider rollback procedures for risky operations

Please structure your response with:
- SUMMARY: Overview of suggested healing approach
- IMMEDIATE_ACTIONS: Critical fixes to apply now
- AUTOMATED_ACTIONS: Safe actions for automation
- MANUAL_ACTIONS: Actions requiring human approval
- PREVENTION: Steps to prevent recurrence
- COMMANDS: Specific kubectl/script commands
`
}

func getSummaryInstructions() string {
	return `
CLUSTER SUMMARY INSTRUCTIONS:
1. Provide a comprehensive assessment of overall cluster health
2. Identify trends and patterns across all health checks
3. Highlight critical issues requiring immediate attention
4. Suggest optimization opportunities
5. Predict potential future issues based on current data
6. Provide actionable insights for cluster improvement

Please structure your response with:
- SUMMARY: Overall cluster health assessment
- CRITICAL_ISSUES: High-priority problems
- TRENDS: Patterns and trends observed
- PREDICTIONS: Potential future issues
- RECOMMENDATIONS: Top improvement suggestions
- OPTIMIZATION: Performance and reliability improvements
`
}

func getRootCauseInstructions() string {
	return `
ROOT CAUSE ANALYSIS INSTRUCTIONS:
1. Perform deep analysis to identify the fundamental cause
2. Trace the chain of events leading to the issue
3. Eliminate symptoms to focus on core problems
4. Consider system interactions and dependencies
5. Provide evidence-based conclusions
6. Suggest systemic fixes to prevent similar issues

Please structure your response with:
- SUMMARY: Root cause identification
- ANALYSIS: Step-by-step causation chain
- EVIDENCE: Supporting data and reasoning
- IMPACT: Scope and consequences of the root cause
- SYSTEMIC_FIXES: Fundamental solutions
- VALIDATION: How to verify the fix worked
`
}
