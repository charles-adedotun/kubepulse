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
	claudePath    string
	maxTurns      int
	timeout       time.Duration
	systemPrompt  string
}

// Config holds configuration for the AI client
type Config struct {
	ClaudePath   string
	MaxTurns     int
	Timeout      time.Duration
	SystemPrompt string
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
		config.Timeout = 60 * time.Second
	}
	if config.SystemPrompt == "" {
		config.SystemPrompt = getDefaultSystemPrompt()
	}

	return &Client{
		claudePath:   config.ClaudePath,
		maxTurns:     config.MaxTurns,
		timeout:      config.Timeout,
		systemPrompt: config.SystemPrompt,
	}
}

// Analyze performs AI analysis on the given request
func (c *Client) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	start := time.Now()
	
	// Check for mock mode
	if os.Getenv("KUBEPULSE_AI_MOCK") == "true" {
		return c.getMockAnalysisResponse(request), nil
	}
	
	prompt, err := c.buildPrompt(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	klog.V(2).Infof("AI Analysis: Running Claude Code CLI analysis for type=%s", request.Type)

	result, err := c.runClaude(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("claude analysis failed: %w", err)
	}

	response, err := c.parseResponse(result, request)
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
	// For testing/demo: return mock data if KUBEPULSE_AI_MOCK is set or if Claude is not available
	if os.Getenv("KUBEPULSE_AI_MOCK") == "true" {
		return c.getMockInsights(clusterHealth), nil
	}

	request := AnalysisRequest{
		Type:        AnalysisTypeSummary,
		Context:     "Comprehensive Kubernetes cluster health analysis and insights",
		ClusterInfo: clusterHealth,
		Timestamp:   time.Now(),
	}

	// Add timeout to prevent hanging
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	response, err := c.Analyze(ctx, request)
	if err != nil {
		klog.Warningf("AI analysis failed, using mock data: %v", err)
		return c.getMockInsights(clusterHealth), nil
	}

	// Convert to InsightSummary
	summary := &InsightSummary{
		OverallHealth:    response.Summary,
		CriticalIssues:   c.countCriticalIssues(clusterHealth),
		Recommendations:  response.Recommendations,
		TrendAnalysis:    response.Diagnosis,
		HealthScore:      clusterHealth.Score.Weighted,
		AIConfidence:     response.Confidence,
		LastAnalyzed:     response.Timestamp,
		Context:          response.Context,
	}

	return summary, nil
}

// runClaude executes the Claude Code CLI with the given prompt
func (c *Client) runClaude(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--max-turns", fmt.Sprintf("%d", c.maxTurns),
		"--system-prompt", c.systemPrompt,
	}

	cmd := exec.CommandContext(ctx, c.claudePath, args...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	klog.V(3).Infof("Executing Claude CLI: %s %s", c.claudePath, strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
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

// parseResponse parses the Claude CLI JSON response
func (c *Client) parseResponse(result string, request AnalysisRequest) (*AnalysisResponse, error) {
	var claudeResult struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal([]byte(result), &claudeResult); err != nil {
		return nil, fmt.Errorf("failed to parse claude response: %w", err)
	}

	// Try to parse structured response from Claude's result
	response := &AnalysisResponse{
		Summary:         extractSummary(claudeResult.Result),
		Diagnosis:       extractDiagnosis(claudeResult.Result),
		Confidence:      extractConfidence(claudeResult.Result),
		Severity:        extractSeverity(claudeResult.Result),
		Recommendations: extractRecommendations(claudeResult.Result),
		Actions:         extractActions(claudeResult.Result),
		Context:         make(map[string]interface{}),
	}

	// Set defaults if extraction failed
	if response.Confidence == 0 {
		response.Confidence = 0.8
	}
	if response.Severity == "" {
		response.Severity = SeverityMedium
	}

	return response, nil
}

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

Your role is to analyze Kubernetes health data and provide:
1. Clear, actionable diagnostic insights
2. Specific remediation recommendations
3. Preventive measures for future issues
4. Automated healing suggestions when appropriate

Always provide responses in a structured format with:
- Clear summary of the situation
- Confidence level in your analysis
- Severity assessment
- Prioritized recommendations
- Specific kubectl commands or scripts when applicable
- References to Kubernetes best practices

Focus on practical, implementable solutions that minimize downtime and prevent future incidents.`
}

// Helper functions to extract information from Claude's response
func extractSummary(text string) string {
	// Simple extraction - in production, use more sophisticated parsing
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "summary") && len(line) > 20 {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(strings.Join(parts[1:], ":"))
			}
		}
	}
	// Return first meaningful line if no summary found
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 20 && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return "AI analysis completed"
}

func extractDiagnosis(text string) string {
	// Extract diagnosis section
	if strings.Contains(text, "DIAGNOSIS:") {
		parts := strings.Split(text, "DIAGNOSIS:")
		if len(parts) > 1 {
			diagnosis := strings.Split(parts[1], "\n\n")[0]
			return strings.TrimSpace(diagnosis)
		}
	}
	return text // Return full text if no specific diagnosis section
}

func extractConfidence(text string) float64 {
	// Look for confidence indicators
	text = strings.ToLower(text)
	if strings.Contains(text, "high confidence") || strings.Contains(text, "very confident") {
		return 0.9
	}
	if strings.Contains(text, "confident") {
		return 0.8
	}
	if strings.Contains(text, "likely") {
		return 0.7
	}
	if strings.Contains(text, "possible") || strings.Contains(text, "might") {
		return 0.6
	}
	return 0.75 // Default confidence
}

func extractSeverity(text string) SeverityLevel {
	text = strings.ToLower(text)
	if strings.Contains(text, "critical") || strings.Contains(text, "urgent") {
		return SeverityCritical
	}
	if strings.Contains(text, "high priority") || strings.Contains(text, "important") {
		return SeverityHigh
	}
	if strings.Contains(text, "medium") || strings.Contains(text, "moderate") {
		return SeverityMedium
	}
	if strings.Contains(text, "low") || strings.Contains(text, "minor") {
		return SeverityLow
	}
	return SeverityMedium
}

func extractRecommendations(text string) []Recommendation {
	recommendations := []Recommendation{}
	
	// Look for numbered recommendations or bullet points
	lines := strings.Split(text, "\n")
	priority := 1
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if isRecommendationLine(line) {
			rec := Recommendation{
				Title:       extractRecommendationTitle(line),
				Description: line,
				Priority:    priority,
				Category:    "general",
				Impact:      "medium",
				Effort:      "medium",
			}
			recommendations = append(recommendations, rec)
			priority++
		}
	}
	
	return recommendations
}

func extractActions(text string) []SuggestedAction {
	actions := []SuggestedAction{}
	
	// Look for kubectl commands or specific actions
	lines := strings.Split(text, "\n")
	actionID := 1
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "kubectl ") {
			action := SuggestedAction{
				ID:               fmt.Sprintf("action-%d", actionID),
				Type:             ActionTypeKubectl,
				Title:            fmt.Sprintf("Execute kubectl command"),
				Description:      line,
				Command:          line,
				IsAutomatic:      false,
				RequiresApproval: true,
			}
			actions = append(actions, action)
			actionID++
		}
	}
	
	return actions
}

func isRecommendationLine(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "recommend") || 
		   strings.Contains(line, "suggest") ||
		   strings.Contains(line, "should") ||
		   (len(line) > 20 && (strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") || strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "- ")))
}

func extractRecommendationTitle(line string) string {
	// Extract first part before any punctuation as title
	words := strings.Fields(line)
	if len(words) > 6 {
		return strings.Join(words[:6], " ") + "..."
	}
	return line
}

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