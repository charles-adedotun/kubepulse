package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kubepulse/kubepulse/pkg/ai"
	"github.com/kubepulse/kubepulse/pkg/alerts"
	"github.com/kubepulse/kubepulse/pkg/ml"
	"github.com/kubepulse/kubepulse/pkg/slo"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Engine is the core monitoring engine
type Engine struct {
	client        kubernetes.Interface
	checks        []HealthCheck
	interval      time.Duration
	results       map[string]CheckResult
	resultsMu     sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	alertChan     chan Alert
	metricsChan   chan Metric
	alertManager  *alerts.Manager
	anomalyEngine *ml.AnomalyDetector
	sloTracker    *slo.Tracker
	aiClient      *ai.Client
	
	// New AI components
	predictiveAnalyzer *ai.PredictiveAnalyzer
	assistant          *ai.Assistant
	remediationEngine  *ai.RemediationEngine
	smartAlertManager  *ai.SmartAlertManager
}

// EngineConfig holds configuration for the monitoring engine
type EngineConfig struct {
	KubeClient   kubernetes.Interface
	Interval     time.Duration
	AlertChan    chan Alert
	MetricsChan  chan Metric
	EnableAI     bool
	AIConfig     *ai.Config
}

// NewEngine creates a new monitoring engine
func NewEngine(config EngineConfig) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}

	// Initialize alert manager with default rules
	alertManager := alerts.NewManager()
	alertManager.RegisterChannel(alerts.NewLogChannel())
	for _, rule := range alerts.CreateDefaultRules() {
		alertManager.AddRule(rule)
	}

	engine := &Engine{
		client:        config.KubeClient,
		checks:        make([]HealthCheck, 0),
		interval:      config.Interval,
		results:       make(map[string]CheckResult),
		ctx:           ctx,
		cancel:        cancel,
		alertChan:     config.AlertChan,
		metricsChan:   config.MetricsChan,
		alertManager:  alertManager,
		anomalyEngine: ml.NewAnomalyDetector(),
		sloTracker:    slo.NewTracker(),
	}

	// Initialize AI client if enabled
	if config.EnableAI {
		aiConfig := config.AIConfig
		if aiConfig == nil {
			aiConfig = &ai.Config{} // Use defaults
		}
		engine.aiClient = ai.NewClient(*aiConfig)
		
		// Initialize AI components
		engine.predictiveAnalyzer = ai.NewPredictiveAnalyzer(engine.aiClient)
		engine.assistant = ai.NewAssistant(engine.aiClient)
		engine.smartAlertManager = ai.NewSmartAlertManager(engine.aiClient)
		
		// Initialize remediation engine with safety checks
		executor := ai.NewKubectlExecutor("")
		safetyChecker := ai.NewDefaultSafetyChecker()
		engine.remediationEngine = ai.NewRemediationEngine(engine.aiClient, executor, safetyChecker)
		
		klog.Info("AI-powered diagnostics enabled with predictive analytics, assistant, and auto-remediation")
	}

	return engine
}

// AddCheck adds a health check to the engine
func (e *Engine) AddCheck(check HealthCheck) {
	e.checks = append(e.checks, check)
}

// RemoveCheck removes a health check from the engine
func (e *Engine) RemoveCheck(name string) error {
	for i, check := range e.checks {
		if check.Name() == name {
			e.checks = append(e.checks[:i], e.checks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("check %s not found", name)
}

// Start begins the monitoring loop
func (e *Engine) Start() error {
	klog.Info("Starting monitoring engine")

	// Run initial checks
	e.runChecks()

	// Start periodic monitoring
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.runChecks()
		case <-e.ctx.Done():
			klog.Info("Monitoring engine stopped")
			return nil
		}
	}
}

// Stop halts the monitoring engine
func (e *Engine) Stop() {
	klog.Info("Stopping monitoring engine")
	e.cancel()
}

// runChecks executes all health checks in parallel
func (e *Engine) runChecks() {
	var wg sync.WaitGroup
	resultsChan := make(chan CheckResult, len(e.checks))

	for _, check := range e.checks {
		wg.Add(1)
		go func(hc HealthCheck) {
			defer wg.Done()

			start := time.Now()
			ctx, cancel := context.WithTimeout(e.ctx, 30*time.Second)
			defer cancel()

			result, err := hc.Check(ctx, e.client)
			result.Duration = time.Since(start)

			if err != nil {
				result.Status = HealthStatusUnknown
				result.Error = err
				result.Message = fmt.Sprintf("Check failed: %v", err)
			}

			resultsChan <- result
		}(check)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		e.storeResult(result)
		e.processResult(result)
	}
}

// storeResult saves a check result
func (e *Engine) storeResult(result CheckResult) {
	e.resultsMu.Lock()
	defer e.resultsMu.Unlock()
	e.results[result.Name] = result
}

// processResult handles alerts and metrics from a check result
func (e *Engine) processResult(result CheckResult) {
	// Run AI analysis for failed health checks
	if e.aiClient != nil && (result.Status == HealthStatusUnhealthy || result.Status == HealthStatusDegraded) {
		go e.runAIAnalysis(result)
	}

	// Convert to alerts.CheckResult to avoid import cycle
	alertResult := alerts.CheckResult{
		Name:      result.Name,
		Status:    alerts.HealthStatus(result.Status),
		Message:   result.Message,
		Details:   result.Details,
		Timestamp: result.Timestamp,
	}

	// Process through alert manager
	if err := e.alertManager.ProcessCheckResult(e.ctx, alertResult); err != nil {
		klog.Errorf("Failed to process alert: %v", err)
	}

	// Run anomaly detection on metrics
	if len(result.Metrics) > 0 {
		// Convert metrics to ML format
		mlMetrics := make([]ml.Metric, len(result.Metrics))
		for i, metric := range result.Metrics {
			mlMetrics[i] = ml.Metric{
				Name:      metric.Name,
				Value:     metric.Value,
				Unit:      metric.Unit,
				Labels:    metric.Labels,
				Timestamp: metric.Timestamp,
			}
		}

		predictions := e.anomalyEngine.DetectAnomalies(e.ctx, mlMetrics)

		// Convert predictions back to core format
		corePredictions := make([]Prediction, len(predictions))
		for i, pred := range predictions {
			corePredictions[i] = Prediction{
				Timestamp:   pred.Timestamp,
				Status:      HealthStatus(pred.Status),
				Probability: pred.Probability,
				Reason:      pred.Reason,
			}
		}
		result.Predictions = corePredictions
	}

	// Send to channels for backward compatibility
	if result.Status == HealthStatusUnhealthy || result.Status == HealthStatusDegraded {
		if e.alertChan != nil {
			alert := Alert{
				ID:        fmt.Sprintf("%s-%d", result.Name, time.Now().Unix()),
				Name:      result.Name,
				Severity:  e.getSeverity(result),
				Message:   result.Message,
				Details:   result.Details,
				Source:    "kubepulse",
				Timestamp: result.Timestamp,
				Status:    AlertStatusFiring,
			}

			select {
			case e.alertChan <- alert:
			case <-time.After(time.Second):
				klog.Warning("Alert channel full, dropping alert")
			}
		}
	}

	// Send metrics
	if e.metricsChan != nil {
		for _, metric := range result.Metrics {
			select {
			case e.metricsChan <- metric:
			case <-time.After(time.Second):
				klog.Warning("Metrics channel full, dropping metric")
			}
		}
	}
}

// getSeverity determines alert severity based on check result
func (e *Engine) getSeverity(result CheckResult) AlertSeverity {
	switch result.Status {
	case HealthStatusUnhealthy:
		return AlertSeverityCritical
	case HealthStatusDegraded:
		return AlertSeverityWarning
	default:
		return AlertSeverityInfo
	}
}

// GetResults returns all current check results
func (e *Engine) GetResults() map[string]CheckResult {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	results := make(map[string]CheckResult)
	for k, v := range e.results {
		results[k] = v
	}
	return results
}

// GetResult returns a specific check result by name
func (e *Engine) GetResult(name string) (CheckResult, bool) {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	result, exists := e.results[name]
	return result, exists
}

// GetClusterHealth returns the overall cluster health
func (e *Engine) GetClusterHealth(clusterName string) ClusterHealth {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	checks := make([]CheckResult, 0, len(e.results))
	var totalScore float64
	var weightedScore float64
	var totalWeight float64
	healthyCount := 0

	for _, result := range e.results {
		checks = append(checks, result)

		// Calculate scores
		score := e.calculateScore(result)
		weight := e.getWeight(result)

		totalScore += score
		weightedScore += score * weight
		totalWeight += weight

		if result.Status == HealthStatusHealthy {
			healthyCount++
		}
	}

	// Calculate overall status
	overallStatus := HealthStatusHealthy
	if healthyCount == 0 {
		overallStatus = HealthStatusUnhealthy
	} else if healthyCount < len(checks) {
		overallStatus = HealthStatusDegraded
	}

	// Calculate health score
	rawScore := 0.0
	if len(checks) > 0 {
		rawScore = (totalScore / float64(len(checks))) * 100
	}

	weighted := 0.0
	if totalWeight > 0 {
		weighted = (weightedScore / totalWeight) * 100
	}

	return ClusterHealth{
		ClusterName: clusterName,
		Status:      overallStatus,
		Score: HealthScore{
			Raw:        rawScore,
			Weighted:   weighted,
			Trend:      "stable", // TODO: Implement trend calculation
			Confidence: 0.95,     // TODO: Implement ML confidence
			Forecast:   "stable", // TODO: Implement forecasting
		},
		Checks:    checks,
		Timestamp: time.Now(),
	}
}

// calculateScore converts health status to numeric score
func (e *Engine) calculateScore(result CheckResult) float64 {
	switch result.Status {
	case HealthStatusHealthy:
		return 1.0
	case HealthStatusDegraded:
		return 0.5
	case HealthStatusUnhealthy:
		return 0.0
	default:
		return 0.25
	}
}

// getWeight returns the weight based on check criticality
func (e *Engine) getWeight(result CheckResult) float64 {
	// TODO: Get criticality from check
	// For now, return default weight
	return 1.0
}

// runAIAnalysis performs AI-powered analysis on health check failures
func (e *Engine) runAIAnalysis(result CheckResult) {
	if e.aiClient == nil {
		return
	}

	klog.V(2).Infof("Running AI analysis for failed health check: %s", result.Name)

	// Build diagnostic context
	context := e.buildDiagnosticContext(result)

	// Convert to AI types and run diagnostic analysis
	aiResult := e.convertToAICheckResult(result)
	diagnosisResp, err := e.aiClient.AnalyzeDiagnostic(e.ctx, &aiResult, context)
	if err != nil {
		klog.Errorf("AI diagnostic analysis failed for %s: %v", result.Name, err)
		return
	}

	klog.Infof("AI Diagnosis for %s: %s (confidence: %.2f)", 
		result.Name, diagnosisResp.Summary, diagnosisResp.Confidence)

	// Run healing analysis if diagnosis confidence is high
	if diagnosisResp.Confidence > 0.7 {
		healingResp, err := e.aiClient.AnalyzeHealing(e.ctx, &aiResult, context)
		if err != nil {
			klog.Errorf("AI healing analysis failed for %s: %v", result.Name, err)
			return
		}

		klog.Infof("AI Healing suggestions for %s: %d recommendations, %d actions", 
			result.Name, len(healingResp.Recommendations), len(healingResp.Actions))

		// Store AI insights in the result
		e.storeAIInsights(result.Name, diagnosisResp, healingResp)
	}
}

// buildDiagnosticContext creates context for AI analysis
func (e *Engine) buildDiagnosticContext(result CheckResult) ai.DiagnosticContext {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	// Get related checks and convert them
	relatedChecks := make([]ai.CheckResult, 0)
	for _, checkResult := range e.results {
		if checkResult.Name != result.Name {
			relatedChecks = append(relatedChecks, e.convertToAICheckResult(checkResult))
		}
	}

	// Convert metrics
	aiMetrics := make([]ai.Metric, len(result.Metrics))
	for i, metric := range result.Metrics {
		aiMetrics[i] = ai.Metric{
			Name:      metric.Name,
			Value:     metric.Value,
			Unit:      metric.Unit,
			Type:      ai.MetricType(metric.Type),
			Labels:    metric.Labels,
			Timestamp: metric.Timestamp,
		}
	}

	// Build context
	context := ai.DiagnosticContext{
		ClusterName:   "default", // TODO: Get from config
		ResourceType:  extractResourceType(result.Name),
		ResourceName:  extractResourceName(result.Name),
		ErrorLogs:     extractErrorLogs(result),
		Events:        extractEvents(result),
		Metrics:       aiMetrics,
		RelatedChecks: relatedChecks,
	}

	return context
}

// storeAIInsights stores AI analysis results
func (e *Engine) storeAIInsights(checkName string, diagnosis *ai.AnalysisResponse, healing *ai.AnalysisResponse) {
	e.resultsMu.Lock()
	defer e.resultsMu.Unlock()

	if result, exists := e.results[checkName]; exists {
		// Add AI insights to the result
		if result.Details == nil {
			result.Details = make(map[string]interface{})
		}
		
		result.Details["ai_diagnosis"] = diagnosis
		result.Details["ai_healing"] = healing
		result.Details["ai_analyzed_at"] = time.Now()
		
		e.results[checkName] = result
	}
}

// GetAIInsights returns AI insights for cluster health
func (e *Engine) GetAIInsights() (*ai.InsightSummary, error) {
	if e.aiClient == nil {
		return nil, fmt.Errorf("AI client not enabled")
	}

	clusterHealth := e.GetClusterHealth("default")
	aiClusterHealth := e.convertToAIClusterHealth(clusterHealth)
	return e.aiClient.AnalyzeCluster(e.ctx, &aiClusterHealth)
}

// QueryAssistant processes natural language queries
func (e *Engine) QueryAssistant(query string) (*ai.QueryResponse, error) {
	if e.assistant == nil {
		return nil, fmt.Errorf("AI assistant not enabled")
	}
	
	clusterHealth := e.GetClusterHealth("default")
	aiClusterHealth := e.convertToAIClusterHealth(clusterHealth)
	return e.assistant.Query(e.ctx, query, &aiClusterHealth)
}

// GetPredictiveInsights returns AI predictions about future issues
func (e *Engine) GetPredictiveInsights() ([]ai.PredictiveInsight, error) {
	if e.predictiveAnalyzer == nil {
		return nil, fmt.Errorf("predictive analyzer not enabled")
	}
	
	// Collect all metrics
	metrics := []ai.Metric{}
	e.resultsMu.RLock()
	for _, result := range e.results {
		for _, metric := range result.Metrics {
			metrics = append(metrics, ai.Metric{
				Name:      metric.Name,
				Value:     metric.Value,
				Unit:      metric.Unit,
				Type:      ai.MetricType(metric.Type),
				Labels:    metric.Labels,
				Timestamp: metric.Timestamp,
			})
		}
	}
	e.resultsMu.RUnlock()
	
	return e.predictiveAnalyzer.AnalyzeTrends(e.ctx, metrics)
}

// GetRemediationSuggestions returns AI-powered remediation suggestions
func (e *Engine) GetRemediationSuggestions(checkName string) ([]ai.RemediationAction, error) {
	if e.remediationEngine == nil {
		return nil, fmt.Errorf("remediation engine not enabled")
	}
	
	result, exists := e.GetResult(checkName)
	if !exists {
		return nil, fmt.Errorf("check result not found: %s", checkName)
	}
	
	if result.Status == HealthStatusHealthy {
		return []ai.RemediationAction{}, nil
	}
	
	// Build diagnostic context
	context := e.buildDiagnosticContext(result)
	aiResult := e.convertToAICheckResult(result)
	
	return e.remediationEngine.GenerateRemediation(e.ctx, aiResult, context)
}

// ExecuteRemediation executes an AI-suggested remediation
func (e *Engine) ExecuteRemediation(actionID string, dryRun bool) (*ai.RemediationRecord, error) {
	if e.remediationEngine == nil {
		return nil, fmt.Errorf("remediation engine not enabled")
	}
	
	// This would need to store and retrieve actions properly
	// For now, return error
	return nil, fmt.Errorf("action retrieval not implemented")
}

// GetSmartAlertInsights returns intelligent alert insights
func (e *Engine) GetSmartAlertInsights() (*ai.AlertInsights, error) {
	if e.smartAlertManager == nil {
		return nil, fmt.Errorf("smart alert manager not enabled")
	}
	
	return e.smartAlertManager.GetAlertInsights(e.ctx)
}

// Helper functions for extracting diagnostic context
func extractResourceType(checkName string) string {
	if strings.Contains(strings.ToLower(checkName), "pod") {
		return "pod"
	}
	if strings.Contains(strings.ToLower(checkName), "node") {
		return "node"
	}
	if strings.Contains(strings.ToLower(checkName), "service") {
		return "service"
	}
	return "unknown"
}

func extractResourceName(checkName string) string {
	// Extract resource name from check name
	parts := strings.Split(checkName, "-")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return checkName
}

func extractErrorLogs(result CheckResult) []string {
	logs := []string{}
	if result.Error != nil {
		logs = append(logs, result.Error.Error())
	}
	if result.Message != "" {
		logs = append(logs, result.Message)
	}
	return logs
}

func extractEvents(result CheckResult) []string {
	events := []string{}
	if result.Details != nil {
		if eventData, exists := result.Details["events"]; exists {
			if eventSlice, ok := eventData.([]string); ok {
				events = append(events, eventSlice...)
			}
		}
	}
	return events
}

// Conversion functions between core and ai types

// convertToAICheckResult converts core.CheckResult to ai.CheckResult
func (e *Engine) convertToAICheckResult(result CheckResult) ai.CheckResult {
	aiMetrics := make([]ai.Metric, len(result.Metrics))
	for i, metric := range result.Metrics {
		aiMetrics[i] = ai.Metric{
			Name:      metric.Name,
			Value:     metric.Value,
			Unit:      metric.Unit,
			Type:      ai.MetricType(metric.Type),
			Labels:    metric.Labels,
			Timestamp: metric.Timestamp,
		}
	}

	aiPredictions := make([]ai.Prediction, len(result.Predictions))
	for i, pred := range result.Predictions {
		aiPredictions[i] = ai.Prediction{
			Timestamp:   pred.Timestamp,
			Status:      ai.HealthStatus(pred.Status),
			Probability: pred.Probability,
			Reason:      pred.Reason,
		}
	}

	return ai.CheckResult{
		Name:        result.Name,
		Status:      ai.HealthStatus(result.Status),
		Message:     result.Message,
		Details:     result.Details,
		Error:       result.Error,
		Timestamp:   result.Timestamp,
		Duration:    result.Duration,
		Metrics:     aiMetrics,
		Predictions: aiPredictions,
	}
}

// convertToAIClusterHealth converts core.ClusterHealth to ai.ClusterHealth
func (e *Engine) convertToAIClusterHealth(clusterHealth ClusterHealth) ai.ClusterHealth {
	aiChecks := make([]ai.CheckResult, len(clusterHealth.Checks))
	for i, check := range clusterHealth.Checks {
		aiChecks[i] = e.convertToAICheckResult(check)
	}

	return ai.ClusterHealth{
		ClusterName: clusterHealth.ClusterName,
		Status:      ai.HealthStatus(clusterHealth.Status),
		Score: ai.HealthScore{
			Raw:        clusterHealth.Score.Raw,
			Weighted:   clusterHealth.Score.Weighted,
			Trend:      clusterHealth.Score.Trend,
			Confidence: clusterHealth.Score.Confidence,
			Forecast:   clusterHealth.Score.Forecast,
		},
		Checks:    aiChecks,
		Timestamp: clusterHealth.Timestamp,
	}
}
