package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewAssistant(t *testing.T) {
	client := NewClient(Config{TestMode: true})
	assistant := NewAssistant(client)
	
	if assistant == nil {
		t.Fatal("expected assistant to be created")
	}
	
	if assistant.client != client {
		t.Error("expected assistant to use provided client")
	}
	
	if assistant.analyzer == nil {
		t.Error("expected predictive analyzer to be initialized")
	}
	
	if assistant.knowledge == nil {
		t.Error("expected knowledge base to be initialized")
	}
	
	// Test knowledge base initialization
	if assistant.knowledge.clusterContext == nil {
		t.Error("expected cluster context to be initialized")
	}
	
	if assistant.knowledge.solutions == nil {
		t.Error("expected solutions to be initialized")
	}
	
	if assistant.knowledge.patterns == nil {
		t.Error("expected patterns to be initialized")
	}
}

func TestQueryTypeDetection(t *testing.T) {
	assistant := &Assistant{}
	
	tests := []struct {
		name     string
		query    string
		expected func(string) bool
	}{
		{
			name:     "performance query - slow",
			query:    "Why is my cluster slow?",
			expected: assistant.isPerformanceQuery,
		},
		{
			name:     "performance query - latency",
			query:    "High latency in my services",
			expected: assistant.isPerformanceQuery,
		},
		{
			name:     "performance query - response time",
			query:    "Response time is too high",
			expected: assistant.isPerformanceQuery,
		},
		{
			name:     "troubleshooting query - error",
			query:    "I'm getting an error in my pods",
			expected: assistant.isTroubleshootingQuery,
		},
		{
			name:     "troubleshooting query - failing",
			query:    "My deployment is failing",
			expected: assistant.isTroubleshootingQuery,
		},
		{
			name:     "troubleshooting query - not working",
			query:    "Service not working properly",
			expected: assistant.isTroubleshootingQuery,
		},
		{
			name:     "optimization query - optimize",
			query:    "How can I optimize my cluster?",
			expected: assistant.isOptimizationQuery,
		},
		{
			name:     "optimization query - cost",
			query:    "Reduce cost of my infrastructure",
			expected: assistant.isOptimizationQuery,
		},
		{
			name:     "optimization query - scale",
			query:    "Scale my application efficiently",
			expected: assistant.isOptimizationQuery,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expected(tt.query) {
				t.Errorf("expected query %q to be detected as matching type", tt.query)
			}
		})
	}
}

func TestQueryTypeDetectionNegative(t *testing.T) {
	assistant := &Assistant{}
	
	tests := []struct {
		name  string
		query string
		check func(string) bool
	}{
		{
			name:  "general query not performance",
			query: "What is the status of my cluster?",
			check: assistant.isPerformanceQuery,
		},
		{
			name:  "general query not troubleshooting",
			query: "List all pods in default namespace",
			check: assistant.isTroubleshootingQuery,
		},
		{
			name:  "general query not optimization",
			query: "Show me cluster information",
			check: assistant.isOptimizationQuery,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.check(tt.query) {
				t.Errorf("expected query %q to NOT match this type", tt.query)
			}
		})
	}
}

func TestCategorizeQuery(t *testing.T) {
	assistant := &Assistant{}
	
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "performance category",
			query:    "My cluster is slow",
			expected: "performance",
		},
		{
			name:     "troubleshooting category",
			query:    "I have an error",
			expected: "troubleshooting",
		},
		{
			name:     "optimization category",
			query:    "How to optimize resources",
			expected: "optimization",
		},
		{
			name:     "general category",
			query:    "What is Kubernetes?",
			expected: "general",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assistant.categorizeQuery(tt.query)
			if result != tt.expected {
				t.Errorf("expected category %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractActions(t *testing.T) {
	assistant := &Assistant{}
	
	response := &AnalysisResponse{
		Actions: []SuggestedAction{
			{Description: "Restart pod"},
			{Description: "Scale deployment"},
			{Description: "Check logs"},
		},
	}
	
	actions := assistant.extractActions(response)
	
	expected := []string{"Restart pod", "Scale deployment", "Check logs"}
	
	if len(actions) != len(expected) {
		t.Fatalf("expected %d actions, got %d", len(expected), len(actions))
	}
	
	for i, action := range actions {
		if action != expected[i] {
			t.Errorf("expected action %q, got %q", expected[i], action)
		}
	}
}

func TestExtractCommands(t *testing.T) {
	assistant := &Assistant{}
	
	response := &AnalysisResponse{
		Actions: []SuggestedAction{
			{Command: "kubectl get pods"},
			{Command: "kubectl logs pod-name"},
			{Description: "Manual action"}, // No command
		},
	}
	
	commands := assistant.extractCommands(response)
	
	expected := []string{"kubectl get pods", "kubectl logs pod-name"}
	
	if len(commands) != len(expected) {
		t.Fatalf("expected %d commands, got %d", len(expected), len(commands))
	}
	
	for i, command := range commands {
		if command != expected[i] {
			t.Errorf("expected command %q, got %q", expected[i], command)
		}
	}
}

func TestExtractReferences(t *testing.T) {
	assistant := &Assistant{}
	
	response := &AnalysisResponse{
		Recommendations: []Recommendation{
			{References: []string{"https://kubernetes.io/docs/concepts/"}},
			{References: []string{"https://kubernetes.io/docs/tasks/", "https://example.com"}},
		},
	}
	
	references := assistant.extractReferences(response)
	
	expected := []string{
		"https://kubernetes.io/docs/concepts/",
		"https://kubernetes.io/docs/tasks/",
		"https://example.com",
	}
	
	if len(references) != len(expected) {
		t.Fatalf("expected %d references, got %d", len(expected), len(references))
	}
	
	for i, ref := range references {
		if ref != expected[i] {
			t.Errorf("expected reference %q, got %q", expected[i], ref)
		}
	}
}

func TestGenerateFollowupQuestions(t *testing.T) {
	assistant := &Assistant{}
	
	tests := []struct {
		name     string
		original string
		response *AnalysisResponse
		contains []string
	}{
		{
			name:     "critical severity",
			original: "What's wrong?",
			response: &AnalysisResponse{
				Severity: SeverityCritical,
			},
			contains: []string{"remediation plan"},
		},
		{
			name:     "has actions",
			original: "Help me fix this",
			response: &AnalysisResponse{
				Actions: []SuggestedAction{
					{Description: "Do something"},
				},
			},
			contains: []string{"execute these commands"},
		},
		{
			name:     "low confidence",
			original: "Diagnose this issue",
			response: &AnalysisResponse{
				Confidence: 0.5,
			},
			contains: []string{"more details"},
		},
		{
			name:     "high confidence, no actions",
			original: "Status check",
			response: &AnalysisResponse{
				Confidence: 0.9,
				Severity:   SeverityLow,
			},
			contains: []string{}, // No specific follow-ups expected
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			followups := assistant.generateFollowupQuestions(tt.original, tt.response)
			
			for _, expected := range tt.contains {
				found := false
				for _, followup := range followups {
					if strings.Contains(strings.ToLower(followup), strings.ToLower(expected)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected followup to contain %q, got %v", expected, followups)
				}
			}
		})
	}
}

func TestFormatResponse(t *testing.T) {
	assistant := &Assistant{}
	
	response := &AnalysisResponse{
		Summary:    "Test summary",
		Confidence: 0.85,
		Actions: []SuggestedAction{
			{Description: "Action 1", Command: "cmd1"},
			{Description: "Action 2", Command: "cmd2"},
		},
		Recommendations: []Recommendation{
			{References: []string{"ref1", "ref2"}},
		},
	}
	
	queryResponse := assistant.formatResponse(response)
	
	if queryResponse.Answer != "Test summary" {
		t.Errorf("expected answer %q, got %q", "Test summary", queryResponse.Answer)
	}
	
	if queryResponse.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", queryResponse.Confidence)
	}
	
	expectedActions := []string{"Action 1", "Action 2"}
	if len(queryResponse.Actions) != len(expectedActions) {
		t.Fatalf("expected %d actions, got %d", len(expectedActions), len(queryResponse.Actions))
	}
	
	expectedCommands := []string{"cmd1", "cmd2"}
	if len(queryResponse.Commands) != len(expectedCommands) {
		t.Fatalf("expected %d commands, got %d", len(expectedCommands), len(queryResponse.Commands))
	}
	
	expectedReferences := []string{"ref1", "ref2"}
	if len(queryResponse.References) != len(expectedReferences) {
		t.Fatalf("expected %d references, got %d", len(expectedReferences), len(queryResponse.References))
	}
}

func TestExtractMetrics(t *testing.T) {
	assistant := &Assistant{}
	
	health := &ClusterHealth{
		Checks: []CheckResult{
			{
				Name: "check1",
				Metrics: []Metric{
					{Name: "cpu", Value: 80.5},
					{Name: "memory", Value: 75.0},
				},
			},
			{
				Name: "check2",
				Metrics: []Metric{
					{Name: "disk", Value: 60.0},
				},
			},
		},
	}
	
	metrics := assistant.extractMetrics(health)
	
	expected := 3 // 2 from check1 + 1 from check2
	if len(metrics) != expected {
		t.Fatalf("expected %d metrics, got %d", expected, len(metrics))
	}
	
	// Verify metric names
	expectedNames := []string{"cpu", "memory", "disk"}
	for i, metric := range metrics {
		if metric.Name != expectedNames[i] {
			t.Errorf("expected metric name %q, got %q", expectedNames[i], metric.Name)
		}
	}
}

func TestLearnFromFeedback(t *testing.T) {
	client := NewClient(Config{TestMode: true})
	assistant := NewAssistant(client)
	
	query := "How to fix pod crashes?"
	response := &QueryResponse{
		Answer:     "Restart the pods",
		Confidence: 0.9,
		Commands:   []string{"kubectl delete pod failing-pod"},
	}
	
	// Test helpful feedback
	assistant.LearnFromFeedback(context.Background(), query, response, true)
	
	// Verify solution was stored
	category := assistant.categorizeQuery(query)
	solutions := assistant.knowledge.solutions[category]
	
	if len(solutions) != 1 {
		t.Fatalf("expected 1 solution to be stored, got %d", len(solutions))
	}
	
	solution := solutions[0]
	if solution.Problem != query {
		t.Errorf("expected problem %q, got %q", query, solution.Problem)
	}
	
	if solution.Solution != response.Answer {
		t.Errorf("expected solution %q, got %q", response.Answer, solution.Solution)
	}
	
	if solution.Confidence != response.Confidence {
		t.Errorf("expected confidence %f, got %f", response.Confidence, solution.Confidence)
	}
	
	// Test unhelpful feedback (should not store)
	initialCount := len(assistant.knowledge.solutions[category])
	assistant.LearnFromFeedback(context.Background(), "Another query", response, false)
	
	if len(assistant.knowledge.solutions[category]) != initialCount {
		t.Error("expected no new solutions to be stored for unhelpful feedback")
	}
}

func TestUpdateContext(t *testing.T) {
	client := NewClient(Config{TestMode: true})
	assistant := NewAssistant(client)
	
	key := "cluster_version"
	value := "1.21.0"
	
	assistant.UpdateContext(key, value)
	
	if assistant.knowledge.clusterContext[key] != value {
		t.Errorf("expected context[%q] = %q, got %v", key, value, assistant.knowledge.clusterContext[key])
	}
	
	// Test updating existing key
	newValue := "1.22.0"
	assistant.UpdateContext(key, newValue)
	
	if assistant.knowledge.clusterContext[key] != newValue {
		t.Errorf("expected updated context[%q] = %q, got %v", key, newValue, assistant.knowledge.clusterContext[key])
	}
}

func TestSolutionStructure(t *testing.T) {
	solution := Solution{
		Problem:     "Test problem",
		Solution:    "Test solution",
		Commands:    []string{"cmd1", "cmd2"},
		Confidence:  0.95,
		LastApplied: time.Now(),
	}
	
	if solution.Problem != "Test problem" {
		t.Errorf("expected problem %q, got %q", "Test problem", solution.Problem)
	}
	
	if solution.Solution != "Test solution" {
		t.Errorf("expected solution %q, got %q", "Test solution", solution.Solution)
	}
	
	if len(solution.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(solution.Commands))
	}
	
	if solution.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", solution.Confidence)
	}
}

func TestPatternStructure(t *testing.T) {
	pattern := Pattern{
		Name:        "test-pattern",
		Description: "Test pattern description",
		Indicators:  []string{"indicator1", "indicator2"},
		LastSeen:    time.Now(),
	}
	
	if pattern.Name != "test-pattern" {
		t.Errorf("expected name %q, got %q", "test-pattern", pattern.Name)
	}
	
	if pattern.Description != "Test pattern description" {
		t.Errorf("expected description %q, got %q", "Test pattern description", pattern.Description)
	}
	
	if len(pattern.Indicators) != 2 {
		t.Errorf("expected 2 indicators, got %d", len(pattern.Indicators))
	}
}

func TestQueryResponseStructure(t *testing.T) {
	response := QueryResponse{
		Answer:     "Test answer",
		Confidence: 0.88,
		Actions:    []string{"action1", "action2"},
		Commands:   []string{"cmd1", "cmd2"},
		References: []string{"ref1", "ref2"},
		Followup:   []string{"followup1", "followup2"},
	}
	
	if response.Answer != "Test answer" {
		t.Errorf("expected answer %q, got %q", "Test answer", response.Answer)
	}
	
	if response.Confidence != 0.88 {
		t.Errorf("expected confidence 0.88, got %f", response.Confidence)
	}
	
	if len(response.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(response.Actions))
	}
	
	if len(response.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(response.Commands))
	}
	
	if len(response.References) != 2 {
		t.Errorf("expected 2 references, got %d", len(response.References))
	}
	
	if len(response.Followup) != 2 {
		t.Errorf("expected 2 followup questions, got %d", len(response.Followup))
	}
}

// Test specialized query handlers with mock data
func TestHandlerStructures(t *testing.T) {
	client := NewClient(Config{TestMode: true})
	assistant := NewAssistant(client)
	
	health := &ClusterHealth{
		Status: HealthStatusHealthy,
		Score: HealthScore{
			Weighted: 85.0,
		},
		Checks: []CheckResult{
			{
				Name:   "test-check",
				Status: HealthStatusHealthy,
			},
		},
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	// Test that handlers don't panic and structure requests properly
	testCases := []struct {
		name string
		fn   func() error
	}{
		{
			name: "performance query handler",
			fn: func() error {
				_, err := assistant.handlePerformanceQuery(ctx, "Why is it slow?", health)
				return err // Expected to fail due to missing claude CLI, but shouldn't panic
			},
		},
		{
			name: "troubleshooting query handler",
			fn: func() error {
				_, err := assistant.handleTroubleshootingQuery(ctx, "What's broken?", health)
				return err // Expected to fail due to missing claude CLI, but shouldn't panic
			},
		},
		{
			name: "optimization query handler",
			fn: func() error {
				_, err := assistant.handleOptimizationQuery(ctx, "How to optimize?", health)
				return err // Expected to fail due to missing claude CLI, but shouldn't panic
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We expect errors due to missing claude CLI, but no panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("handler %s panicked: %v", tc.name, r)
				}
			}()
			
			err := tc.fn()
			if err == nil {
				t.Errorf("expected error from %s due to missing claude CLI", tc.name)
			}
		})
	}
}