package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// Assistant provides natural language interface to cluster insights
type Assistant struct {
	client    *Client
	analyzer  *PredictiveAnalyzer
	knowledge *KnowledgeBase
}

// KnowledgeBase stores cluster-specific knowledge
type KnowledgeBase struct {
	clusterContext map[string]interface{}
	solutions      map[string][]Solution
	patterns       []Pattern
}

// Solution represents a solution to a known problem
type Solution struct {
	Problem     string
	Solution    string
	Commands    []string
	Confidence  float64
	LastApplied time.Time
}

// Pattern represents a recognized pattern in the cluster
type Pattern struct {
	Name        string
	Description string
	Indicators  []string
	LastSeen    time.Time
}

// QueryResponse represents assistant's response to a query
type QueryResponse struct {
	Answer     string   `json:"answer"`
	Confidence float64  `json:"confidence"`
	Actions    []string `json:"suggested_actions,omitempty"`
	Commands   []string `json:"commands,omitempty"`
	References []string `json:"references,omitempty"`
	Followup   []string `json:"followup_questions,omitempty"`
}

// NewAssistant creates a new AI assistant
func NewAssistant(client *Client) *Assistant {
	return &Assistant{
		client:   client,
		analyzer: NewPredictiveAnalyzer(client),
		knowledge: &KnowledgeBase{
			clusterContext: make(map[string]interface{}),
			solutions:      make(map[string][]Solution),
			patterns:       []Pattern{},
		},
	}
}

// Query processes natural language queries about the cluster
func (a *Assistant) Query(ctx context.Context, question string, clusterHealth *ClusterHealth) (*QueryResponse, error) {
	klog.V(2).Infof("Processing natural language query: %s", question)

	request := AnalysisRequest{
		Type:    AnalysisTypeSummary,
		Context: "Natural language query from user",
		Data: map[string]interface{}{
			"user_question":   question,
			"cluster_context": a.knowledge.clusterContext,
			"known_patterns":  a.knowledge.patterns,
		},
		ClusterInfo: clusterHealth,
		Timestamp:   time.Now(),
	}

	// Special handling for common query types
	if a.isPerformanceQuery(question) {
		return a.handlePerformanceQuery(ctx, question, clusterHealth)
	}

	if a.isTroubleshootingQuery(question) {
		return a.handleTroubleshootingQuery(ctx, question, clusterHealth)
	}

	if a.isOptimizationQuery(question) {
		return a.handleOptimizationQuery(ctx, question, clusterHealth)
	}

	// General query handling
	response, err := a.client.Analyze(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("assistant query failed: %w", err)
	}

	return &QueryResponse{
		Answer:     response.Summary,
		Confidence: response.Confidence,
		Actions:    a.extractActions(response),
		Commands:   a.extractCommands(response),
		References: a.extractReferences(response),
		Followup:   a.generateFollowupQuestions(question, response),
	}, nil
}

// LearnFromFeedback updates the knowledge base based on user feedback
func (a *Assistant) LearnFromFeedback(ctx context.Context, query string, response *QueryResponse, helpful bool) {
	if helpful {
		// Store successful solutions
		solution := Solution{
			Problem:     query,
			Solution:    response.Answer,
			Commands:    response.Commands,
			Confidence:  response.Confidence,
			LastApplied: time.Now(),
		}

		key := a.categorizeQuery(query)
		a.knowledge.solutions[key] = append(a.knowledge.solutions[key], solution)

		klog.V(2).Infof("Learned new solution for category: %s", key)
	}
}

// UpdateContext updates the cluster context for better responses
func (a *Assistant) UpdateContext(key string, value interface{}) {
	a.knowledge.clusterContext[key] = value
}

// handlePerformanceQuery handles performance-related questions
func (a *Assistant) handlePerformanceQuery(ctx context.Context, question string, health *ClusterHealth) (*QueryResponse, error) {
	prompt := fmt.Sprintf(`Performance Analysis Query:
Question: %s

Current Cluster State:
- Overall Health: %s
- Health Score: %.2f%%
- Active Checks: %d

Analyze performance issues and provide:
1. Direct answer to the question
2. Performance metrics interpretation
3. Specific optimization recommendations
4. kubectl commands to investigate further

Focus on actionable insights.`,
		question, health.Status, health.Score.Weighted, len(health.Checks))

	request := AnalysisRequest{
		Type:        AnalysisTypeOptimization,
		Context:     prompt,
		ClusterInfo: health,
		Timestamp:   time.Now(),
	}

	response, err := a.client.Analyze(ctx, request)
	if err != nil {
		return nil, err
	}

	return a.formatResponse(response), nil
}

// handleTroubleshootingQuery handles troubleshooting questions
func (a *Assistant) handleTroubleshootingQuery(ctx context.Context, question string, health *ClusterHealth) (*QueryResponse, error) {
	// Find relevant failing checks
	failingChecks := []CheckResult{}
	for _, check := range health.Checks {
		if check.Status != HealthStatusHealthy {
			failingChecks = append(failingChecks, check)
		}
	}

	prompt := fmt.Sprintf(`Troubleshooting Query:
Question: %s

Failing Health Checks:
%+v

Known Patterns:
%+v

Provide:
1. Root cause analysis
2. Step-by-step troubleshooting guide
3. Specific commands to run
4. Prevention measures`,
		question, failingChecks, a.knowledge.patterns)

	request := AnalysisRequest{
		Type:        AnalysisTypeRootCause,
		Context:     prompt,
		ClusterInfo: health,
		Timestamp:   time.Now(),
	}

	response, err := a.client.Analyze(ctx, request)
	if err != nil {
		return nil, err
	}

	return a.formatResponse(response), nil
}

// handleOptimizationQuery handles optimization questions
func (a *Assistant) handleOptimizationQuery(ctx context.Context, question string, health *ClusterHealth) (*QueryResponse, error) {
	// Analyze resource usage patterns
	predictions, _ := a.analyzer.AnalyzeTrends(ctx, a.extractMetrics(health))

	prompt := fmt.Sprintf(`Optimization Query:
Question: %s

Predictions:
%+v

Provide:
1. Optimization opportunities
2. Resource rightsizing recommendations
3. Cost-saving suggestions
4. Implementation commands`,
		question, predictions)

	request := AnalysisRequest{
		Type:        AnalysisTypeOptimization,
		Context:     prompt,
		ClusterInfo: health,
		Data: map[string]interface{}{
			"predictions": predictions,
		},
		Timestamp: time.Now(),
	}

	response, err := a.client.Analyze(ctx, request)
	if err != nil {
		return nil, err
	}

	return a.formatResponse(response), nil
}

// Query type detection helpers
func (a *Assistant) isPerformanceQuery(q string) bool {
	keywords := []string{"slow", "performance", "latency", "speed", "fast", "response time"}
	q = strings.ToLower(q)
	for _, kw := range keywords {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func (a *Assistant) isTroubleshootingQuery(q string) bool {
	keywords := []string{"error", "fail", "crash", "not working", "issue", "problem", "debug", "fix"}
	q = strings.ToLower(q)
	for _, kw := range keywords {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func (a *Assistant) isOptimizationQuery(q string) bool {
	keywords := []string{"optimize", "improve", "reduce", "cost", "efficient", "save", "scale"}
	q = strings.ToLower(q)
	for _, kw := range keywords {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

// Helper methods
// enhanceWithContext is unused but kept for future implementation
// func (a *Assistant) enhanceWithContext(question string, health *ClusterHealth) string {
// 	return fmt.Sprintf(`User Question: %s
// Cluster Status: %s (Score: %.2f)
// Context: %+v`,
// 		question, health.Status, health.Score.Weighted, a.knowledge.clusterContext)
// }

func (a *Assistant) categorizeQuery(query string) string {
	if a.isPerformanceQuery(query) {
		return "performance"
	}
	if a.isTroubleshootingQuery(query) {
		return "troubleshooting"
	}
	if a.isOptimizationQuery(query) {
		return "optimization"
	}
	return "general"
}

func (a *Assistant) extractActions(response *AnalysisResponse) []string {
	actions := []string{}
	for _, action := range response.Actions {
		actions = append(actions, action.Description)
	}
	return actions
}

func (a *Assistant) extractCommands(response *AnalysisResponse) []string {
	commands := []string{}
	for _, action := range response.Actions {
		if action.Command != "" {
			commands = append(commands, action.Command)
		}
	}
	return commands
}

func (a *Assistant) extractReferences(response *AnalysisResponse) []string {
	refs := []string{}
	for _, rec := range response.Recommendations {
		refs = append(refs, rec.References...)
	}
	return refs
}

func (a *Assistant) generateFollowupQuestions(original string, response *AnalysisResponse) []string {
	// Generate contextual follow-up questions
	followups := []string{}

	if response.Severity == SeverityCritical {
		followups = append(followups, "Would you like me to generate a remediation plan?")
	}

	if len(response.Actions) > 0 {
		followups = append(followups, "Should I execute these commands for you?")
	}

	if response.Confidence < 0.7 {
		followups = append(followups, "Can you provide more details about the issue?")
	}

	return followups
}

func (a *Assistant) formatResponse(response *AnalysisResponse) *QueryResponse {
	return &QueryResponse{
		Answer:     response.Summary,
		Confidence: response.Confidence,
		Actions:    a.extractActions(response),
		Commands:   a.extractCommands(response),
		References: a.extractReferences(response),
		Followup:   []string{},
	}
}

func (a *Assistant) extractMetrics(health *ClusterHealth) []Metric {
	metrics := []Metric{}
	for _, check := range health.Checks {
		metrics = append(metrics, check.Metrics...)
	}
	return metrics
}
