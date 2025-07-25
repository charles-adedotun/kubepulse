package ai

import (
	"context"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// AIScheduler manages both scheduled and event-driven AI analysis
type AIScheduler struct {
	client         *Client
	analysisEngine *AnalysisEngine
	ctx            context.Context
	cancel         context.CancelFunc
	mutex          sync.RWMutex

	// Scheduling configuration
	config SchedulerConfig

	// Event tracking
	lastAnalysis map[string]time.Time
	eventQueue   chan AIEvent

	// Callbacks for results
	onInsights func(insights interface{})
	onAlert    func(alert interface{})
}

// SchedulerConfig defines the AI analysis schedule
type SchedulerConfig struct {
	// Scheduled analysis intervals
	DailyAnalysisTime string        // "02:00" for 2 AM daily analysis
	PeriodicInterval  time.Duration // 3 hours for periodic checks

	// Event-driven thresholds
	FailureThreshold int     // Number of failures to trigger AI analysis
	AnomalyThreshold float64 // Anomaly score threshold (0-1)

	// Rate limiting
	MinAnalysisInterval time.Duration // Minimum time between AI calls (15 minutes)
	MaxDailyAnalyses    int           // Maximum AI analyses per day

	// Feature flags
	EnableScheduled   bool // Enable scheduled analysis
	EnableEventDriven bool // Enable event-driven analysis
}

// AIEvent represents an event that might trigger AI analysis
type AIEvent struct {
	Type        EventType              `json:"type"`
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Source      string                 `json:"source"`   // component that triggered the event
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// EventType defines the types of events that can trigger AI analysis
type EventType string

const (
	EventHealthCheckFailed      EventType = "health_check_failed"
	EventPodCrashLoop           EventType = "pod_crash_loop"
	EventResourceExhaustion     EventType = "resource_exhaustion"
	EventAnomalyDetected        EventType = "anomaly_detected"
	EventPerformanceDegradation EventType = "performance_degradation"
	EventSecurityAlert          EventType = "security_alert"
	EventScheduledAnalysis      EventType = "scheduled_analysis"
)

// Default configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		DailyAnalysisTime:   "02:00",          // 2 AM daily
		PeriodicInterval:    3 * time.Hour,    // Every 3 hours
		FailureThreshold:    3,                // 3 failures trigger analysis
		AnomalyThreshold:    0.7,              // 70% anomaly confidence
		MinAnalysisInterval: 15 * time.Minute, // Min 15 minutes between analyses
		MaxDailyAnalyses:    8,                // Max 8 AI calls per day
		EnableScheduled:     true,
		EnableEventDriven:   true,
	}
}

// NewAIScheduler creates a new AI scheduler
func NewAIScheduler(client *Client, analysisEngine *AnalysisEngine, config SchedulerConfig) *AIScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &AIScheduler{
		client:         client,
		analysisEngine: analysisEngine,
		ctx:            ctx,
		cancel:         cancel,
		config:         config,
		lastAnalysis:   make(map[string]time.Time),
		eventQueue:     make(chan AIEvent, 100), // Buffer up to 100 events
	}

	return scheduler
}

// Start begins the AI scheduler
func (s *AIScheduler) Start() error {
	klog.Info("Starting AI scheduler with hybrid event+schedule approach")

	// Start the main event processing loop
	go s.processEvents()

	// Start scheduled analysis if enabled
	if s.config.EnableScheduled {
		go s.runScheduledAnalysis()
	}

	klog.Infof("AI scheduler configuration: daily=%s, periodic=%v, max_daily=%d",
		s.config.DailyAnalysisTime, s.config.PeriodicInterval, s.config.MaxDailyAnalyses)

	return nil
}

// Stop gracefully stops the AI scheduler
func (s *AIScheduler) Stop() {
	klog.Info("Stopping AI scheduler")
	s.cancel()
}

// TriggerEvent sends an event to the scheduler for potential AI analysis
func (s *AIScheduler) TriggerEvent(event AIEvent) {
	if !s.config.EnableEventDriven {
		return
	}

	event.Timestamp = time.Now()

	select {
	case s.eventQueue <- event:
		klog.V(3).Infof("Queued AI event: %s (%s)", event.Type, event.Severity)
	default:
		klog.Warning("AI event queue is full, dropping event")
	}
}

// SetCallbacks sets callback functions for AI results
func (s *AIScheduler) SetCallbacks(onInsights, onAlert func(interface{})) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onInsights = onInsights
	s.onAlert = onAlert
}

// processEvents is the main event processing loop
func (s *AIScheduler) processEvents() {
	ticker := time.NewTicker(1 * time.Minute) // Check for events every minute
	defer ticker.Stop()

	eventBuffer := make([]AIEvent, 0, 10)

	for {
		select {
		case <-s.ctx.Done():
			return

		case event := <-s.eventQueue:
			eventBuffer = append(eventBuffer, event)

			// Process events if we have enough or if a critical event occurred
			if s.shouldTriggerAnalysis(eventBuffer) {
				s.processEventBatch(eventBuffer)
				eventBuffer = eventBuffer[:0] // Clear buffer
			}

		case <-ticker.C:
			// Process any buffered events periodically
			if len(eventBuffer) > 0 && s.shouldTriggerPeriodicAnalysis() {
				s.processEventBatch(eventBuffer)
				eventBuffer = eventBuffer[:0]
			}
		}
	}
}

// shouldTriggerAnalysis determines if events should trigger AI analysis
func (s *AIScheduler) shouldTriggerAnalysis(events []AIEvent) bool {
	if len(events) == 0 {
		return false
	}

	// Check if we've hit the minimum interval
	if !s.canRunAnalysis("event-driven") {
		return false
	}

	// Critical events always trigger analysis
	for _, event := range events {
		if event.Severity == "critical" {
			return true
		}
	}

	// Check failure threshold
	failureCount := 0
	for _, event := range events {
		if event.Type == EventHealthCheckFailed || event.Type == EventPodCrashLoop {
			failureCount++
		}
	}

	if failureCount >= s.config.FailureThreshold {
		return true
	}

	// Check for high-severity events
	highSeverityCount := 0
	for _, event := range events {
		if event.Severity == "high" {
			highSeverityCount++
		}
	}

	return highSeverityCount >= 2 // Two high-severity events trigger analysis
}

// shouldTriggerPeriodicAnalysis checks if periodic analysis is due
func (s *AIScheduler) shouldTriggerPeriodicAnalysis() bool {
	return s.canRunAnalysis("periodic")
}

// canRunAnalysis checks if we can run AI analysis based on rate limits
func (s *AIScheduler) canRunAnalysis(analysisType string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check minimum interval
	if lastTime, exists := s.lastAnalysis[analysisType]; exists {
		if time.Since(lastTime) < s.config.MinAnalysisInterval {
			return false
		}
	}

	// Check daily limit
	today := time.Now().Format("2006-01-02")
	dailyCount := 0
	for _, lastTime := range s.lastAnalysis {
		if lastTime.Format("2006-01-02") == today {
			dailyCount++
		}
	}

	return dailyCount < s.config.MaxDailyAnalyses
}

// processEventBatch processes a batch of events with AI analysis
func (s *AIScheduler) processEventBatch(events []AIEvent) {
	if len(events) == 0 {
		return
	}

	klog.Infof("Processing %d events for AI analysis", len(events))

	// Update last analysis time
	s.mutex.Lock()
	s.lastAnalysis["event-driven"] = time.Now()
	s.mutex.Unlock()

	// Create analysis context from events
	analysisContext := s.createAnalysisContext(events)

	// Run AI analysis in a separate goroutine to avoid blocking
	go func() {
		insights, err := s.runAIAnalysis(analysisContext)
		if err != nil {
			klog.Errorf("Event-driven AI analysis failed: %v", err)
			return
		}

		// Send insights via callback
		if s.onInsights != nil {
			s.onInsights(insights)
		}

		// Check if we need to generate alerts
		s.checkForAlerts(insights, events)
	}()
}

// runScheduledAnalysis handles scheduled AI analysis
func (s *AIScheduler) runScheduledAnalysis() {
	// Daily analysis ticker
	dailyTicker := s.createDailyTicker()
	defer dailyTicker.Stop()

	// Periodic analysis ticker
	periodicTicker := time.NewTicker(s.config.PeriodicInterval)
	defer periodicTicker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-dailyTicker.C:
			s.runDailyAnalysis()

		case <-periodicTicker.C:
			s.runPeriodicAnalysis()
		}
	}
}

// createDailyTicker creates a ticker that fires at the specified daily time
func (s *AIScheduler) createDailyTicker() *time.Ticker {
	// Parse the daily analysis time
	now := time.Now()
	targetTime, err := time.Parse("15:04", s.config.DailyAnalysisTime)
	if err != nil {
		klog.Errorf("Invalid daily analysis time format: %v", err)
		// Fallback to every 24 hours
		return time.NewTicker(24 * time.Hour)
	}

	// Calculate next occurrence
	next := time.Date(now.Year(), now.Month(), now.Day(),
		targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())

	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	duration := time.Until(next)

	// Create ticker that fires at the target time daily
	return time.NewTicker(duration)
}

// runDailyAnalysis performs comprehensive daily cluster analysis
func (s *AIScheduler) runDailyAnalysis() {
	if !s.canRunAnalysis("daily") {
		klog.V(2).Info("Skipping daily analysis due to rate limits")
		return
	}

	klog.Info("Starting scheduled daily AI analysis")

	s.mutex.Lock()
	s.lastAnalysis["daily"] = time.Now()
	s.mutex.Unlock()

	// Create comprehensive analysis context
	analysisContext := map[string]interface{}{
		"type":           "daily_comprehensive",
		"timestamp":      time.Now(),
		"scope":          "full_cluster",
		"analysis_depth": "comprehensive",
	}

	go func() {
		insights, err := s.runAIAnalysis(analysisContext)
		if err != nil {
			klog.Errorf("Daily AI analysis failed: %v", err)
			return
		}

		if s.onInsights != nil {
			s.onInsights(insights)
		}

		klog.Info("Daily AI analysis completed successfully")
	}()
}

// runPeriodicAnalysis performs periodic health checks
func (s *AIScheduler) runPeriodicAnalysis() {
	if !s.canRunAnalysis("periodic") {
		klog.V(2).Info("Skipping periodic analysis due to rate limits")
		return
	}

	klog.V(2).Info("Starting scheduled periodic AI analysis")

	s.mutex.Lock()
	s.lastAnalysis["periodic"] = time.Now()
	s.mutex.Unlock()

	analysisContext := map[string]interface{}{
		"type":           "periodic_health",
		"timestamp":      time.Now(),
		"scope":          "health_trends",
		"analysis_depth": "focused",
	}

	go func() {
		insights, err := s.runAIAnalysis(analysisContext)
		if err != nil {
			klog.Errorf("Periodic AI analysis failed: %v", err)
			return
		}

		if s.onInsights != nil {
			s.onInsights(insights)
		}
	}()
}

// createAnalysisContext creates context for AI analysis from events
func (s *AIScheduler) createAnalysisContext(events []AIEvent) map[string]interface{} {
	context := map[string]interface{}{
		"type":           "event_driven",
		"timestamp":      time.Now(),
		"event_count":    len(events),
		"events":         events,
		"analysis_depth": "focused",
	}

	// Summarize event types and severities
	eventSummary := make(map[EventType]int)
	severitySummary := make(map[string]int)

	for _, event := range events {
		eventSummary[event.Type]++
		severitySummary[event.Severity]++
	}

	context["event_summary"] = eventSummary
	context["severity_summary"] = severitySummary

	return context
}

// runAIAnalysis executes AI analysis with the given context
func (s *AIScheduler) runAIAnalysis(analysisContext map[string]interface{}) (interface{}, error) {
	// For now, return the mock insights we implemented earlier
	// In production, this would call the actual AI client

	analysisType, _ := analysisContext["type"].(string)

	insights := map[string]interface{}{
		"status":    "available",
		"timestamp": time.Now().Format(time.RFC3339),
		"type":      analysisType,
		"summary":   "AI analysis completed successfully",
		"insights": []map[string]interface{}{
			{
				"type":       "performance",
				"title":      "Scheduled Analysis Complete",
				"message":    "Cluster performance is optimal based on " + analysisType + " analysis",
				"confidence": 0.88,
				"trend":      "stable",
			},
		},
		"recommendations": []string{
			"Continue monitoring based on scheduled intervals",
			"Review event patterns for optimization opportunities",
		},
		"analysis_context": analysisContext,
	}

	return insights, nil
}

// checkForAlerts examines insights to determine if alerts should be generated
func (s *AIScheduler) checkForAlerts(insights interface{}, events []AIEvent) {
	// Check if any critical conditions require immediate alerts
	hasCriticalEvents := false
	for _, event := range events {
		if event.Severity == "critical" {
			hasCriticalEvents = true
			break
		}
	}

	if hasCriticalEvents && s.onAlert != nil {
		alert := map[string]interface{}{
			"type":        "critical_cluster_event",
			"message":     "Critical cluster events detected and analyzed by AI",
			"timestamp":   time.Now(),
			"events":      events,
			"ai_insights": insights,
		}

		s.onAlert(alert)
	}
}

// GetScheduleStatus returns the current scheduler status
func (s *AIScheduler) GetScheduleStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status := map[string]interface{}{
		"enabled":           s.config.EnableScheduled && s.config.EnableEventDriven,
		"daily_analysis":    s.config.DailyAnalysisTime,
		"periodic_interval": s.config.PeriodicInterval.String(),
		"max_daily_calls":   s.config.MaxDailyAnalyses,
		"last_analyses":     s.lastAnalysis,
		"event_queue_size":  len(s.eventQueue),
	}

	// Calculate daily usage
	today := time.Now().Format("2006-01-02")
	dailyCount := 0
	for _, lastTime := range s.lastAnalysis {
		if lastTime.Format("2006-01-02") == today {
			dailyCount++
		}
	}
	status["daily_usage"] = dailyCount

	return status
}
