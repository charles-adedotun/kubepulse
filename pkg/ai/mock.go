package ai

import (
	"fmt"
	"strings"
	"time"
)

// getMockInsights returns mock AI insights for testing/demo
func (c *Client) getMockInsights(health *ClusterHealth) *InsightSummary {
	// Count issues
	criticalCount := 0
	for _, check := range health.Checks {
		if check.Status == "unhealthy" {
			criticalCount++
		}
	}

	// Generate dynamic recommendations based on health
	recommendations := []Recommendation{
		{
			Title:       "Regular Health Monitoring",
			Description: "Continue monitoring cluster health with KubePulse for early detection of issues",
			Priority:    1,
			Impact:      "Preventive maintenance",
			Effort:      "Low",
		},
	}

	overallHealth := "Your Kubernetes cluster is operating optimally with all systems healthy."
	if criticalCount > 0 {
		overallHealth = fmt.Sprintf("Detected %d critical issues requiring immediate attention.", criticalCount)
		recommendations = append([]Recommendation{
			{
				Title:       "Address Critical Issues",
				Description: fmt.Sprintf("Investigate and resolve %d unhealthy components", criticalCount),
				Priority:    3,
				Impact:      "Critical - Service availability at risk",
				Effort:      "Medium",
			},
		}, recommendations...)
	}

	// Add performance optimization recommendations
	if health.Score.Weighted < 90 {
		recommendations = append(recommendations, Recommendation{
			Title:       "Performance Optimization",
			Description: "Consider resource optimization to improve cluster efficiency",
			Priority:    2,
			Impact:      "Improved resource utilization",
			Effort:      "Medium",
		})
	}

	// Predicted issues based on trends
	predictedIssues := []string{}
	if health.Score.Weighted < 80 {
		predictedIssues = append(predictedIssues, "Resource pressure may lead to pod evictions if current trends continue")
	}

	return &InsightSummary{
		OverallHealth:   overallHealth,
		CriticalIssues:  criticalCount,
		TrendAnalysis:   "Cluster performance has been stable over the past monitoring period.",
		PredictedIssues: predictedIssues,
		Recommendations: recommendations,
		HealthScore:     health.Score.Weighted,
		AIConfidence:    0.85,
		LastAnalyzed:    time.Now(),
	}
}

// getMockAnalysisResponse returns mock AI analysis response for testing
func (c *Client) getMockAnalysisResponse(request AnalysisRequest) *AnalysisResponse {
	response := &AnalysisResponse{
		ID:         fmt.Sprintf("mock-%s-%d", request.Type, time.Now().Unix()),
		Type:       request.Type,
		Summary:    getMockSummaryForType(request),
		Diagnosis:  getMockDiagnosisForType(request),
		Confidence: 0.85,
		Severity:   SeverityInfo,
		Timestamp:  time.Now(),
		Duration:   50 * time.Millisecond,
		Context:    map[string]interface{}{"request_context": request.Context},
	}

	// Add type-specific mock data
	switch request.Type {
	case AnalysisTypeDiagnostic:
		response.Actions = []SuggestedAction{
			{
				Type:        ActionTypeInvestigate,
				Title:       "Check Pod Logs",
				Description: "Check pod logs for error messages",
				Command:     "kubectl logs <pod-name> --tail=50",
			},
			{
				Type:        ActionTypeInvestigate,
				Title:       "Verify Resources",
				Description: "Verify resource limits and requests",
				Command:     "kubectl describe pod <pod-name>",
			},
		}
		response.Severity = SeverityMedium
		
	case AnalysisTypeHealing:
		response.Actions = []SuggestedAction{
			{
				Type:        ActionTypeRestart,
				Title:       "Restart Pod",
				Description: "Restart the pod to clear transient issues",
				Command:     "kubectl delete pod <pod-name>",
			},
		}
		response.Recommendations = []Recommendation{
			{
				Title:       "Increase resource limits",
				Description: "Consider increasing memory limit to prevent OOM kills",
				Priority:    2,
				Impact:      "Improved stability",
				Effort:      "Low",
			},
		}
		
	case AnalysisTypePredictive:
		response.Summary = "Based on current trends, no immediate issues predicted"
		response.Diagnosis = "Resource usage patterns are within normal parameters"
		
	case AnalysisTypeSummary:
		response.Summary = "Cluster health is generally good with minor optimization opportunities"
		response.Recommendations = []Recommendation{
			{
				Title:       "Resource optimization",
				Description: "Some pods have excessive resource requests",
				Priority:    1,
				Impact:      "Cost reduction",
				Effort:      "Medium",
			},
		}
		
	case AnalysisTypeOptimization:
		response.Actions = []SuggestedAction{
			{
				Type:        ActionTypeConfiguration,
				Title:       "Optimize Resources",
				Description: "Right-size pod resource requests",
				Command:     "kubectl set resources deployment <name> --requests=cpu=100m,memory=128Mi",
			},
		}
		
	case AnalysisTypeRootCause:
		response.Summary = "Root cause identified: Resource constraints leading to pod evictions"
		response.Diagnosis = "Node memory pressure causing kubelet to evict pods"
		response.Severity = SeverityCritical
	}
	
	return response
}

func getMockSummaryForType(request AnalysisRequest) string {
	// Handle natural language queries
	if request.Type == AnalysisTypeSummary && request.Data != nil {
		if question, ok := request.Data["user_question"].(string); ok {
			if strings.Contains(strings.ToLower(question), "crash") || 
			   strings.Contains(strings.ToLower(question), "fail") {
				return "Your pod is likely crashing due to resource constraints or application errors. Common causes include OOM kills, failed health checks, or configuration issues."
			}
			if strings.Contains(strings.ToLower(question), "slow") || 
			   strings.Contains(strings.ToLower(question), "performance") {
				return "Performance issues can stem from insufficient resources, network latency, or inefficient application code. Check CPU/memory utilization and network metrics."
			}
		}
	}
	
	switch request.Type {
	case AnalysisTypeDiagnostic:
		return "Diagnostic analysis reveals potential memory pressure affecting pod stability"
	case AnalysisTypeHealing:
		return "Self-healing actions available to restore service functionality"
	case AnalysisTypePredictive:
		return "Predictive analysis shows stable trends with no immediate concerns"
	case AnalysisTypeOptimization:
		return "Optimization opportunities identified for resource efficiency"
	case AnalysisTypeRootCause:
		return "Root cause analysis points to resource contention on nodes"
	default:
		return "Analysis complete with actionable insights generated"
	}
}

func getMockDiagnosisForType(request AnalysisRequest) string {
	// For natural language queries, provide detailed diagnosis
	if request.Type == AnalysisTypeSummary && request.Data != nil {
		if _, ok := request.Data["user_question"].(string); ok {
			return `Based on cluster analysis:

1. **Recent Events**: Check kubectl events for error messages
2. **Resource Usage**: Monitor CPU and memory consumption
3. **Application Logs**: Review container logs for stack traces
4. **Network Issues**: Verify DNS resolution and service connectivity
5. **Configuration**: Ensure ConfigMaps and Secrets are properly mounted

Recommended next steps:
- Run: kubectl describe pod <pod-name>
- Check: kubectl logs <pod-name> --previous
- Monitor: kubectl top pod <pod-name>`
		}
	}
	
	switch request.Type {
	case AnalysisTypeDiagnostic:
		return "Memory usage exceeding 85% of limits, leading to potential OOM kills. CPU throttling detected."
	case AnalysisTypeHealing:
		return "Restarting pods and adjusting resource limits can mitigate current issues."
	case AnalysisTypePredictive:
		return "No anomalies detected in time-series data. Resource usage following expected patterns."
	case AnalysisTypeOptimization:
		return "25% of pods are over-provisioned. Right-sizing can reduce costs by ~30%."
	case AnalysisTypeRootCause:
		return "Chain of events: High memory usage → Node pressure → Pod eviction → Service disruption"
	default:
		return "Comprehensive analysis completed with focus on stability and performance."
	}
}