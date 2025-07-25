package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

// ContextAwareClient extends the AI client with context-specific data handling
type ContextAwareClient struct {
	*Client
	database        *Database
	contextManager  *ContextManager
	sessionHistory  map[string][]AnalysisSession // cluster -> sessions
}

// ContextManager is defined in context_manager.go

// NewContextAwareClient creates a context-aware AI client
func NewContextAwareClient(config Config, database *Database) *ContextAwareClient {
	client := NewClient(config)
	
	return &ContextAwareClient{
		Client:         client,
		database:       database,
		contextManager: NewContextManager(database),
		sessionHistory: make(map[string][]AnalysisSession),
	}
}

// NewContextManager is defined in context_manager.go

// AnalyzeWithContext performs AI analysis with context-specific data
func (c *ContextAwareClient) AnalyzeWithContext(ctx context.Context, request AnalysisRequestV2) (*AnalysisResult, error) {
	start := time.Now()
	
	// Get or create cluster context
	clusterContext, err := c.contextManager.GetContext(request.ClusterName)
	if err != nil {
		klog.Warningf("Failed to get cluster context for %s: %v", request.ClusterName, err)
		// Continue with empty context
		clusterContext = &ClusterContext{
			ClusterName:     request.ClusterName,
			BaselineMetrics: make(map[string]float64),
			KnownIssues:     []Issue{},
			AIConfidence:    0.5,
			UpdatedAt:       time.Now(),
		}
	}

	// Get historical analysis data for context
	if request.HistoryData == nil {
		since := time.Now().Add(-24 * time.Hour) // Last 24 hours
		history, err := c.database.GetAnalysisHistory(request.ClusterName, since)
		if err != nil {
			klog.V(2).Infof("No analysis history found for cluster %s: %v", request.ClusterName, err)
			history = []AnalysisSession{}
		}
		request.HistoryData = history
	}

	// Get patterns for this cluster
	if request.PatternData == nil {
		patterns, err := c.database.GetPatterns(request.ClusterName, "")
		if err != nil {
			klog.V(2).Infof("No patterns found for cluster %s: %v", request.ClusterName, err)
			patterns = []ClusterPattern{}
		}
		request.PatternData = patterns
	}

	// Build enhanced prompt with cluster context
	enhancedPrompt, err := c.buildContextAwarePrompt(request, clusterContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build context-aware prompt: %w", err)
	}

	// Convert to legacy format for now
	legacyRequest := AnalysisRequest{
		Type:        AnalysisType(request.Type),
		Context:     enhancedPrompt,
		Data:        request.KubectlData,
		Timestamp:   request.Timestamp,
		ClusterInfo: nil, // Will be handled via prompt
	}

	// Perform analysis
	response, err := c.Analyze(ctx, legacyRequest)
	if err != nil {
		// Record failed analysis
		session := &AnalysisSession{
			ID:             request.ID,
			ClusterName:    request.ClusterName,
			AnalysisType:   request.Type,
			KubectlOutputs: c.serializeKubectlData(request.KubectlData),
			AIResponse:     "",
			Confidence:     0.0,
			Timestamp:      time.Now(),
			Duration:       time.Since(start),
			Success:        false,
			ErrorMessage:   err.Error(),
		}
		_ = c.database.StoreAnalysisSession(session)
		return nil, err
	}

	// Convert response to new format
	result := c.convertToAnalysisResult(request.ID, request.ClusterName, response, time.Since(start))

	// Store successful analysis session
	session := &AnalysisSession{
		ID:             request.ID,
		ClusterName:    request.ClusterName,
		AnalysisType:   request.Type,
		KubectlOutputs: c.serializeKubectlData(request.KubectlData),
		AIResponse:     result.Summary,
		Confidence:     result.Confidence,
		Timestamp:      time.Now(),
		Duration:       time.Since(start),
		Success:        true,
		ErrorMessage:   "",
	}

	if err := c.database.StoreAnalysisSession(session); err != nil {
		klog.Errorf("Failed to store analysis session: %v", err)
	}

	// Update cluster context
	c.updateClusterContext(clusterContext, result)
	updates := map[string]interface{}{
		"confidence": result.Confidence,
		"last_analysis": time.Now(),
		"health_score": clusterContext.HealthScore,
	}
	if err := c.contextManager.UpdateContext(clusterContext.ClusterName, updates); err != nil {
		klog.Errorf("Failed to update cluster context: %v", err)
	}

	// Record metrics
	metric := &AIMetric{
		OperationType: request.Type,
		ClusterName:   request.ClusterName,
		ResponseTime:  time.Since(start),
		Confidence:    result.Confidence,
		Success:       true,
		TokensUsed:    result.TokensUsed,
		CostEstimate:  result.CostEstimate,
	}
	if err := c.database.RecordMetric(metric); err != nil {
		klog.Errorf("Failed to record AI metric: %v", err)
	}

	klog.V(2).Infof("Context-aware analysis completed for cluster %s in %v", request.ClusterName, time.Since(start))
	return result, nil
}

// buildContextAwarePrompt creates an enhanced prompt with cluster-specific context
func (c *ContextAwareClient) buildContextAwarePrompt(request AnalysisRequestV2, clusterContext *ClusterContext) (string, error) {
	prompt := fmt.Sprintf(`# Kubernetes Cluster Analysis Request

## Cluster: %s
- **Analysis Type**: %s
- **Timestamp**: %s
- **Context**: %s

## Cluster Context & History
- **Health Score**: %.2f
- **AI Confidence**: %.2f  
- **Last Analysis**: %s
- **Node Count**: %d
- **Namespace Count**: %d

### Baseline Metrics
%s

### Known Issues
%s

### Recent Analysis History (%d sessions)
%s

### Recognized Patterns (%d patterns)
%s

## Current kubectl Data
%s

---

**Instructions:**
Based on the above context and historical data for cluster "%s", provide comprehensive analysis focusing on:

1. **Immediate Issues**: Critical problems requiring immediate attention
2. **Trends**: Changes from baseline metrics and historical patterns  
3. **Context-Aware Insights**: Recommendations based on this cluster's specific history
4. **Actionable Items**: Specific kubectl commands and remediation steps
5. **Confidence Assessment**: How confident you are in your analysis given the available context

Please structure your response with clear sections and actionable recommendations. Consider the cluster's historical behavior and known patterns when making suggestions.`,
		request.ClusterName,
		request.Type,
		request.Timestamp.Format(time.RFC3339),
		request.Context,
		clusterContext.HealthScore,
		clusterContext.AIConfidence,
		clusterContext.LastAnalysis.Format(time.RFC3339),
		clusterContext.NodeCount,
		clusterContext.NamespaceCount,
		c.formatBaslineMetrics(clusterContext.BaselineMetrics),
		c.formatKnownIssues(clusterContext.KnownIssues),
		len(request.HistoryData),
		c.formatAnalysisHistory(request.HistoryData),
		len(request.PatternData),
		c.formatPatterns(request.PatternData),
		c.formatKubectlData(request.KubectlData),
		request.ClusterName,
	)

	return prompt, nil
}

// Helper methods for formatting context data
func (c *ContextAwareClient) formatBaslineMetrics(metrics map[string]float64) string {
	if len(metrics) == 0 {
		return "No baseline metrics available"
	}
	
	result := ""
	for key, value := range metrics {
		result += fmt.Sprintf("- %s: %.2f\n", key, value)
	}
	return result
}

func (c *ContextAwareClient) formatKnownIssues(issues []Issue) string {
	if len(issues) == 0 {
		return "No known issues"
	}
	
	result := ""
	for _, issue := range issues {
		result += fmt.Sprintf("- [%s] %s: %s (Status: %s)\n", 
			issue.Severity, issue.Type, issue.Description, issue.Status)
	}
	return result
}

func (c *ContextAwareClient) formatAnalysisHistory(history []AnalysisSession) string {
	if len(history) == 0 {
		return "No recent analysis history"
	}
	
	result := ""
	for i, session := range history {
		if i >= 5 { // Limit to last 5 sessions for brevity
			break
		}
		status := "Success"
		if !session.Success {
			status = fmt.Sprintf("Failed: %s", session.ErrorMessage)
		}
		result += fmt.Sprintf("- %s [%s]: %s (Confidence: %.2f)\n", 
			session.Timestamp.Format("2006-01-02 15:04"), session.AnalysisType, status, session.Confidence)
	}
	return result
}

func (c *ContextAwareClient) formatPatterns(patterns []ClusterPattern) string {
	if len(patterns) == 0 {
		return "No recognized patterns"
	}
	
	result := ""
	for i, pattern := range patterns {
		if i >= 5 { // Limit to top 5 patterns
			break
		}
		result += fmt.Sprintf("- [%s] %s: %s (Confidence: %.2f, Frequency: %d)\n", 
			pattern.PatternType, pattern.PatternName, pattern.Description, pattern.Confidence, pattern.Frequency)
	}
	return result
}

func (c *ContextAwareClient) formatKubectlData(data map[string]interface{}) string {
	if len(data) == 0 {
		return "No kubectl data provided"
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting kubectl data: %v", err)
	}
	
	return fmt.Sprintf("```json\n%s\n```", string(jsonData))
}

func (c *ContextAwareClient) serializeKubectlData(data map[string]interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func (c *ContextAwareClient) convertToAnalysisResult(requestID, clusterName string, response *AnalysisResponse, duration time.Duration) *AnalysisResult {
	// Convert legacy response to new format
	findings := make([]Finding, 0) // Will extract from summary or other fields

	recommendations := make([]Recommendation, len(response.Recommendations))
	for i, rec := range response.Recommendations {
		recommendations[i] = Recommendation{
			Title:       rec.Title,
			Description: rec.Description,
			Category:    rec.Category,
			Priority:    "medium", // Convert from int priority
			Commands:    []string{}, // Could extract from actions
			Impact:      rec.Impact,
			References:  rec.References,
		}
	}

	return &AnalysisResult{
		ID:                requestID,
		RequestID:         requestID,
		ClusterName:       clusterName,
		AnalysisType:      string(response.Type),
		Summary:           response.Summary,
		Findings:          findings,
		Recommendations:   recommendations,
		KubectlCommands:   []string{}, // Could be extracted from actions
		Confidence:        response.Confidence,
		Severity:          response.Severity,
		ActionRequired:    response.Severity >= SeverityHigh,
		Timestamp:         time.Now(),
		Duration:          duration,
		TokensUsed:        0, // Would need to be tracked
		CostEstimate:      0, // Would need to be calculated
	}
}

func (c *ContextAwareClient) updateClusterContext(context *ClusterContext, result *AnalysisResult) {
	context.LastAnalysis = time.Now()
	context.AIConfidence = result.Confidence
	context.UpdatedAt = time.Now()
	
	// Update health score based on findings
	criticalFindings := 0
	for _, finding := range result.Findings {
		if finding.Severity == SeverityCritical {
			criticalFindings++
		}
	}
	
	if criticalFindings > 0 {
		context.HealthScore = 100.0 - float64(criticalFindings*20) // Reduce by 20 per critical finding
		if context.HealthScore < 0 {
			context.HealthScore = 0
		}
	} else {
		context.HealthScore = 100.0
	}
}

// Context management methods are implemented in context_manager.go