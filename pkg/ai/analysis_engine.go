package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// AnalysisEngine orchestrates all AI analysis capabilities
type AnalysisEngine struct {
	client       *ContextAwareClient
	database     *Database
	executor     *KubectlExecutor
	toolRegistry *ToolRegistry
	promptBuilder *PromptBuilder
}

// AnalysisEngineConfig holds configuration for the analysis engine
type AnalysisEngineConfig struct {
	AIClient *Client
	Database *Database
	Executor *KubectlExecutor
}

// NewAnalysisEngine creates a new analysis engine
func NewAnalysisEngine(config AnalysisEngineConfig) *AnalysisEngine {
	// Create context-aware client
	contextAwareClient := NewContextAwareClient(
		Config{
			ClaudePath:   "claude",
			MaxTurns:     3,
			Timeout:      120 * time.Second,
			SystemPrompt: getDefaultSystemPrompt(),
		},
		config.Database,
	)

	// Initialize tool registry
	toolRegistry := NewToolRegistry(config.Executor, config.Database)

	return &AnalysisEngine{
		client:        contextAwareClient,
		database:      config.Database,
		executor:      config.Executor,
		toolRegistry:  toolRegistry,
		promptBuilder: NewPromptBuilder(),
	}
}

// AnalyzeCluster performs comprehensive cluster analysis using kubectl runbook tools
func (ae *AnalysisEngine) AnalyzeCluster(ctx context.Context, clusterName string, analysisType string) (*AnalysisResult, error) {
	klog.V(2).Infof("Starting cluster analysis for %s (type: %s)", clusterName, analysisType)

	// Execute all kubectl tools to gather comprehensive data
	toolResults, err := ae.toolRegistry.ExecuteAll(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to execute kubectl tools: %w", err)
	}

	// Convert tool results to kubectl data format
	kubectlData := ae.convertToolResultsToKubectlData(toolResults)

	// Create analysis request
	request := AnalysisRequestV2{
		ID:          fmt.Sprintf("analysis-%d", time.Now().Unix()),
		Type:        analysisType,
		ClusterName: clusterName,
		Context:     fmt.Sprintf("Comprehensive cluster analysis using kubectl runbook"),
		KubectlData: kubectlData,
		Timestamp:   time.Now(),
		Timeout:     120 * time.Second,
		Priority:    "normal",
	}

	// Perform context-aware analysis
	result, err := ae.client.AnalyzeWithContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Enhance result with tool metadata
	ae.enhanceResultWithToolData(result, toolResults)

	klog.V(2).Infof("Cluster analysis completed for %s: confidence=%.2f", clusterName, result.Confidence)
	return result, nil
}

// QueryAssistant handles natural language queries with context
func (ae *AnalysisEngine) QueryAssistant(ctx context.Context, query string, clusterName string) (*QueryResponse, error) {
	klog.V(2).Infof("Processing assistant query for cluster %s: %s", clusterName, query)

	// Execute relevant kubectl tools based on query type
	toolNames := ae.selectToolsForQuery(query)
	toolResults := make(map[string]*KubectlToolResult)

	for _, toolName := range toolNames {
		if tool, exists := ae.toolRegistry.GetTool(toolName); exists {
			result, err := tool.Execute(ctx, clusterName)
			if err != nil {
				klog.Warningf("Tool %s failed: %v", toolName, err)
				continue
			}
			toolResults[toolName] = result
		}
	}

	// Convert to kubectl data
	kubectlData := ae.convertToolResultsToKubectlData(toolResults)

	// Create focused analysis request
	request := AnalysisRequestV2{
		ID:          fmt.Sprintf("query-%d", time.Now().Unix()),
		Type:        "query",
		ClusterName: clusterName,
		Context:     fmt.Sprintf("Natural language query: %s", query),
		KubectlData: kubectlData,
		Timestamp:   time.Now(),
		Timeout:     60 * time.Second,
		Priority:    "high",
	}

	// Perform analysis
	result, err := ae.client.AnalyzeWithContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("query analysis failed: %w", err)
	}

	// Convert to query response format
	response := &QueryResponse{
		Answer:     result.Summary,
		Confidence: result.Confidence,
		Actions:    ae.extractActionsFromResult(result),
		Commands:   result.KubectlCommands,
		References: ae.extractReferencesFromResult(result),
		Followup:   result.FollowUpQuestions,
	}

	return response, nil
}

// GetClusterInsights provides AI-powered insights for the cluster
func (ae *AnalysisEngine) GetClusterInsights(ctx context.Context, clusterName string) (*InsightsResponse, error) {
	// Execute comprehensive analysis
	result, err := ae.AnalyzeCluster(ctx, clusterName, "comprehensive")
	if err != nil {
		return nil, err
	}

	// Convert to insights format
	insights := &InsightsResponse{
		ClusterName:     clusterName,
		OverallHealth:   ae.calculateOverallHealth(result),
		CriticalIssues:  ae.extractCriticalIssues(result),
		Recommendations: result.Recommendations,
		Confidence:      result.Confidence,
		LastAnalysis:    result.Timestamp,
		Trends:          ae.calculateTrends(clusterName),
	}

	return insights, nil
}

// GetSmartAlerts provides intelligent alerting based on patterns
func (ae *AnalysisEngine) GetSmartAlerts(ctx context.Context, clusterName string) (*SmartAlertsResponse, error) {
	// Get recent patterns and analysis history
	patterns, err := ae.database.GetPatterns(clusterName, "")
	if err != nil {
		patterns = []ClusterPattern{} // Continue with empty patterns
	}

	since := time.Now().Add(-24 * time.Hour)
	history, err := ae.database.GetAnalysisHistory(clusterName, since)
	if err != nil {
		history = []AnalysisSession{} // Continue with empty history
	}

	// Execute targeted tools for alerting
	alertTools := []string{"control-plane", "nodes", "workloads"}
	toolResults := make(map[string]*KubectlToolResult)

	for _, toolName := range alertTools {
		if tool, exists := ae.toolRegistry.GetTool(toolName); exists {
			result, err := tool.Execute(ctx, clusterName)
			if err != nil {
				continue
			}
			toolResults[toolName] = result
		}
	}

	// Analyze for smart alerts
	alerts := ae.generateSmartAlerts(toolResults, patterns, history)

	response := &SmartAlertsResponse{
		ClusterName: clusterName,
		Alerts:      alerts,
		Timestamp:   time.Now(),
		Confidence:  ae.calculateAlertConfidence(alerts),
	}

	return response, nil
}

// GetPredictions provides predictive analytics
func (ae *AnalysisEngine) GetPredictions(ctx context.Context, clusterName string) (*PredictionsResponse, error) {
	// Get metrics and historical data
	since := time.Now().Add(-7 * 24 * time.Hour) // Last week
	history, err := ae.database.GetAnalysisHistory(clusterName, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis history: %w", err)
	}

	// Execute resource-focused tools
	resourceTools := []string{"nodes", "workloads"}
	toolResults := make(map[string]*KubectlToolResult)

	for _, toolName := range resourceTools {
		if tool, exists := ae.toolRegistry.GetTool(toolName); exists {
			result, err := tool.Execute(ctx, clusterName)
			if err != nil {
				continue
			}
			toolResults[toolName] = result
		}
	}

	// Generate predictions based on trends
	predictions := ae.generatePredictions(toolResults, history)

	response := &PredictionsResponse{
		ClusterName: clusterName,
		Predictions: predictions,
		Timestamp:   time.Now(),
		Confidence:  ae.calculatePredictionConfidence(predictions),
	}

	return response, nil
}

// Helper methods

func (ae *AnalysisEngine) convertToolResultsToKubectlData(toolResults map[string]*KubectlToolResult) map[string]interface{} {
	kubectlData := make(map[string]interface{})

	for toolName, result := range toolResults {
		kubectlData[toolName] = map[string]interface{}{
			"outputs":        result.Outputs,
			"errors":         result.Errors,
			"summary":        result.Summary,
			"success":        result.Success,
			"execution_time": result.ExecutionTime.Milliseconds(),
			"metadata":       result.Metadata,
		}
	}

	return kubectlData
}

func (ae *AnalysisEngine) enhanceResultWithToolData(result *AnalysisResult, toolResults map[string]*KubectlToolResult) {
	if result.Context == nil {
		result.Context = make(map[string]interface{})
	}

	result.Context["tool_results"] = len(toolResults)
	result.Context["successful_tools"] = ae.countSuccessfulTools(toolResults)
	result.Context["analysis_timestamp"] = time.Now().Format(time.RFC3339)
}

func (ae *AnalysisEngine) selectToolsForQuery(query string) []string {
	// Simple keyword-based tool selection
	// This could be enhanced with ML-based classification
	query = fmt.Sprintf(" %s ", query) // Add spaces for word boundary matching

	selectedTools := []string{}

	// Always include control-plane for infrastructure queries
	if containsAny(query, []string{"api", "control", "etcd", "scheduler", "controller"}) {
		selectedTools = append(selectedTools, "control-plane")
	}

	// Node-related queries
	if containsAny(query, []string{"node", "capacity", "resource", "cpu", "memory"}) {
		selectedTools = append(selectedTools, "nodes")
	}

	// Workload-related queries
	if containsAny(query, []string{"pod", "deployment", "service", "workload", "app"}) {
		selectedTools = append(selectedTools, "workloads")
	}

	// If no specific tools selected, use all major ones
	if len(selectedTools) == 0 {
		selectedTools = []string{"control-plane", "nodes", "workloads"}
	}

	return selectedTools
}

func containsAny(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (ae *AnalysisEngine) extractActionsFromResult(result *AnalysisResult) []string {
	actions := []string{}
	for _, rec := range result.Recommendations {
		if len(rec.Commands) > 0 {
			actions = append(actions, rec.Description)
		}
	}
	return actions
}

func (ae *AnalysisEngine) extractReferencesFromResult(result *AnalysisResult) []string {
	refs := []string{}
	for _, rec := range result.Recommendations {
		refs = append(refs, rec.References...)
	}
	return refs
}

func (ae *AnalysisEngine) countSuccessfulTools(toolResults map[string]*KubectlToolResult) int {
	count := 0
	for _, result := range toolResults {
		if result.Success {
			count++
		}
	}
	return count
}

// Response types for the API

type InsightsResponse struct {
	ClusterName     string          `json:"cluster_name"`
	OverallHealth   string          `json:"overall_health"`
	CriticalIssues  []Finding       `json:"critical_issues"`
	Recommendations []Recommendation `json:"recommendations"`
	Confidence      float64         `json:"confidence"`
	LastAnalysis    time.Time       `json:"last_analysis"`
	Trends          []Trend         `json:"trends"`
}

type SmartAlertsResponse struct {
	ClusterName string      `json:"cluster_name"`
	Alerts      []SmartAlert `json:"alerts"`
	Timestamp   time.Time   `json:"timestamp"`
	Confidence  float64     `json:"confidence"`
}

type PredictionsResponse struct {
	ClusterName string      `json:"cluster_name"`
	Predictions []Prediction `json:"predictions"`
	Timestamp   time.Time   `json:"timestamp"`
	Confidence  float64     `json:"confidence"`
}

// SmartAlert and Prediction types are defined in database_types.go

type Trend struct {
	Metric    string  `json:"metric"`
	Direction string  `json:"direction"` // "up", "down", "stable"
	Change    float64 `json:"change"`
	Period    string  `json:"period"`
}

// Placeholder implementations for helper methods
func (ae *AnalysisEngine) calculateOverallHealth(result *AnalysisResult) string {
	criticalCount := 0
	for _, finding := range result.Findings {
		if finding.Severity == SeverityCritical {
			criticalCount++
		}
	}
	
	if criticalCount > 0 {
		return "Critical"
	}
	if len(result.Findings) > 5 {
		return "Warning"
	}
	return "Healthy"
}

func (ae *AnalysisEngine) extractCriticalIssues(result *AnalysisResult) []Finding {
	critical := []Finding{}
	for _, finding := range result.Findings {
		if finding.Severity == SeverityCritical {
			critical = append(critical, finding)
		}
	}
	return critical
}

func (ae *AnalysisEngine) calculateTrends(clusterName string) []Trend {
	// Placeholder - would analyze historical data
	return []Trend{}
}

func (ae *AnalysisEngine) generateSmartAlerts(toolResults map[string]*KubectlToolResult, patterns []ClusterPattern, history []AnalysisSession) []SmartAlert {
	alerts := []SmartAlert{}
	
	// Simple example: check for failed pods
	if workloadResult, exists := toolResults["workloads"]; exists {
		if failedPods, exists := workloadResult.Metadata["extracted_metrics"]; exists {
			if metrics, ok := failedPods.(map[string]interface{}); ok {
				if problemPods, exists := metrics["problem_pods"]; exists {
					if count, ok := problemPods.(int); ok && count > 0 {
						alerts = append(alerts, SmartAlert{
							ID:          fmt.Sprintf("failed-pods-%d", time.Now().Unix()),
							Type:        "workload-issue",
							Severity:    "warning",
							Title:       "Failed Pods Detected",
							Description: fmt.Sprintf("%d pods are in non-running state", count),
							Evidence:    []string{"kubectl get pods -A --field-selector status.phase!=Running"},
							Timestamp:   time.Now(),
						})
					}
				}
			}
		}
	}
	
	return alerts
}

func (ae *AnalysisEngine) generatePredictions(toolResults map[string]*KubectlToolResult, history []AnalysisSession) []Prediction {
	predictions := []Prediction{}
	
	// Simple example: resource usage prediction
	if nodeResult, exists := toolResults["nodes"]; exists {
		if nodeResult.Success {
			predictions = append(predictions, Prediction{
				ID:          fmt.Sprintf("resource-trend-%d", time.Now().Unix()),
				Type:        "resource-usage",
				Description: "Node resource usage trending upward",
				Likelihood:  0.7,
				Timeline:    "next 7 days",
				Impact:      "medium",
				Evidence:    []string{"kubectl top nodes"},
			})
		}
	}
	
	return predictions
}

func (ae *AnalysisEngine) calculateAlertConfidence(alerts []SmartAlert) float64 {
	if len(alerts) == 0 {
		return 1.0
	}
	return 0.8 // Placeholder
}

func (ae *AnalysisEngine) calculatePredictionConfidence(predictions []Prediction) float64 {
	if len(predictions) == 0 {
		return 1.0
	}
	
	total := 0.0
	for _, pred := range predictions {
		total += pred.Likelihood
	}
	return total / float64(len(predictions))
}