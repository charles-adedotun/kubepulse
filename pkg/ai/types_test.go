package ai

import (
	"testing"
	"time"
)

func TestAnalysisType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    AnalysisType
		expected string
	}{
		{"diagnostic", AnalysisTypeDiagnostic, "diagnostic"},
		{"healing", AnalysisTypeHealing, "healing"},
		{"predictive", AnalysisTypePredictive, "predictive"},
		{"optimization", AnalysisTypeOptimization, "optimization"},
		{"summary", AnalysisTypeSummary, "summary"},
		{"root_cause", AnalysisTypeRootCause, "root_cause"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.value))
			}
		})
	}
}

func TestSeverityLevel_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    SeverityLevel
		expected string
	}{
		{"low", SeverityLow, "low"},
		{"medium", SeverityMedium, "medium"},
		{"high", SeverityHigh, "high"},
		{"critical", SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.value))
			}
		})
	}
}

func TestActionType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    ActionType
		expected string
	}{
		{"kubectl", ActionTypeKubectl, "kubectl"},
		{"script", ActionTypeScript, "script"},
		{"manual", ActionTypeManual, "manual"},
		{"restart", ActionTypeRestart, "restart"},
		{"scale", ActionTypeScale, "scale"},
		{"configuration", ActionTypeConfiguration, "configuration"},
		{"investigate", ActionTypeInvestigate, "investigate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.value))
			}
		})
	}
}

func TestHealthStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    HealthStatus
		expected string
	}{
		{"healthy", HealthStatusHealthy, "healthy"},
		{"degraded", HealthStatusDegraded, "degraded"},
		{"unhealthy", HealthStatusUnhealthy, "unhealthy"},
		{"unknown", HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.value))
			}
		})
	}
}

func TestMetricType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		value    MetricType
		expected string
	}{
		{"gauge", MetricTypeGauge, "gauge"},
		{"counter", MetricTypeCounter, "counter"},
		{"histogram", MetricTypeHistogram, "histogram"},
		{"summary", MetricTypeSummary, "summary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.value))
			}
		})
	}
}

func TestAnalysisRequest_Structure(t *testing.T) {
	now := time.Now()
	data := map[string]interface{}{
		"pods":  5,
		"nodes": 3,
	}

	healthCheck := &CheckResult{
		Name:      "test-check",
		Status:    HealthStatusHealthy,
		Message:   "All good",
		Timestamp: now,
	}

	clusterInfo := &ClusterHealth{
		ClusterName: "test-cluster",
		Status:      HealthStatusHealthy,
		Timestamp:   now,
	}

	request := AnalysisRequest{
		Type:        AnalysisTypeDiagnostic,
		Context:     "test-context",
		Data:        data,
		HealthCheck: healthCheck,
		ClusterInfo: clusterInfo,
		Timestamp:   now,
	}

	if request.Type != AnalysisTypeDiagnostic {
		t.Errorf("expected type %s, got %s", AnalysisTypeDiagnostic, request.Type)
	}

	if request.Context != "test-context" {
		t.Errorf("expected context 'test-context', got %s", request.Context)
	}

	if request.Data["pods"] != 5 {
		t.Errorf("expected pods 5, got %v", request.Data["pods"])
	}

	if request.HealthCheck.Name != "test-check" {
		t.Errorf("expected health check name 'test-check', got %s", request.HealthCheck.Name)
	}

	if request.ClusterInfo.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got %s", request.ClusterInfo.ClusterName)
	}

	if !request.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, request.Timestamp)
	}
}

func TestAnalysisResponse_Structure(t *testing.T) {
	now := time.Now()
	duration := 5 * time.Second

	recommendations := []Recommendation{
		{
			Title:       "Test Recommendation",
			Description: "Test description",
			Priority:    1,
			Impact:      "medium",
		},
	}

	actions := []SuggestedAction{
		{
			ID:          "action-1",
			Type:        ActionTypeKubectl,
			Command:     "kubectl get pods",
			Description: "Check pod status",
		},
	}

	context := map[string]interface{}{
		"namespace": "default",
		"resource":  "pod",
	}

	response := AnalysisResponse{
		ID:              "analysis-123",
		Type:            AnalysisTypeDiagnostic,
		Summary:         "Test summary",
		Diagnosis:       "Test diagnosis",
		Confidence:      0.95,
		Severity:        SeverityHigh,
		Recommendations: recommendations,
		Actions:         actions,
		Context:         context,
		Timestamp:       now,
		Duration:        duration,
	}

	if response.ID != "analysis-123" {
		t.Errorf("expected ID 'analysis-123', got %s", response.ID)
	}

	if response.Type != AnalysisTypeDiagnostic {
		t.Errorf("expected type %s, got %s", AnalysisTypeDiagnostic, response.Type)
	}

	if response.Summary != "Test summary" {
		t.Errorf("expected summary 'Test summary', got %s", response.Summary)
	}

	if response.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", response.Confidence)
	}

	if response.Severity != SeverityHigh {
		t.Errorf("expected severity %s, got %s", SeverityHigh, response.Severity)
	}

	if len(response.Recommendations) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(response.Recommendations))
	}

	if response.Recommendations[0].Title != "Test Recommendation" {
		t.Errorf("expected recommendation title 'Test Recommendation', got %s", response.Recommendations[0].Title)
	}

	if len(response.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(response.Actions))
	}

	if response.Actions[0].Command != "kubectl get pods" {
		t.Errorf("expected action command 'kubectl get pods', got %s", response.Actions[0].Command)
	}

	if response.Context["namespace"] != "default" {
		t.Errorf("expected namespace 'default', got %v", response.Context["namespace"])
	}

	if response.Duration != duration {
		t.Errorf("expected duration %v, got %v", duration, response.Duration)
	}
}

func TestRecommendation_Structure(t *testing.T) {
	rec := Recommendation{
		Title:       "Scale Up Deployment",
		Description: "Increase replica count to handle increased load",
		Priority:    1,
		Impact:      "medium",
		Category:    "scaling",
		Effort:      "low",
	}

	if rec.Title != "Scale Up Deployment" {
		t.Errorf("expected title 'Scale Up Deployment', got %s", rec.Title)
	}

	if rec.Priority != 1 {
		t.Errorf("expected priority 1, got %d", rec.Priority)
	}

	if rec.Impact != "medium" {
		t.Errorf("expected impact 'medium', got %s", rec.Impact)
	}

	if rec.Category != "scaling" {
		t.Errorf("expected category 'scaling', got %s", rec.Category)
	}
}

func TestSuggestedAction_Structure(t *testing.T) {
	action := SuggestedAction{
		ID:               "action-456",
		Type:             ActionTypeKubectl,
		Title:            "Scale Deployment",
		Command:          "kubectl scale deployment myapp --replicas=5",
		Description:      "Scale deployment to 5 replicas",
		IsAutomatic:      false,
		RequiresApproval: true,
	}

	if action.ID != "action-456" {
		t.Errorf("expected ID 'action-456', got %s", action.ID)
	}

	if action.Type != ActionTypeKubectl {
		t.Errorf("expected type %s, got %s", ActionTypeKubectl, action.Type)
	}

	if action.Title != "Scale Deployment" {
		t.Errorf("expected title 'Scale Deployment', got %s", action.Title)
	}

	if action.IsAutomatic {
		t.Error("expected IsAutomatic to be false")
	}

	if !action.RequiresApproval {
		t.Error("expected RequiresApproval to be true")
	}
}

func TestCheckResult_Structure(t *testing.T) {
	now := time.Now()
	details := map[string]interface{}{
		"pod_count":  5,
		"node_ready": true,
	}

	result := CheckResult{
		Name:      "pod-health-check",
		Status:    HealthStatusDegraded,
		Message:   "Some pods are not ready",
		Details:   details,
		Timestamp: now,
	}

	if result.Name != "pod-health-check" {
		t.Errorf("expected name 'pod-health-check', got %s", result.Name)
	}

	if result.Status != HealthStatusDegraded {
		t.Errorf("expected status %s, got %s", HealthStatusDegraded, result.Status)
	}

	if result.Details["pod_count"] != 5 {
		t.Errorf("expected pod_count 5, got %v", result.Details["pod_count"])
	}

	if result.Message != "Some pods are not ready" {
		t.Errorf("expected message 'Some pods are not ready', got %s", result.Message)
	}
}

func TestClusterHealth_Structure(t *testing.T) {
	now := time.Now()

	health := ClusterHealth{
		ClusterName: "test-cluster",
		Status:      HealthStatusHealthy,
		Timestamp:   now,
	}

	if health.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got %s", health.ClusterName)
	}

	if health.Status != HealthStatusHealthy {
		t.Errorf("expected status %s, got %s", HealthStatusHealthy, health.Status)
	}

	if !health.Timestamp.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, health.Timestamp)
	}
}

func TestMetric_Structure(t *testing.T) {
	now := time.Now()
	labels := map[string]string{
		"node": "worker-1",
		"zone": "us-west-1a",
	}

	metric := Metric{
		Name:      "cpu_usage",
		Value:     75.5,
		Unit:      "percent",
		Type:      MetricTypeGauge,
		Labels:    labels,
		Timestamp: now,
	}

	if metric.Name != "cpu_usage" {
		t.Errorf("expected name 'cpu_usage', got %s", metric.Name)
	}

	if metric.Value != 75.5 {
		t.Errorf("expected value 75.5, got %f", metric.Value)
	}

	if metric.Type != MetricTypeGauge {
		t.Errorf("expected type %s, got %s", MetricTypeGauge, metric.Type)
	}

	if metric.Labels["node"] != "worker-1" {
		t.Errorf("expected node label 'worker-1', got %s", metric.Labels["node"])
	}
}

func TestInsightSummary_Structure(t *testing.T) {
	now := time.Now()

	recommendations := []Recommendation{
		{
			Title:    "Optimize Resources",
			Priority: 1,
			Impact:   "high",
		},
	}

	summary := InsightSummary{
		OverallHealth:   "healthy",
		CriticalIssues:  0,
		Recommendations: recommendations,
		TrendAnalysis:   "Stable performance",
		PredictedIssues: []string{"none"},
		HealthScore:     95.5,
		AIConfidence:    0.88,
		LastAnalyzed:    now,
		Context: map[string]interface{}{
			"cluster": "production",
		},
	}

	if summary.OverallHealth != "healthy" {
		t.Errorf("expected overall health 'healthy', got %s", summary.OverallHealth)
	}

	if summary.CriticalIssues != 0 {
		t.Errorf("expected critical issues 0, got %d", summary.CriticalIssues)
	}

	if len(summary.Recommendations) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(summary.Recommendations))
	}

	if summary.HealthScore != 95.5 {
		t.Errorf("expected health score 95.5, got %f", summary.HealthScore)
	}

	if summary.AIConfidence != 0.88 {
		t.Errorf("expected AI confidence 0.88, got %f", summary.AIConfidence)
	}
}

func TestDiagnosticContext_Structure(t *testing.T) {
	metrics := []Metric{
		{
			Name:  "cpu_usage",
			Value: 80.0,
			Type:  MetricTypeGauge,
		},
	}

	checks := []CheckResult{
		{
			Name:   "pod-health",
			Status: HealthStatusHealthy,
		},
	}

	context := DiagnosticContext{
		ClusterName:    "test-cluster",
		Namespace:      "default",
		ResourceType:   "deployment",
		ResourceName:   "myapp",
		ErrorLogs:      []string{"Error connecting to database"},
		Events:         []string{"Pod started successfully"},
		Metrics:        metrics,
		RelatedChecks:  checks,
		HistoricalData: checks,
		ClusterState: map[string]interface{}{
			"node_count": 3,
		},
	}

	if context.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got %s", context.ClusterName)
	}

	if context.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %s", context.Namespace)
	}

	if len(context.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(context.Metrics))
	}

	if context.Metrics[0].Name != "cpu_usage" {
		t.Errorf("expected metric name 'cpu_usage', got %s", context.Metrics[0].Name)
	}

	if len(context.RelatedChecks) != 1 {
		t.Errorf("expected 1 related check, got %d", len(context.RelatedChecks))
	}

	if context.ClusterState["node_count"] != 3 {
		t.Errorf("expected node count 3, got %v", context.ClusterState["node_count"])
	}
}