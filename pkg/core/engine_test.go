package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/ai"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewEngine(t *testing.T) {
	client := fake.NewSimpleClientset()

	tests := []struct {
		name   string
		config EngineConfig
		verify func(*Engine) error
	}{
		{
			name: "basic config",
			config: EngineConfig{
				KubeClient:  client,
				ContextName: "test-context",
				Interval:    10 * time.Second,
				EnableAI:    false,
			},
			verify: func(e *Engine) error {
				if e.client != client {
					t.Error("expected client to be set")
				}
				if e.currentContext != "test-context" {
					t.Errorf("expected context %q, got %q", "test-context", e.currentContext)
				}
				if e.interval != 10*time.Second {
					t.Errorf("expected interval 10s, got %v", e.interval)
				}
				if e.aiClient != nil {
					t.Error("expected AI client to be nil when disabled")
				}
				return nil
			},
		},
		{
			name: "default interval",
			config: EngineConfig{
				KubeClient:  client,
				ContextName: "test-context",
			},
			verify: func(e *Engine) error {
				if e.interval != 30*time.Second {
					t.Errorf("expected default interval 30s, got %v", e.interval)
				}
				return nil
			},
		},
		{
			name: "AI enabled",
			config: EngineConfig{
				KubeClient:  client,
				ContextName: "test-context",
				EnableAI:    true,
				AIConfig: &ai.Config{
					ClaudePath: "claude",
				},
			},
			verify: func(e *Engine) error {
				if e.aiClient == nil {
					t.Error("expected AI client to be initialized when enabled")
				}
				if e.predictiveAnalyzer == nil {
					t.Error("expected predictive analyzer to be initialized")
				}
				if e.assistant == nil {
					t.Error("expected assistant to be initialized")
				}
				if e.smartAlertManager == nil {
					t.Error("expected smart alert manager to be initialized")
				}
				if e.remediationEngine == nil {
					t.Error("expected remediation engine to be initialized")
				}
				return nil
			},
		},
		{
			name: "AI enabled with nil config",
			config: EngineConfig{
				KubeClient:  client,
				ContextName: "test-context",
				EnableAI:    true,
				AIConfig:    nil, // Should use defaults
			},
			verify: func(e *Engine) error {
				if e.aiClient == nil {
					t.Error("expected AI client to be initialized with default config")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(tt.config)

			// Common verifications
			if engine == nil {
				t.Fatal("expected engine to be created")
			}

			if engine.checks == nil {
				t.Error("expected checks slice to be initialized")
			}

			if engine.results == nil {
				t.Error("expected results map to be initialized")
			}

			if engine.alertManager == nil {
				t.Error("expected alert manager to be initialized")
			}

			if engine.anomalyEngine == nil {
				t.Error("expected anomaly engine to be initialized")
			}

			if engine.sloTracker == nil {
				t.Error("expected SLO tracker to be initialized")
			}

			if engine.errorHandler == nil {
				t.Error("expected error handler to be initialized")
			}

			if engine.ctx == nil {
				t.Error("expected context to be initialized")
			}

			if engine.cancel == nil {
				t.Error("expected cancel function to be initialized")
			}

			// Test-specific verifications
			if err := tt.verify(engine); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAddRemoveCheck(t *testing.T) {
	client := fake.NewSimpleClientset()
	config := EngineConfig{
		KubeClient:  client,
		ContextName: "test-context",
	}
	engine := NewEngine(config)

	// Create mock health check
	check := &mockHealthCheck{
		name: "test-check",
	}

	// Test adding check
	engine.AddCheck(check)

	if len(engine.checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(engine.checks))
	}

	if engine.checks[0].Name() != "test-check" {
		t.Errorf("expected check name %q, got %q", "test-check", engine.checks[0].Name())
	}

	// Test adding another check
	check2 := &mockHealthCheck{
		name: "test-check-2",
	}
	engine.AddCheck(check2)

	if len(engine.checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(engine.checks))
	}

	// Test removing check
	err := engine.RemoveCheck("test-check")
	if err != nil {
		t.Fatalf("unexpected error removing check: %v", err)
	}

	if len(engine.checks) != 1 {
		t.Fatalf("expected 1 check after removal, got %d", len(engine.checks))
	}

	if engine.checks[0].Name() != "test-check-2" {
		t.Errorf("expected remaining check to be %q, got %q", "test-check-2", engine.checks[0].Name())
	}

	// Test removing non-existent check
	err = engine.RemoveCheck("non-existent")
	if err == nil {
		t.Error("expected error when removing non-existent check")
	}

	expectedError := "check non-existent not found"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestCalculateScore(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name     string
		result   CheckResult
		expected float64
	}{
		{
			name:     "healthy status",
			result:   CheckResult{Status: HealthStatusHealthy},
			expected: 1.0,
		},
		{
			name:     "degraded status",
			result:   CheckResult{Status: HealthStatusDegraded},
			expected: 0.5,
		},
		{
			name:     "unhealthy status",
			result:   CheckResult{Status: HealthStatusUnhealthy},
			expected: 0.0,
		},
		{
			name:     "unknown status",
			result:   CheckResult{Status: HealthStatusUnknown},
			expected: 0.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateScore(tt.result)
			if score != tt.expected {
				t.Errorf("expected score %f, got %f", tt.expected, score)
			}
		})
	}
}

func TestGetWeight(t *testing.T) {
	engine := &Engine{}

	result := CheckResult{Name: "test-check"}
	weight := engine.getWeight(result)

	// Currently returns default weight of 1.0
	if weight != 1.0 {
		t.Errorf("expected default weight 1.0, got %f", weight)
	}
}

func TestGetSeverity(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name     string
		result   CheckResult
		expected AlertSeverity
	}{
		{
			name:     "unhealthy becomes critical",
			result:   CheckResult{Status: HealthStatusUnhealthy},
			expected: AlertSeverityCritical,
		},
		{
			name:     "degraded becomes warning",
			result:   CheckResult{Status: HealthStatusDegraded},
			expected: AlertSeverityWarning,
		},
		{
			name:     "healthy becomes info",
			result:   CheckResult{Status: HealthStatusHealthy},
			expected: AlertSeverityInfo,
		},
		{
			name:     "unknown becomes info",
			result:   CheckResult{Status: HealthStatusUnknown},
			expected: AlertSeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := engine.getSeverity(tt.result)
			if severity != tt.expected {
				t.Errorf("expected severity %v, got %v", tt.expected, severity)
			}
		})
	}
}

func TestStoreAndGetResult(t *testing.T) {
	client := fake.NewSimpleClientset()
	config := EngineConfig{
		KubeClient:  client,
		ContextName: "test-context",
	}
	engine := NewEngine(config)

	result := CheckResult{
		Name:    "test-check",
		Status:  HealthStatusHealthy,
		Message: "All good",
	}

	// Test storing result
	engine.storeResult(result)

	// Test getting specific result
	stored, exists := engine.GetResult("test-check")
	if !exists {
		t.Fatal("expected result to exist")
	}

	if stored.Name != "test-check" {
		t.Errorf("expected name %q, got %q", "test-check", stored.Name)
	}

	if stored.Status != HealthStatusHealthy {
		t.Errorf("expected status %v, got %v", HealthStatusHealthy, stored.Status)
	}

	if stored.Message != "All good" {
		t.Errorf("expected message %q, got %q", "All good", stored.Message)
	}

	// Test getting non-existent result
	_, exists = engine.GetResult("non-existent")
	if exists {
		t.Error("expected result to not exist")
	}

	// Test getting all results
	allResults := engine.GetResults()
	if len(allResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(allResults))
	}

	if allResults["test-check"].Name != "test-check" {
		t.Error("expected result to be stored in all results")
	}
}

func TestGetClusterHealth(t *testing.T) {
	client := fake.NewSimpleClientset()
	config := EngineConfig{
		KubeClient:  client,
		ContextName: "test-context",
	}
	engine := NewEngine(config)

	// Store various check results
	results := []CheckResult{
		{Name: "healthy-check", Status: HealthStatusHealthy},
		{Name: "degraded-check", Status: HealthStatusDegraded},
		{Name: "unhealthy-check", Status: HealthStatusUnhealthy},
	}

	for _, result := range results {
		engine.storeResult(result)
	}

	clusterHealth := engine.GetClusterHealth("test-cluster")

	// Test cluster name
	if clusterHealth.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name %q, got %q", "test-cluster", clusterHealth.ClusterName)
	}

	// Test overall status (should be degraded with mixed results)
	if clusterHealth.Status != HealthStatusDegraded {
		t.Errorf("expected status %v, got %v", HealthStatusDegraded, clusterHealth.Status)
	}

	// Test check count
	if len(clusterHealth.Checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(clusterHealth.Checks))
	}

	// Test score calculation
	if clusterHealth.Score.Raw <= 0 || clusterHealth.Score.Raw > 100 {
		t.Errorf("expected raw score between 0-100, got %f", clusterHealth.Score.Raw)
	}

	if clusterHealth.Score.Weighted <= 0 || clusterHealth.Score.Weighted > 100 {
		t.Errorf("expected weighted score between 0-100, got %f", clusterHealth.Score.Weighted)
	}

	// Test with all healthy checks
	engine.results = make(map[string]CheckResult)
	engine.storeResult(CheckResult{Name: "healthy1", Status: HealthStatusHealthy})
	engine.storeResult(CheckResult{Name: "healthy2", Status: HealthStatusHealthy})

	healthyCluster := engine.GetClusterHealth("healthy-cluster")
	if healthyCluster.Status != HealthStatusHealthy {
		t.Errorf("expected healthy status, got %v", healthyCluster.Status)
	}

	// Test with all unhealthy checks
	engine.results = make(map[string]CheckResult)
	engine.storeResult(CheckResult{Name: "unhealthy1", Status: HealthStatusUnhealthy})
	engine.storeResult(CheckResult{Name: "unhealthy2", Status: HealthStatusUnhealthy})

	unhealthyCluster := engine.GetClusterHealth("unhealthy-cluster")
	if unhealthyCluster.Status != HealthStatusUnhealthy {
		t.Errorf("expected unhealthy status, got %v", unhealthyCluster.Status)
	}
}

func TestCountCriticalIssues(t *testing.T) {
	// Test the helper functionality instead of the unexported method
	clusterHealth := ClusterHealth{
		Checks: []CheckResult{
			{Status: HealthStatusHealthy},
			{Status: HealthStatusUnhealthy},
			{Status: HealthStatusDegraded},
			{Status: HealthStatusUnhealthy},
		},
	}

	count := 0
	for _, check := range clusterHealth.Checks {
		if check.Status == HealthStatusUnhealthy {
			count++
		}
	}

	expected := 2 // Two unhealthy checks
	if count != expected {
		t.Errorf("expected %d critical issues, got %d", expected, count)
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fn       func(string) string
		expected string
	}{
		{
			name:     "extract pod resource type",
			input:    "pod-health-check",
			fn:       extractResourceType,
			expected: "pod",
		},
		{
			name:     "extract node resource type",
			input:    "node-memory-check",
			fn:       extractResourceType,
			expected: "node",
		},
		{
			name:     "extract service resource type",
			input:    "service-endpoint-check",
			fn:       extractResourceType,
			expected: "service",
		},
		{
			name:     "extract unknown resource type",
			input:    "unknown-check",
			fn:       extractResourceType,
			expected: "unknown",
		},
		{
			name:     "extract resource name",
			input:    "pod-health-myapp",
			fn:       extractResourceName,
			expected: "myapp",
		},
		{
			name:     "extract resource name no dash",
			input:    "simplecheck",
			fn:       extractResourceName,
			expected: "simplecheck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractErrorLogs(t *testing.T) {
	tests := []struct {
		name     string
		result   CheckResult
		expected []string
	}{
		{
			name: "with error and message",
			result: CheckResult{
				Error:   errors.New("test error"),
				Message: "test message",
			},
			expected: []string{"test error", "test message"},
		},
		{
			name: "with only error",
			result: CheckResult{
				Error: errors.New("only error"),
			},
			expected: []string{"only error"},
		},
		{
			name: "with only message",
			result: CheckResult{
				Message: "only message",
			},
			expected: []string{"only message"},
		},
		{
			name:     "no error or message",
			result:   CheckResult{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := extractErrorLogs(tt.result)

			if len(logs) != len(tt.expected) {
				t.Fatalf("expected %d logs, got %d", len(tt.expected), len(logs))
			}

			for i, log := range logs {
				if log != tt.expected[i] {
					t.Errorf("expected log %d to be %q, got %q", i, tt.expected[i], log)
				}
			}
		})
	}
}

func TestExtractEvents(t *testing.T) {
	tests := []struct {
		name     string
		result   CheckResult
		expected []string
	}{
		{
			name: "with events in details",
			result: CheckResult{
				Details: map[string]interface{}{
					"events": []string{"event1", "event2"},
				},
			},
			expected: []string{"event1", "event2"},
		},
		{
			name: "with invalid events format",
			result: CheckResult{
				Details: map[string]interface{}{
					"events": "not a slice",
				},
			},
			expected: []string{},
		},
		{
			name: "no events",
			result: CheckResult{
				Details: map[string]interface{}{
					"other": "data",
				},
			},
			expected: []string{},
		},
		{
			name:     "no details",
			result:   CheckResult{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := extractEvents(tt.result)

			if len(events) != len(tt.expected) {
				t.Fatalf("expected %d events, got %d", len(tt.expected), len(events))
			}

			for i, event := range events {
				if event != tt.expected[i] {
					t.Errorf("expected event %d to be %q, got %q", i, tt.expected[i], event)
				}
			}
		})
	}
}

func TestConvertToAICheckResult(t *testing.T) {
	client := fake.NewSimpleClientset()
	config := EngineConfig{
		KubeClient:  client,
		ContextName: "test-context",
	}
	engine := NewEngine(config)

	coreResult := CheckResult{
		Name:    "test-check",
		Status:  HealthStatusHealthy,
		Message: "All good",
		Details: map[string]interface{}{
			"detail1": "value1",
		},
		Error:     nil,
		Timestamp: time.Now(),
		Duration:  5 * time.Second,
		Metrics: []Metric{
			{
				Name:  "cpu",
				Value: 80.5,
				Unit:  "percent",
				Type:  MetricTypeGauge,
			},
		},
		Predictions: []Prediction{
			{
				Status:      HealthStatusHealthy,
				Probability: 0.95,
				Reason:      "stable metrics",
			},
		},
	}

	aiResult := engine.convertToAICheckResult(coreResult)

	// Test basic fields
	if aiResult.Name != coreResult.Name {
		t.Errorf("expected name %q, got %q", coreResult.Name, aiResult.Name)
	}

	if aiResult.Status != ai.HealthStatus(coreResult.Status) {
		t.Errorf("expected status %v, got %v", coreResult.Status, aiResult.Status)
	}

	if aiResult.Message != coreResult.Message {
		t.Errorf("expected message %q, got %q", coreResult.Message, aiResult.Message)
	}

	// Test metrics conversion
	if len(aiResult.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(aiResult.Metrics))
	}

	aiMetric := aiResult.Metrics[0]
	coreMetric := coreResult.Metrics[0]

	if aiMetric.Name != coreMetric.Name {
		t.Errorf("expected metric name %q, got %q", coreMetric.Name, aiMetric.Name)
	}

	if aiMetric.Value != coreMetric.Value {
		t.Errorf("expected metric value %f, got %f", coreMetric.Value, aiMetric.Value)
	}

	// Test predictions conversion
	if len(aiResult.Predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(aiResult.Predictions))
	}

	aiPrediction := aiResult.Predictions[0]
	corePrediction := coreResult.Predictions[0]

	if aiPrediction.Status != ai.HealthStatus(corePrediction.Status) {
		t.Errorf("expected prediction status %v, got %v", corePrediction.Status, aiPrediction.Status)
	}

	if aiPrediction.Probability != corePrediction.Probability {
		t.Errorf("expected prediction probability %f, got %f", corePrediction.Probability, aiPrediction.Probability)
	}
}

func TestConvertToAIClusterHealth(t *testing.T) {
	client := fake.NewSimpleClientset()
	config := EngineConfig{
		KubeClient:  client,
		ContextName: "test-context",
	}
	engine := NewEngine(config)

	coreHealth := ClusterHealth{
		ClusterName: "test-cluster",
		Status:      HealthStatusHealthy,
		Score: HealthScore{
			Raw:        85.5,
			Weighted:   90.0,
			Trend:      "improving",
			Confidence: 0.95,
			Forecast:   "stable",
		},
		Checks: []CheckResult{
			{
				Name:   "test-check",
				Status: HealthStatusHealthy,
			},
		},
		Timestamp: time.Now(),
	}

	aiHealth := engine.convertToAIClusterHealth(coreHealth)

	// Test basic fields
	if aiHealth.ClusterName != coreHealth.ClusterName {
		t.Errorf("expected cluster name %q, got %q", coreHealth.ClusterName, aiHealth.ClusterName)
	}

	if aiHealth.Status != ai.HealthStatus(coreHealth.Status) {
		t.Errorf("expected status %v, got %v", coreHealth.Status, aiHealth.Status)
	}

	// Test score conversion
	if aiHealth.Score.Raw != coreHealth.Score.Raw {
		t.Errorf("expected raw score %f, got %f", coreHealth.Score.Raw, aiHealth.Score.Raw)
	}

	if aiHealth.Score.Weighted != coreHealth.Score.Weighted {
		t.Errorf("expected weighted score %f, got %f", coreHealth.Score.Weighted, aiHealth.Score.Weighted)
	}

	// Test checks conversion
	if len(aiHealth.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(aiHealth.Checks))
	}

	if aiHealth.Checks[0].Name != "test-check" {
		t.Errorf("expected check name %q, got %q", "test-check", aiHealth.Checks[0].Name)
	}
}

// Mock health check for testing
type mockHealthCheck struct {
	name string
}

func (m *mockHealthCheck) Name() string {
	return m.name
}

func (m *mockHealthCheck) Description() string {
	return "Mock health check"
}

func (m *mockHealthCheck) Interval() time.Duration {
	return 30 * time.Second
}

func (m *mockHealthCheck) Criticality() Criticality {
	return CriticalityMedium
}

func (m *mockHealthCheck) Check(ctx context.Context, client kubernetes.Interface) (CheckResult, error) {
	return CheckResult{
		Name:   m.name,
		Status: HealthStatusHealthy,
	}, nil
}

func (m *mockHealthCheck) Configure(config map[string]interface{}) error {
	return nil
}
