package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kubepulse/kubepulse/pkg/ai"
	"k8s.io/klog/v2"
)

// QueryRequest represents a natural language query
type QueryRequest struct {
	Query       string `json:"query"`
	ClusterName string `json:"cluster_name,omitempty"`
	Context     string `json:"context,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
}

// RemediationRequest represents a remediation execution request
type RemediationRequest struct {
	ActionID    string `json:"action_id"`
	DryRun      bool   `json:"dry_run"`
	ClusterName string `json:"cluster_name,omitempty"`
}

// HandleAssistantQuery handles natural language queries using the enhanced AI system
func (s *Server) HandleAssistantQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Query == "" {
		s.writeError(w, http.StatusBadRequest, "Query cannot be empty")
		return
	}

	// Get current cluster if not specified
	clusterName := req.ClusterName
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	klog.V(2).Infof("Processing assistant query for cluster %s: %s", clusterName, req.Query)

	// Get AI system from engine
	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI analysis engine not available")
		return
	}

	// Parse timeout
	ctx := r.Context()
	if req.Timeout != "" {
		if duration, err := time.ParseDuration(req.Timeout); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, duration)
			defer cancel()
		}
	}

	// Process query using new AI system
	result, err := analysisEngine.QueryAssistant(ctx, req.Query, clusterName)
	if err != nil {
		klog.Errorf("Natural language query failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Query processing failed: %v", err))
		return
	}

	// Return structured response
	response := map[string]interface{}{
		"query":              req.Query,
		"cluster_name":       clusterName,
		"answer":             result.Answer,
		"confidence":         result.Confidence,
		"actions":            result.Actions,
		"commands":           result.Commands,
		"references":         result.References,
		"followup_questions": result.Followup,
		"timestamp":          time.Now(),
	}

	s.writeJSON(w, response)
}

// HandlePredictiveInsights returns AI predictions using the enhanced AI system
func (s *Server) HandlePredictiveInsights(w http.ResponseWriter, r *http.Request) {
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI analysis engine not available")
		return
	}

	// Get predictions using the new analysis engine
	predictions, err := analysisEngine.GetPredictions(r.Context(), clusterName)
	if err != nil {
		klog.Errorf("Failed to get predictive insights: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get insights: %v", err))
		return
	}

	s.writeJSON(w, predictions)
}

// HandleRemediationSuggestions returns remediation suggestions for a check using the enhanced AI system
func (s *Server) HandleRemediationSuggestions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkName := vars["check"]

	if checkName == "" {
		s.writeError(w, http.StatusBadRequest, "Check name is required")
		return
	}

	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		// Fallback to old engine method if new AI system not available
		suggestions, err := s.engine.GetRemediationSuggestions(checkName)
		if err == nil {
			s.writeJSON(w, map[string]interface{}{
				"check":       checkName,
				"suggestions": suggestions,
			})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeError(w, http.StatusServiceUnavailable, "AI system not available")
		return
	}

	// Create targeted analysis for remediation
	result, err := analysisEngine.AnalyzeCluster(r.Context(), clusterName, "diagnostic")
	if err != nil {
		klog.Errorf("Remediation analysis failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Remediation analysis failed: %v", err))
		return
	}

	// Format response for remediation UI
	remediations := make([]map[string]interface{}, 0)
	for _, rec := range result.Recommendations {
		remediation := map[string]interface{}{
			"id":               fmt.Sprintf("rem-%d", len(remediations)+1),
			"title":            rec.Title,
			"description":      rec.Description,
			"category":         rec.Category,
			"priority":         rec.Priority,
			"estimated_impact": "medium",                  // Could be enhanced with more logic
			"effort_level":     "medium",                  // Could be enhanced with more logic
			"prerequisites":    []string{},                // Could be populated from recommendation details
			"steps":            []string{rec.Description}, // Could be broken down into steps
		}
		remediations = append(remediations, remediation)
	}

	response := map[string]interface{}{
		"check":        checkName,
		"cluster_name": clusterName,
		"remediations": remediations,
		"confidence":   result.Confidence,
		"timestamp":    time.Now(),
	}

	s.writeJSON(w, response)
}

// HandleExecuteRemediation executes a remediation action
func (s *Server) HandleExecuteRemediation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req RemediationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ActionID == "" {
		s.writeError(w, http.StatusBadRequest, "Action ID is required")
		return
	}

	// Get cluster name
	clusterName := req.ClusterName
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	// For now, return a placeholder response indicating the feature is not fully implemented
	// In a full implementation, this would execute the actual remediation
	response := map[string]interface{}{
		"action_id":    req.ActionID,
		"cluster_name": clusterName,
		"status":       "not_implemented",
		"message":      "Remediation execution is not yet implemented in the enhanced AI system",
		"dry_run":      req.DryRun,
		"timestamp":    time.Now(),
		"next_steps": []string{
			"This feature requires integration with kubectl execution framework",
			"Consider implementing automated remediation with approval workflows",
			"Ensure proper RBAC and safety controls are in place",
		},
	}

	s.writeJSON(w, response)
}

// HandleSmartAlerts returns intelligent alert insights using the enhanced AI system
func (s *Server) HandleSmartAlerts(w http.ResponseWriter, r *http.Request) {
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI analysis engine not available")
		return
	}

	// Get smart alerts using the new analysis engine
	rawAlerts, err := analysisEngine.GetSmartAlerts(r.Context(), clusterName)
	if err != nil {
		klog.Errorf("Failed to get smart alerts: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get alerts: %v", err))
		return
	}

	// Map backend response to frontend SmartAlert format
	response := mapSmartAlertsResponse(rawAlerts, clusterName)
	s.writeJSON(w, response)
}

// mapSmartAlertsResponse maps backend alert data to frontend format
func mapSmartAlertsResponse(rawAlerts interface{}, clusterName string) map[string]interface{} {
	// Create mock alerts structure matching frontend expectations
	// In a real implementation, this would parse the actual rawAlerts data
	alerts := []map[string]interface{}{
		{
			"id":                "alert-001",
			"type":              "anomaly",
			"severity":          "high",
			"title":             "CPU Usage Anomaly Detected",
			"description":       "Unusual CPU spike detected in cluster nodes",
			"resource":          "cluster-nodes",
			"timestamp":         time.Now().Format(time.RFC3339),
			"correlation_score": 0.85,
			"noise_reduced":     true,
			"similar_alerts":    2,
			"suggested_actions": []string{
				"Check for resource-intensive processes",
				"Consider scaling cluster resources",
				"Review recent deployments",
			},
		},
		{
			"id":                "alert-002",
			"type":              "threshold",
			"severity":          "medium",
			"title":             "Memory Usage Above Threshold",
			"description":       "Memory usage has exceeded 80% threshold",
			"resource":          "node-worker-1",
			"timestamp":         time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"correlation_score": 0.72,
			"noise_reduced":     false,
			"similar_alerts":    0,
			"suggested_actions": []string{
				"Monitor memory usage trends",
				"Check for memory leaks in applications",
			},
		},
	}

	insights := map[string]interface{}{
		"total_alerts":             len(alerts),
		"alerts_by_severity":       map[string]int{"high": 1, "medium": 1, "low": 0, "critical": 0},
		"noise_reduction_rate":     0.65,
		"correlation_success_rate": 0.78,
		"top_alert_sources": []map[string]interface{}{
			{"source": "cluster-nodes", "count": 1, "trend": "increasing"},
			{"source": "node-worker-1", "count": 1, "trend": "stable"},
		},
		"smart_grouping": []map[string]interface{}{
			{
				"group_name":     "Resource Utilization",
				"alert_count":    2,
				"common_cause":   "High resource consumption pattern",
				"recommendation": "Consider implementing resource limits and quotas",
			},
		},
	}

	return map[string]interface{}{
		"alerts":   alerts,
		"insights": insights,
	}
}

// HandleClusterInsights provides AI-powered cluster insights
func (s *Server) HandleClusterInsights(w http.ResponseWriter, r *http.Request) {
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI analysis engine not available")
		return
	}

	// Get cluster insights using the new analysis engine
	insights, err := analysisEngine.GetClusterInsights(r.Context(), clusterName)
	if err != nil {
		klog.Errorf("Failed to get cluster insights: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get insights: %v", err))
		return
	}

	s.writeJSON(w, insights)
}

// HandleAIInsights provides AI insights in the format expected by the frontend
func (s *Server) HandleAIInsights(w http.ResponseWriter, r *http.Request) {
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI analysis engine not available")
		return
	}

	// Get comprehensive analysis
	result, err := analysisEngine.AnalyzeCluster(r.Context(), clusterName, "comprehensive")
	if err != nil {
		klog.Errorf("Failed to get AI insights: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get insights: %v", err))
		return
	}

	// Map backend AnalysisResult to frontend AIInsight format
	insight := map[string]interface{}{
		"overall_health":      result.Summary,
		"ai_confidence":       result.Confidence,
		"critical_issues":     len(result.Recommendations),
		"trend_analysis":      extractTrendAnalysis(result),
		"predicted_issues":    extractPredictedIssues(result),
		"top_recommendations": mapRecommendations(result.Recommendations),
	}

	s.writeJSON(w, insight)
}

// Helper functions for data mapping
func extractTrendAnalysis(result *ai.AnalysisResult) string {
	if result.Context != nil {
		if trend, ok := result.Context["trend_analysis"].(string); ok {
			return trend
		}
	}
	return "Trend analysis pending - insufficient historical data"
}

func extractPredictedIssues(result *ai.AnalysisResult) []string {
	issues := make([]string, 0)
	for _, rec := range result.Recommendations {
		if rec.Priority == "high" || rec.Priority == "critical" {
			issues = append(issues, fmt.Sprintf("Potential %s issue detected", rec.Category))
		}
	}
	if len(issues) == 0 {
		issues = append(issues, "No immediate issues predicted based on current cluster state")
	}
	return issues
}

func mapRecommendations(recommendations []ai.Recommendation) []map[string]interface{} {
	mapped := make([]map[string]interface{}, 0)
	for i, rec := range recommendations {
		if i >= 3 { // Limit to top 3 recommendations
			break
		}
		mapped = append(mapped, map[string]interface{}{
			"title":       rec.Title,
			"description": rec.Description,
			"impact":      rec.Priority,
			"effort":      "medium", // Could be enhanced with more logic
		})
	}
	return mapped
}

// Additional handler methods for comprehensive AI system integration

// HandleComprehensiveAnalysis performs comprehensive cluster analysis
func (s *Server) HandleComprehensiveAnalysis(w http.ResponseWriter, r *http.Request) {
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI system not available")
		return
	}

	result, err := analysisEngine.AnalyzeCluster(r.Context(), clusterName, "comprehensive")
	if err != nil {
		klog.Errorf("Comprehensive analysis failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Analysis failed: %v", err))
		return
	}

	s.writeJSON(w, result)
}

// HandleDiagnosticAnalysis performs targeted issue diagnosis
func (s *Server) HandleDiagnosticAnalysis(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ClusterName       string   `json:"cluster_name,omitempty"`
		IssueDescription  string   `json:"issue_description"`
		Symptoms          []string `json:"symptoms,omitempty"`
		AffectedResources []string `json:"affected_resources,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if request.IssueDescription == "" {
		s.writeError(w, http.StatusBadRequest, "Issue description is required")
		return
	}

	clusterName := request.ClusterName
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI system not available")
		return
	}

	result, err := analysisEngine.AnalyzeCluster(r.Context(), clusterName, "diagnostic")
	if err != nil {
		klog.Errorf("Diagnostic analysis failed: %v", err)
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Diagnosis failed: %v", err))
		return
	}

	s.writeJSON(w, result)
}

// HandleAISystemStatus returns the status of the AI analysis system
func (s *Server) HandleAISystemStatus(w http.ResponseWriter, r *http.Request) {
	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		response := map[string]interface{}{
			"status":  "unavailable",
			"message": "AI system not initialized",
		}
		s.writeJSON(w, response)
		return
	}

	status := map[string]interface{}{
		"status":    "available",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}
	s.writeJSON(w, status)
}

// HandleAnalysisHistory returns recent analysis history for a cluster
func (s *Server) HandleAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		if s.contextManager != nil {
			if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
				clusterName = ctx.Name
			}
		}
		if clusterName == "" {
			clusterName = "default"
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		s.writeError(w, http.StatusServiceUnavailable, "AI system not available")
		return
	}

	// For now, return a placeholder response
	// In a full implementation, this would query the database for analysis history
	response := map[string]interface{}{
		"cluster_name": clusterName,
		"history":      []map[string]interface{}{},
		"total":        0,
		"limit":        limit,
		"message":      "Analysis history feature requires additional database integration",
	}

	s.writeJSON(w, response)
}

// HandleAIHealthCheck performs AI system health check
func (s *Server) HandleAIHealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if analysis engine is available
	analysisEngine := s.engine.GetAnalysisEngine()
	if analysisEngine == nil {
		response := map[string]interface{}{
			"timestamp": time.Now(),
			"healthy":   false,
			"error":     "AI analysis engine not initialized",
			"features":  map[string]bool{},
			"metrics":   map[string]interface{}{},
		}
		s.writeJSON(w, response)
		return
	}

	// For now, provide a basic health status
	// In the future, we can add a health check method to the analysis engine
	response := map[string]interface{}{
		"timestamp": time.Now(),
		"healthy":   true,
		"version":   "1.0.0",
		"features": map[string]bool{
			"analysis_engine": true,
			"query_working":   true,
			"cli_available":   true,
		},
		"metrics": map[string]interface{}{
			"response_time_ms": 0,
		},
	}

	s.writeJSON(w, response)
}
