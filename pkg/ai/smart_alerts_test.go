package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSmartAlertManager(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	if manager == nil {
		t.Fatal("expected smart alert manager to be created")
	}
	
	if manager.client != client {
		t.Error("expected manager to use provided client")
	}
	
	if manager.alertHistory == nil {
		t.Error("expected alert history to be initialized")
	}
	
	if manager.correlator == nil {
		t.Error("expected correlator to be initialized")
	}
	
	if manager.suppressor == nil {
		t.Error("expected suppressor to be initialized")
	}
	
	// Test alert history initialization
	if manager.alertHistory.alerts == nil {
		t.Error("expected alerts slice to be initialized")
	}
	
	if len(manager.alertHistory.alerts) != 0 {
		t.Error("expected alerts slice to be empty initially")
	}
	
	if manager.alertHistory.patterns == nil {
		t.Error("expected patterns map to be initialized")
	}
	
	if manager.alertHistory.maxHistory != 10000 {
		t.Errorf("expected maxHistory to be 10000, got %d", manager.alertHistory.maxHistory)
	}
	
	// Test correlator initialization
	if manager.correlator.timeWindow != 5*time.Minute {
		t.Errorf("expected timeWindow to be 5 minutes, got %v", manager.correlator.timeWindow)
	}
	
	if manager.correlator.correlations == nil {
		t.Error("expected correlations map to be initialized")
	}
	
	// Test suppressor initialization
	if len(manager.suppressor.thresholds) == 0 {
		t.Error("expected thresholds to be set")
	}
	
	if len(manager.suppressor.rules) == 0 {
		t.Error("expected default suppression rules to be set")
	}
}

func TestCalculateNoiseScore(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	tests := []struct {
		name      string
		alert     SmartAlert
		setup     func(*SmartAlertManager)
		minScore  float64
		maxScore  float64
	}{
		{
			name: "info alert",
			alert: SmartAlert{
				Name:     "test-alert",
				Severity: "info",
			},
			minScore: 0.4, // Should get points for low severity
			maxScore: 1.0,
		},
		{
			name: "critical alert",
			alert: SmartAlert{
				Name:     "critical-alert",
				Severity: "critical",
			},
			minScore: 0.0,
			maxScore: 0.6, // Should get fewer points
		},
		{
			name: "frequent alert",
			alert: SmartAlert{
				Name:     "frequent-alert",
				Severity: "warning",
			},
			setup: func(m *SmartAlertManager) {
				// Add pattern with high frequency
				m.alertHistory.patterns["frequent-alert"] = &AlertPattern{
					Name:      "frequent-alert",
					Frequency: 2 * time.Minute, // Very frequent
				}
			},
			minScore: 0.6, // Should get points for frequency
			maxScore: 1.0,
		},
		{
			name: "transient issue",
			alert: SmartAlert{
				Name:     "transient-alert",
				Message:  "connection reset by peer",
				Severity: "warning",
			},
			minScore: 0.7, // Should get points for being transient
			maxScore: 1.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(manager)
			}
			
			score := manager.calculateNoiseScore(tt.alert)
			
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("expected noise score between %f and %f, got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestCalculatePriority(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	tests := []struct {
		name        string
		alert       SmartAlert
		setup       func(*SmartAlertManager)
		minPriority int
		maxPriority int
	}{
		{
			name: "critical alert",
			alert: SmartAlert{
				Name:       "critical-alert",
				Severity:   "critical",
				NoiseScore: 0.1, // Low noise
			},
			minPriority: 80,
			maxPriority: 100,
		},
		{
			name: "warning alert",
			alert: SmartAlert{
				Name:       "warning-alert",
				Severity:   "warning",
				NoiseScore: 0.3,
			},
			minPriority: 50,
			maxPriority: 70,
		},
		{
			name: "noisy alert",
			alert: SmartAlert{
				Name:       "noisy-alert",
				Severity:   "error",
				NoiseScore: 0.9, // High noise
			},
			minPriority: 20,
			maxPriority: 60,
		},
		{
			name: "correlated alert",
			alert: SmartAlert{
				Name:        "correlated-alert",
				Severity:    "warning",
				NoiseScore:  0.2,
				Correlation: []string{"alert1", "alert2", "alert3"},
			},
			minPriority: 65, // Gets boost from correlations
			maxPriority: 85,
		},
		{
			name: "frequent pattern alert",
			alert: SmartAlert{
				Name:       "frequent-alert",
				Severity:   "error",
				NoiseScore: 0.2,
			},
			setup: func(m *SmartAlertManager) {
				m.alertHistory.patterns["frequent-alert"] = &AlertPattern{
					Name:        "frequent-alert",
					Occurrences: 15, // Very frequent
				}
			},
			minPriority: 60, // Gets penalty for frequency
			maxPriority: 80,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(manager)
			}
			
			priority := manager.calculatePriority(tt.alert)
			
			if priority < tt.minPriority || priority > tt.maxPriority {
				t.Errorf("expected priority between %d and %d, got %d", tt.minPriority, tt.maxPriority, priority)
			}
			
			// Ensure priority is within valid range
			if priority < 1 || priority > 100 {
				t.Errorf("priority %d is outside valid range [1, 100]", priority)
			}
		})
	}
}

func TestIsTransientIssue(t *testing.T) {
	manager := &SmartAlertManager{}
	
	tests := []struct {
		name     string
		alert    SmartAlert
		expected bool
	}{
		{
			name: "connection reset",
			alert: SmartAlert{
				Message: "connection reset by peer",
			},
			expected: true,
		},
		{
			name: "timeout error",
			alert: SmartAlert{
				Message: "request timeout occurred",
			},
			expected: true,
		},
		{
			name: "temporary failure",
			alert: SmartAlert{
				Message: "temporary failure in system",
			},
			expected: true,
		},
		{
			name: "pod being terminated",
			alert: SmartAlert{
				Message: "pod is being terminated",
			},
			expected: true,
		},
		{
			name: "container starting",
			alert: SmartAlert{
				Message: "container is starting up",
			},
			expected: true,
		},
		{
			name: "permanent error",
			alert: SmartAlert{
				Message: "disk full error",
			},
			expected: false,
		},
		{
			name: "configuration error",
			alert: SmartAlert{
				Message: "invalid configuration detected",
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isTransientIssue(tt.alert)
			if result != tt.expected {
				t.Errorf("expected isTransientIssue to return %v for %q, got %v", tt.expected, tt.alert.Message, result)
			}
		})
	}
}

func TestCanAutoResolve(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	alert := SmartAlert{
		Name:     "test-alert",
		Resource: "test-resource",
	}
	
	// Test with no similar alerts
	result := manager.canAutoResolve(alert)
	if result {
		t.Error("expected canAutoResolve to return false with no similar alerts")
	}
	
	// Add similar alerts with high auto-resolve rate
	now := time.Now()
	for i := 0; i < 5; i++ {
		similar := SmartAlert{
			Name:        "test-alert",
			Resource:    "test-resource",
			AutoResolve: true,
			Timestamp:   now.Add(-time.Duration(i) * time.Hour),
		}
		manager.alertHistory.alerts = append(manager.alertHistory.alerts, similar)
	}
	
	result = manager.canAutoResolve(alert)
	if !result {
		t.Error("expected canAutoResolve to return true with high auto-resolve rate")
	}
	
	// Add more alerts with low auto-resolve rate
	for i := 0; i < 3; i++ {
		similar := SmartAlert{
			Name:        "test-alert",
			Resource:    "test-resource",
			AutoResolve: false,
			Timestamp:   now.Add(-time.Duration(i+6) * time.Hour),
		}
		manager.alertHistory.alerts = append(manager.alertHistory.alerts, similar)
	}
	
	result = manager.canAutoResolve(alert)
	if result {
		t.Error("expected canAutoResolve to return false with mixed auto-resolve rate")
	}
}

func TestUpdateHistory(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	alert := SmartAlert{
		ID:   "test-alert-1",
		Name: "test-alert",
	}
	
	// Test adding alert
	manager.updateHistory(alert)
	
	if len(manager.alertHistory.alerts) != 1 {
		t.Errorf("expected 1 alert in history, got %d", len(manager.alertHistory.alerts))
	}
	
	if manager.alertHistory.alerts[0].ID != "test-alert-1" {
		t.Errorf("expected alert ID %q, got %q", "test-alert-1", manager.alertHistory.alerts[0].ID)
	}
	
	// Test history trimming
	manager.alertHistory.maxHistory = 2
	
	alert2 := SmartAlert{ID: "test-alert-2", Name: "test-alert-2"}
	alert3 := SmartAlert{ID: "test-alert-3", Name: "test-alert-3"}
	
	manager.updateHistory(alert2)
	manager.updateHistory(alert3)
	
	if len(manager.alertHistory.alerts) != 2 {
		t.Errorf("expected history to be trimmed to 2 alerts, got %d", len(manager.alertHistory.alerts))
	}
	
	// Should have removed the first alert
	if manager.alertHistory.alerts[0].ID != "test-alert-2" {
		t.Errorf("expected first alert to be %q, got %q", "test-alert-2", manager.alertHistory.alerts[0].ID)
	}
}

func TestUpdatePatterns(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	now := time.Now()
	alert := SmartAlert{
		Name:        "test-alert",
		Timestamp:   now,
		Correlation: []string{"related-alert"},
	}
	
	// Test creating new pattern
	manager.updatePatterns(alert)
	
	pattern, exists := manager.alertHistory.patterns["test-alert"]
	if !exists {
		t.Fatal("expected pattern to be created")
	}
	
	if pattern.Name != "test-alert" {
		t.Errorf("expected pattern name %q, got %q", "test-alert", pattern.Name)
	}
	
	if pattern.Occurrences != 1 {
		t.Errorf("expected 1 occurrence, got %d", pattern.Occurrences)
	}
	
	if pattern.Frequency != 24*time.Hour {
		t.Errorf("expected default frequency 24h, got %v", pattern.Frequency)
	}
	
	if len(pattern.Correlated) != 1 || pattern.Correlated[0] != "related-alert" {
		t.Errorf("expected correlated alerts [%q], got %v", "related-alert", pattern.Correlated)
	}
	
	// Test updating existing pattern
	alert2 := SmartAlert{
		Name:      "test-alert",
		Timestamp: now.Add(30 * time.Minute),
	}
	
	manager.updatePatterns(alert2)
	
	if pattern.Occurrences != 2 {
		t.Errorf("expected 2 occurrences, got %d", pattern.Occurrences)
	}
	
	if pattern.Frequency != 30*time.Minute {
		t.Errorf("expected frequency 30m, got %v", pattern.Frequency)
	}
}

func TestGetRecentAlerts(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	now := time.Now()
	
	// Add alerts at different times
	alerts := []SmartAlert{
		{ID: "old", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "recent1", Timestamp: now.Add(-10 * time.Minute)},
		{ID: "recent2", Timestamp: now.Add(-5 * time.Minute)},
		{ID: "very-recent", Timestamp: now.Add(-1 * time.Minute)},
	}
	
	for _, alert := range alerts {
		manager.alertHistory.alerts = append(manager.alertHistory.alerts, alert)
	}
	
	// Get alerts from last 15 minutes
	recent := manager.getRecentAlerts(15 * time.Minute)
	
	expected := 3 // recent1, recent2, very-recent
	if len(recent) != expected {
		t.Errorf("expected %d recent alerts, got %d", expected, len(recent))
	}
	
	// Verify they are in correct order (most recent first)
	expectedIDs := []string{"very-recent", "recent2", "recent1"}
	for i, alert := range recent {
		if alert.ID != expectedIDs[i] {
			t.Errorf("expected alert %d to be %q, got %q", i, expectedIDs[i], alert.ID)
		}
	}
}

func TestFindSimilarAlerts(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	now := time.Now()
	target := SmartAlert{
		Name:     "test-alert",
		Resource: "test-resource",
	}
	
	// Add similar and different alerts
	alerts := []SmartAlert{
		{Name: "test-alert", Resource: "test-resource", Timestamp: now.Add(-10 * time.Minute)}, // Similar
		{Name: "test-alert", Resource: "other-resource", Timestamp: now.Add(-5 * time.Minute)}, // Different resource
		{Name: "other-alert", Resource: "test-resource", Timestamp: now.Add(-3 * time.Minute)}, // Different name
		{Name: "test-alert", Resource: "test-resource", Timestamp: now.Add(-2 * time.Hour)},    // Too old
		{Name: "test-alert", Resource: "test-resource", Timestamp: now.Add(-1 * time.Minute)},  // Similar
	}
	
	for _, alert := range alerts {
		manager.alertHistory.alerts = append(manager.alertHistory.alerts, alert)
	}
	
	similar := manager.findSimilarAlerts(target, 30*time.Minute)
	
	expected := 2 // Only 2 alerts match name, resource, and time window
	if len(similar) != expected {
		t.Errorf("expected %d similar alerts, got %d", expected, len(similar))
	}
	
	// Verify all similar alerts have matching name and resource
	for _, alert := range similar {
		if alert.Name != target.Name {
			t.Errorf("similar alert has wrong name: %q vs %q", alert.Name, target.Name)
		}
		if alert.Resource != target.Resource {
			t.Errorf("similar alert has wrong resource: %q vs %q", alert.Resource, target.Resource)
		}
	}
}

func TestIdentifyPatterns(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	// Add patterns with different occurrence counts
	manager.alertHistory.patterns = map[string]*AlertPattern{
		"frequent": {
			Name:        "frequent",
			Occurrences: 10,
			Frequency:   5 * time.Minute,
		},
		"infrequent": {
			Name:        "infrequent",
			Occurrences: 3, // Below threshold
			Frequency:   1 * time.Hour,
		},
		"very-frequent": {
			Name:        "very-frequent",
			Occurrences: 20,
			Frequency:   2 * time.Minute,
		},
	}
	
	patterns := manager.identifyPatterns()
	
	// Should only return patterns with > 5 occurrences
	expected := 2
	if len(patterns) != expected {
		t.Errorf("expected %d patterns, got %d", expected, len(patterns))
	}
	
	// Verify sorting by frequency (most frequent first)
	if patterns[0].Name != "very-frequent" {
		t.Errorf("expected first pattern to be %q, got %q", "very-frequent", patterns[0].Name)
	}
	
	if patterns[1].Name != "frequent" {
		t.Errorf("expected second pattern to be %q, got %q", "frequent", patterns[1].Name)
	}
}

func TestPredictFutureAlerts(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	predictions := manager.predictFutureAlerts(context.Background())
	
	// Should return at least one prediction (placeholder)
	if len(predictions) == 0 {
		t.Error("expected at least one prediction")
	}
	
	// Verify prediction structure
	pred := predictions[0]
	if pred.AlertType == "" {
		t.Error("expected prediction to have alert type")
	}
	
	if pred.Probability <= 0 || pred.Probability > 1 {
		t.Errorf("expected probability between 0 and 1, got %f", pred.Probability)
	}
	
	if pred.TimeWindow == "" {
		t.Error("expected prediction to have time window")
	}
	
	if pred.Prevention == "" {
		t.Error("expected prediction to have prevention advice")
	}
}

func TestGenerateRecommendations(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	patterns := []AlertPattern{
		{
			Name:      "high-frequency-alert",
			Frequency: 5 * time.Minute, // Very frequent
		},
		{
			Name:      "normal-alert",
			Frequency: 30 * time.Minute, // Normal frequency
		},
	}
	
	recommendations := manager.generateRecommendations(context.Background(), patterns)
	
	// Should generate recommendation for high-frequency alert
	if len(recommendations) == 0 {
		t.Error("expected at least one recommendation")
	}
	
	found := false
	for _, rec := range recommendations {
		if strings.Contains(rec, "high-frequency-alert") && strings.Contains(rec, "frequently") {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("expected recommendation about high-frequency alert")
	}
}

func TestCalculateNoiseReduction(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	// Test with no alerts
	reduction := manager.calculateNoiseReduction()
	if reduction != 0 {
		t.Errorf("expected 0 noise reduction with no alerts, got %f", reduction)
	}
	
	// Add alerts with some suppressed
	now := time.Now()
	alerts := []SmartAlert{
		{Suppressed: true, Timestamp: now.Add(-1 * time.Hour)},
		{Suppressed: false, Timestamp: now.Add(-30 * time.Minute)},
		{Suppressed: true, Timestamp: now.Add(-10 * time.Minute)},
		{Suppressed: false, Timestamp: now.Add(-5 * time.Minute)},
	}
	
	for _, alert := range alerts {
		manager.alertHistory.alerts = append(manager.alertHistory.alerts, alert)
	}
	
	reduction = manager.calculateNoiseReduction()
	expected := 0.5 // 2 out of 4 alerts suppressed
	if reduction != expected {
		t.Errorf("expected noise reduction %f, got %f", expected, reduction)
	}
}

func TestGetAlertVolumeStats(t *testing.T) {
	client := NewClient(Config{})
	manager := NewSmartAlertManager(client)
	
	// Add alerts with different severities
	now := time.Now()
	alerts := []SmartAlert{
		{Severity: "critical", Timestamp: now.Add(-1 * time.Hour)},
		{Severity: "critical", Timestamp: now.Add(-30 * time.Minute)},
		{Severity: "warning", Timestamp: now.Add(-20 * time.Minute)},
		{Severity: "warning", Timestamp: now.Add(-10 * time.Minute)},
		{Severity: "warning", Timestamp: now.Add(-5 * time.Minute)},
		{Severity: "info", Timestamp: now.Add(-2 * time.Minute)},
	}
	
	for _, alert := range alerts {
		manager.alertHistory.alerts = append(manager.alertHistory.alerts, alert)
	}
	
	stats := manager.getAlertVolumeStats()
	
	expected := map[string]int{
		"critical": 2,
		"warning":  3,
		"info":     1,
	}
	
	for severity, expectedCount := range expected {
		if count, exists := stats[severity]; !exists || count != expectedCount {
			t.Errorf("expected %d alerts of severity %q, got %d", expectedCount, severity, count)
		}
	}
}

func TestDefaultSuppressionRules(t *testing.T) {
	rules := getDefaultSuppressionRules()
	
	if len(rules) == 0 {
		t.Error("expected default suppression rules to be defined")
	}
	
	// Test rule structure
	for i, rule := range rules {
		if rule.Name == "" {
			t.Errorf("rule %d has empty name", i)
		}
		
		if rule.Condition == nil {
			t.Errorf("rule %d has nil condition", i)
		}
		
		if rule.Action == "" {
			t.Errorf("rule %d has empty action", i)
		}
	}
	
	// Test duplicate suppression rule
	duplicateRule := rules[0] // First rule should be duplicate suppression
	
	now := time.Now()
	alert := SmartAlert{
		Name:     "test-alert",
		Resource: "test-resource",
	}
	
	history := []SmartAlert{
		{
			Name:      "test-alert",
			Resource:  "test-resource",
			Timestamp: now.Add(-30 * time.Second), // Within 1 minute
		},
	}
	
	if !duplicateRule.Condition(alert, history) {
		t.Error("duplicate suppression rule should trigger for recent duplicate alert")
	}
	
	// Test high noise suppression rule
	noiseRule := rules[1] // Second rule should be noise suppression
	
	noisyAlert := SmartAlert{
		NoiseScore: 0.9, // High noise
	}
	
	if !noiseRule.Condition(noisyAlert, []SmartAlert{}) {
		t.Error("noise suppression rule should trigger for high noise score")
	}
}

func TestNoiseSuppressorShouldSuppress(t *testing.T) {
	suppressor := &NoiseSuppressor{
		thresholds: map[string]float64{
			"info":     0.8,
			"warning":  0.6,
			"critical": 0.3,
		},
		rules: getDefaultSuppressionRules(),
	}
	
	tests := []struct {
		name     string
		alert    SmartAlert
		history  []SmartAlert
		expected bool
	}{
		{
			name: "high noise info alert",
			alert: SmartAlert{
				Severity:   "info",
				NoiseScore: 0.9,
			},
			expected: true,
		},
		{
			name: "low noise critical alert",
			alert: SmartAlert{
				Severity:   "critical",
				NoiseScore: 0.1,
			},
			expected: false,
		},
		{
			name: "duplicate alert",
			alert: SmartAlert{
				Name:     "test-alert",
				Resource: "test-resource",
			},
			history: []SmartAlert{
				{
					Name:      "test-alert",
					Resource:  "test-resource",
					Timestamp: time.Now().Add(-30 * time.Second),
				},
			},
			expected: true,
		},
		{
			name: "normal alert",
			alert: SmartAlert{
				Name:       "normal-alert",
				Severity:   "warning",
				NoiseScore: 0.3,
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suppressor.shouldSuppress(tt.alert, tt.history)
			if result != tt.expected {
				t.Errorf("expected shouldSuppress to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAlertCorrelatorFindCorrelations(t *testing.T) {
	correlator := &AlertCorrelator{
		timeWindow: 5 * time.Minute,
		correlations: make(map[string][]string),
	}
	
	now := time.Now()
	alert := SmartAlert{
		ID:        "target-alert",
		Resource:  "test-resource",
		RootCause: "memory pressure",
		Timestamp: now,
	}
	
	history := []SmartAlert{
		{
			ID:        "corr1",
			Resource:  "test-resource", // Same resource
			Timestamp: now.Add(-2 * time.Minute),
		},
		{
			ID:        "corr2",
			RootCause: "memory pressure", // Same root cause
			Timestamp: now.Add(-3 * time.Minute),
		},
		{
			ID:        "corr3",
			Timestamp: now.Add(-10 * time.Second), // Time proximity
		},
		{
			ID:        "not-corr1",
			Timestamp: now.Add(-10 * time.Minute), // Outside time window
		},
		{
			ID:        "not-corr2",
			Resource:  "other-resource",
			Timestamp: now.Add(-1 * time.Minute), // Different resource, no other correlation
		},
	}
	
	correlations := correlator.findCorrelations(alert, history)
	
	expected := 3 // Should find 3 correlations
	if len(correlations) != expected {
		t.Errorf("expected %d correlations, got %d: %v", expected, len(correlations), correlations)
	}
	
	// Verify specific correlations
	expectedIDs := []string{"corr1", "corr2", "corr3"}
	for _, expectedID := range expectedIDs {
		found := false
		for _, corrID := range correlations {
			if corrID == expectedID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find correlation with %q", expectedID)
		}
	}
}

func TestAlertCorrelatorAreCorrelated(t *testing.T) {
	correlator := &AlertCorrelator{}
	
	now := time.Now()
	
	tests := []struct {
		name     string
		alert1   SmartAlert
		alert2   SmartAlert
		expected bool
	}{
		{
			name: "same resource",
			alert1: SmartAlert{
				Resource: "test-resource",
			},
			alert2: SmartAlert{
				Resource: "test-resource",
			},
			expected: true,
		},
		{
			name: "same root cause",
			alert1: SmartAlert{
				RootCause: "memory pressure",
			},
			alert2: SmartAlert{
				RootCause: "memory pressure",
			},
			expected: true,
		},
		{
			name: "time proximity",
			alert1: SmartAlert{
				Timestamp: now,
			},
			alert2: SmartAlert{
				Timestamp: now.Add(20 * time.Second),
			},
			expected: true,
		},
		{
			name: "not correlated",
			alert1: SmartAlert{
				Resource:  "resource1",
				RootCause: "cause1",
				Timestamp: now,
			},
			alert2: SmartAlert{
				Resource:  "resource2",
				RootCause: "cause2",
				Timestamp: now.Add(2 * time.Minute),
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := correlator.areCorrelated(tt.alert1, tt.alert2)
			if result != tt.expected {
				t.Errorf("expected areCorrelated to return %v, got %v", tt.expected, result)
			}
		})
	}
}