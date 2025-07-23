package ai

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// Alert represents a basic alert (temporary until we import from core)
type Alert struct {
	ID        string
	Name      string
	Severity  string
	Message   string
	Source    string
	Timestamp time.Time
}

// SmartAlertManager provides AI-powered alert management
type SmartAlertManager struct {
	client       *Client
	alertHistory *AlertHistory
	correlator   *AlertCorrelator
	suppressor   *NoiseSuppressor
}

// AlertHistory tracks alert patterns
type AlertHistory struct {
	alerts     []SmartAlert
	patterns   map[string]*AlertPattern
	maxHistory int
}

// AlertPattern represents a recurring alert pattern
type AlertPattern struct {
	ID          string
	Name        string
	Occurrences int
	LastSeen    time.Time
	Frequency   time.Duration
	Correlated  []string
}

// SmartAlert represents an intelligent alert
type SmartAlert struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Resource  string    `json:"resource"`
	Timestamp time.Time `json:"timestamp"`

	// AI enhancements
	RootCause   string   `json:"root_cause"`
	Impact      string   `json:"impact"`
	Correlation []string `json:"correlated_alerts"`
	Suppressed  bool     `json:"suppressed"`
	Priority    int      `json:"priority"`
	NoiseScore  float64  `json:"noise_score"`

	// Smart features
	AutoResolve   bool   `json:"auto_resolve"`
	Remediation   string `json:"suggested_remediation"`
	TimeToResolve string `json:"estimated_time_to_resolve"`
}

// AlertCorrelator correlates related alerts
type AlertCorrelator struct {
	timeWindow   time.Duration
	correlations map[string][]string
}

// NoiseSuppressor reduces alert noise
type NoiseSuppressor struct {
	thresholds map[string]float64
	rules      []SuppressionRule
}

// SuppressionRule defines when to suppress alerts
type SuppressionRule struct {
	Name      string
	Condition func(alert SmartAlert, history []SmartAlert) bool
	Action    string
}

// NewSmartAlertManager creates a new smart alert manager
func NewSmartAlertManager(client *Client) *SmartAlertManager {
	return &SmartAlertManager{
		client: client,
		alertHistory: &AlertHistory{
			alerts:     []SmartAlert{},
			patterns:   make(map[string]*AlertPattern),
			maxHistory: 10000,
		},
		correlator: &AlertCorrelator{
			timeWindow:   5 * time.Minute,
			correlations: make(map[string][]string),
		},
		suppressor: &NoiseSuppressor{
			thresholds: map[string]float64{
				"info":     0.8,
				"warning":  0.6,
				"critical": 0.3,
			},
			rules: getDefaultSuppressionRules(),
		},
	}
}

// ProcessAlert intelligently processes an alert
func (m *SmartAlertManager) ProcessAlert(ctx context.Context, basicAlert Alert) (*SmartAlert, error) {
	// Convert to smart alert
	smartAlert := SmartAlert{
		ID:        basicAlert.ID,
		Name:      basicAlert.Name,
		Severity:  string(basicAlert.Severity),
		Message:   basicAlert.Message,
		Resource:  basicAlert.Source,
		Timestamp: basicAlert.Timestamp,
	}

	// AI analysis for root cause
	rootCause, impact := m.analyzeRootCause(ctx, smartAlert)
	smartAlert.RootCause = rootCause
	smartAlert.Impact = impact

	// Find correlated alerts
	smartAlert.Correlation = m.correlator.findCorrelations(smartAlert, m.alertHistory.alerts)

	// Calculate noise score
	smartAlert.NoiseScore = m.calculateNoiseScore(smartAlert)

	// Determine if should suppress
	smartAlert.Suppressed = m.suppressor.shouldSuppress(smartAlert, m.alertHistory.alerts)

	// Calculate priority
	smartAlert.Priority = m.calculatePriority(smartAlert)

	// Get remediation suggestion
	if remediation, ttResolve := m.suggestRemediation(ctx, smartAlert); remediation != "" {
		smartAlert.Remediation = remediation
		smartAlert.TimeToResolve = ttResolve
	}

	// Check if can auto-resolve
	smartAlert.AutoResolve = m.canAutoResolve(smartAlert)

	// Update history and patterns
	m.updateHistory(smartAlert)
	m.updatePatterns(smartAlert)

	klog.V(2).Infof("Processed smart alert: %s, priority: %d, noise: %.2f, suppressed: %v",
		smartAlert.Name, smartAlert.Priority, smartAlert.NoiseScore, smartAlert.Suppressed)

	return &smartAlert, nil
}

// GetAlertInsights provides AI insights on alert patterns
func (m *SmartAlertManager) GetAlertInsights(ctx context.Context) (*AlertInsights, error) {
	patterns := m.identifyPatterns()
	predictions := m.predictFutureAlerts(ctx)
	recommendations := m.generateRecommendations(ctx, patterns)

	return &AlertInsights{
		TopPatterns:     patterns,
		Predictions:     predictions,
		Recommendations: recommendations,
		NoiseReduction:  m.calculateNoiseReduction(),
		AlertVolume:     m.getAlertVolumeStats(),
	}, nil
}

// analyzeRootCause uses AI to determine root cause
func (m *SmartAlertManager) analyzeRootCause(ctx context.Context, alert SmartAlert) (string, string) {
	// Get recent alerts for context
	recentAlerts := m.getRecentAlerts(10 * time.Minute)

	prompt := fmt.Sprintf(`Analyze this Kubernetes alert for root cause:

Alert: %s
Message: %s
Resource: %s

Recent related alerts:
%+v

Determine:
1. Root cause (not symptoms)
2. Business impact
3. Blast radius

Be specific and actionable.`,
		alert.Name, alert.Message, alert.Resource, recentAlerts)

	request := AnalysisRequest{
		Type:    AnalysisTypeRootCause,
		Context: prompt,
		Data: map[string]interface{}{
			"alert":         alert,
			"recent_alerts": recentAlerts,
		},
		Timestamp: time.Now(),
	}

	response, err := m.client.Analyze(ctx, request)
	if err != nil {
		klog.Errorf("Root cause analysis failed: %v", err)
		return "Unknown", "Unknown impact"
	}

	return response.Summary, response.Diagnosis
}

// calculateNoiseScore determines if alert is noise
func (m *SmartAlertManager) calculateNoiseScore(alert SmartAlert) float64 {
	score := 0.0
	factors := 0

	// Factor 1: Frequency
	pattern := m.alertHistory.patterns[alert.Name]
	if pattern != nil {
		if pattern.Frequency < 5*time.Minute {
			score += 0.8
			factors++
		}
	}

	// Factor 2: Auto-resolved quickly
	similarAlerts := m.findSimilarAlerts(alert, 24*time.Hour)
	autoResolved := 0
	for _, similar := range similarAlerts {
		if similar.AutoResolve {
			autoResolved++
		}
	}
	if len(similarAlerts) > 0 {
		score += float64(autoResolved) / float64(len(similarAlerts))
		factors++
	}

	// Factor 3: Low impact
	if alert.Severity == "info" || alert.Severity == "warning" {
		score += 0.5
		factors++
	}

	// Factor 4: Known transient issues
	if m.isTransientIssue(alert) {
		score += 0.9
		factors++
	}

	if factors == 0 {
		return 0.5 // neutral
	}

	return score / float64(factors)
}

// calculatePriority determines alert priority
func (m *SmartAlertManager) calculatePriority(alert SmartAlert) int {
	priority := 50 // baseline

	// Severity impact
	switch alert.Severity {
	case "critical":
		priority += 40
	case "error":
		priority += 30
	case "warning":
		priority += 10
	}

	// Noise reduction
	priority -= int(alert.NoiseScore * 30)

	// Correlation boost
	priority += len(alert.Correlation) * 5

	// Pattern matching
	if pattern := m.alertHistory.patterns[alert.Name]; pattern != nil {
		if pattern.Occurrences > 10 {
			priority -= 10 // Frequent alerts are less urgent
		}
	}

	// Clamp between 1-100
	if priority < 1 {
		priority = 1
	}
	if priority > 100 {
		priority = 100
	}

	return priority
}

// suggestRemediation provides AI remediation suggestions
func (m *SmartAlertManager) suggestRemediation(ctx context.Context, alert SmartAlert) (string, string) {
	prompt := fmt.Sprintf(`Suggest remediation for this alert:

Alert: %s
Root Cause: %s
Impact: %s

Provide:
1. Specific remediation steps
2. Estimated time to resolve
3. Preventive measures

Be concise and actionable.`,
		alert.Name, alert.RootCause, alert.Impact)

	request := AnalysisRequest{
		Type:    AnalysisTypeHealing,
		Context: prompt,
		Data: map[string]interface{}{
			"alert": alert,
		},
		Timestamp: time.Now(),
	}

	response, err := m.client.Analyze(ctx, request)
	if err != nil {
		return "", ""
	}

	// Extract time estimate
	timeEstimate := "Unknown"
	if strings.Contains(response.Diagnosis, "minutes") {
		timeEstimate = "5-10 minutes"
	} else if strings.Contains(response.Diagnosis, "immediate") {
		timeEstimate = "< 1 minute"
	}

	return response.Summary, timeEstimate
}

// Helper methods

func (m *SmartAlertManager) updateHistory(alert SmartAlert) {
	m.alertHistory.alerts = append(m.alertHistory.alerts, alert)

	// Trim history
	if len(m.alertHistory.alerts) > m.alertHistory.maxHistory {
		m.alertHistory.alerts = m.alertHistory.alerts[1:]
	}
}

func (m *SmartAlertManager) updatePatterns(alert SmartAlert) {
	pattern, exists := m.alertHistory.patterns[alert.Name]
	if !exists {
		pattern = &AlertPattern{
			ID:   alert.Name,
			Name: alert.Name,
		}
		m.alertHistory.patterns[alert.Name] = pattern
	}

	// Update pattern
	if pattern.LastSeen.IsZero() {
		pattern.Frequency = 24 * time.Hour
	} else {
		pattern.Frequency = alert.Timestamp.Sub(pattern.LastSeen)
	}

	pattern.Occurrences++
	pattern.LastSeen = alert.Timestamp
	pattern.Correlated = alert.Correlation
}

func (m *SmartAlertManager) getRecentAlerts(window time.Duration) []SmartAlert {
	cutoff := time.Now().Add(-window)
	recent := []SmartAlert{}

	for i := len(m.alertHistory.alerts) - 1; i >= 0; i-- {
		if m.alertHistory.alerts[i].Timestamp.After(cutoff) {
			recent = append(recent, m.alertHistory.alerts[i])
		} else {
			break
		}
	}

	return recent
}

func (m *SmartAlertManager) findSimilarAlerts(alert SmartAlert, window time.Duration) []SmartAlert {
	similar := []SmartAlert{}
	cutoff := time.Now().Add(-window)

	for _, historical := range m.alertHistory.alerts {
		if historical.Timestamp.After(cutoff) &&
			historical.Name == alert.Name &&
			historical.Resource == alert.Resource {
			similar = append(similar, historical)
		}
	}

	return similar
}

func (m *SmartAlertManager) isTransientIssue(alert SmartAlert) bool {
	transientPatterns := []string{
		"connection reset",
		"timeout",
		"temporary failure",
		"being terminated",
		"is starting",
	}

	message := strings.ToLower(alert.Message)
	for _, pattern := range transientPatterns {
		if strings.Contains(message, pattern) {
			return true
		}
	}

	return false
}

func (m *SmartAlertManager) canAutoResolve(alert SmartAlert) bool {
	// Check if similar alerts auto-resolved
	similar := m.findSimilarAlerts(alert, 7*24*time.Hour)
	if len(similar) < 3 {
		return false
	}

	autoResolved := 0
	for _, s := range similar {
		if s.AutoResolve {
			autoResolved++
		}
	}

	return float64(autoResolved)/float64(len(similar)) > 0.8
}

func (m *SmartAlertManager) identifyPatterns() []AlertPattern {
	patterns := []AlertPattern{}

	for _, pattern := range m.alertHistory.patterns {
		if pattern.Occurrences > 5 {
			patterns = append(patterns, *pattern)
		}
	}

	// Sort by frequency
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency < patterns[j].Frequency
	})

	return patterns
}

func (m *SmartAlertManager) predictFutureAlerts(ctx context.Context) []AlertPrediction {
	// Placeholder for ML prediction
	return []AlertPrediction{
		{
			AlertType:   "Memory Pressure",
			Probability: 0.75,
			TimeWindow:  "2-4 hours",
			Prevention:  "Scale nodes or optimize pod resources",
		},
	}
}

func (m *SmartAlertManager) generateRecommendations(ctx context.Context, patterns []AlertPattern) []string {
	recommendations := []string{}

	for _, pattern := range patterns {
		if pattern.Frequency < 10*time.Minute {
			recommendations = append(recommendations,
				fmt.Sprintf("Alert '%s' fires too frequently (every %v). Consider adjusting thresholds.",
					pattern.Name, pattern.Frequency))
		}
	}

	return recommendations
}

func (m *SmartAlertManager) calculateNoiseReduction() float64 {
	if len(m.alertHistory.alerts) == 0 {
		return 0
	}

	suppressed := 0
	for _, alert := range m.getRecentAlerts(24 * time.Hour) {
		if alert.Suppressed {
			suppressed++
		}
	}

	return float64(suppressed) / float64(len(m.alertHistory.alerts))
}

func (m *SmartAlertManager) getAlertVolumeStats() map[string]int {
	stats := make(map[string]int)

	for _, alert := range m.getRecentAlerts(24 * time.Hour) {
		stats[alert.Severity]++
	}

	return stats
}

// Supporting types

type AlertInsights struct {
	TopPatterns     []AlertPattern    `json:"top_patterns"`
	Predictions     []AlertPrediction `json:"predictions"`
	Recommendations []string          `json:"recommendations"`
	NoiseReduction  float64           `json:"noise_reduction_rate"`
	AlertVolume     map[string]int    `json:"alert_volume_by_severity"`
}

type AlertPrediction struct {
	AlertType   string  `json:"alert_type"`
	Probability float64 `json:"probability"`
	TimeWindow  string  `json:"time_window"`
	Prevention  string  `json:"prevention"`
}

// Default suppression rules
func getDefaultSuppressionRules() []SuppressionRule {
	return []SuppressionRule{
		{
			Name: "Duplicate suppression",
			Condition: func(alert SmartAlert, history []SmartAlert) bool {
				// Suppress if same alert fired in last minute
				for i := len(history) - 1; i >= 0; i-- {
					if history[i].Name == alert.Name &&
						history[i].Resource == alert.Resource &&
						time.Since(history[i].Timestamp) < time.Minute {
						return true
					}
				}
				return false
			},
			Action: "suppress",
		},
		{
			Name: "High noise suppression",
			Condition: func(alert SmartAlert, history []SmartAlert) bool {
				return alert.NoiseScore > 0.8
			},
			Action: "suppress",
		},
	}
}

// NoiseSuppressor methods
func (n *NoiseSuppressor) shouldSuppress(alert SmartAlert, history []SmartAlert) bool {
	// Check rules
	for _, rule := range n.rules {
		if rule.Condition(alert, history) {
			return true
		}
	}

	// Check threshold
	threshold, exists := n.thresholds[alert.Severity]
	if exists && alert.NoiseScore > threshold {
		return true
	}

	return false
}

// AlertCorrelator methods
func (c *AlertCorrelator) findCorrelations(alert SmartAlert, history []SmartAlert) []string {
	correlated := []string{}
	cutoff := alert.Timestamp.Add(-c.timeWindow)

	for _, historical := range history {
		if historical.Timestamp.After(cutoff) &&
			historical.ID != alert.ID &&
			c.areCorrelated(alert, historical) {
			correlated = append(correlated, historical.ID)
		}
	}

	return correlated
}

func (c *AlertCorrelator) areCorrelated(a1, a2 SmartAlert) bool {
	// Same resource
	if a1.Resource == a2.Resource {
		return true
	}

	// Similar root cause
	if a1.RootCause != "" && a1.RootCause == a2.RootCause {
		return true
	}

	// Time proximity
	timeDiff := math.Abs(a1.Timestamp.Sub(a2.Timestamp).Seconds())
	if timeDiff < 30 {
		return true
	}

	return false
}
