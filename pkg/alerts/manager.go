package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager handles alert routing, deduplication, and persistence
type Manager struct {
	channels   map[string]NotificationChannel
	rules      []AlertRule
	silences   map[string]time.Time
	history    []Alert
	mu         sync.RWMutex
	maxHistory int
}

// NotificationChannel interface for alert delivery
type NotificationChannel interface {
	Send(ctx context.Context, alert Alert) error
	Name() string
}

// AlertRule defines when and how to generate alerts
type AlertRule struct {
	Name      string
	Condition func(CheckResult) bool
	Severity  AlertSeverity
	Cooldown  time.Duration
	LastFired time.Time
	Channel   string
	Template  string
}

// NewManager creates a new alert manager
func NewManager() *Manager {
	return &Manager{
		channels:   make(map[string]NotificationChannel),
		rules:      make([]AlertRule, 0),
		silences:   make(map[string]time.Time),
		history:    make([]Alert, 0),
		maxHistory: 1000,
	}
}

// RegisterChannel adds a notification channel
func (m *Manager) RegisterChannel(channel NotificationChannel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[channel.Name()] = channel
}

// AddRule adds an alert rule
func (m *Manager) AddRule(rule AlertRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = append(m.rules, rule)
}

// ProcessCheckResult processes a check result and generates alerts
func (m *Manager) ProcessCheckResult(ctx context.Context, result CheckResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, rule := range m.rules {
		if rule.Condition(result) && m.shouldFire(rule) {
			alert := Alert{
				ID:          fmt.Sprintf("%s-%d", rule.Name, time.Now().Unix()),
				Name:        rule.Name,
				Severity:    rule.Severity,
				Message:     m.formatMessage(rule.Template, result),
				Source:      "kubepulse",
				Timestamp:   time.Now(),
				Fingerprint: m.generateFingerprint(rule.Name, result),
				Status:      AlertStatusFiring,
				Labels: map[string]string{
					"check":    result.Name,
					"rule":     rule.Name,
					"severity": string(rule.Severity),
				},
			}

			// Check if silenced
			if !m.isSilenced(alert.Fingerprint) {
				if err := m.sendAlert(ctx, alert, rule.Channel); err != nil {
					return fmt.Errorf("failed to send alert: %w", err)
				}
			}

			// Update rule
			m.rules[i].LastFired = time.Now()

			// Store in history
			m.addToHistory(alert)
		}
	}

	return nil
}

// SilenceAlert silences an alert for a duration
func (m *Manager) SilenceAlert(fingerprint string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.silences[fingerprint] = time.Now().Add(duration)
}

// GetHistory returns alert history
func (m *Manager) GetHistory(limit int) []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.history) {
		limit = len(m.history)
	}

	// Return most recent alerts
	start := len(m.history) - limit
	return m.history[start:]
}

// shouldFire checks if a rule should fire based on cooldown
func (m *Manager) shouldFire(rule AlertRule) bool {
	return time.Since(rule.LastFired) >= rule.Cooldown
}

// isSilenced checks if an alert is currently silenced
func (m *Manager) isSilenced(fingerprint string) bool {
	silencedUntil, exists := m.silences[fingerprint]
	if !exists {
		return false
	}

	if time.Now().After(silencedUntil) {
		delete(m.silences, fingerprint)
		return false
	}

	return true
}

// sendAlert sends an alert through the specified channel
func (m *Manager) sendAlert(ctx context.Context, alert Alert, channelName string) error {
	channel, exists := m.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	return channel.Send(ctx, alert)
}

// addToHistory adds an alert to history with size limit
func (m *Manager) addToHistory(alert Alert) {
	m.history = append(m.history, alert)

	if len(m.history) > m.maxHistory {
		m.history = m.history[len(m.history)-m.maxHistory:]
	}
}

// formatMessage formats alert message using template
func (m *Manager) formatMessage(template string, result CheckResult) string {
	if template == "" {
		return result.Message
	}

	// Simple template replacement (would use proper templating in production)
	message := template
	message = fmt.Sprintf(message, result.Name, result.Status, result.Message)
	return message
}

// generateFingerprint creates a unique fingerprint for alert deduplication
func (m *Manager) generateFingerprint(ruleName string, result CheckResult) string {
	return fmt.Sprintf("%s-%s", ruleName, result.Name)
}

// LogChannel is a simple logging notification channel
type LogChannel struct{}

// NewLogChannel creates a new log channel
func NewLogChannel() *LogChannel {
	return &LogChannel{}
}

// Name returns the channel name
func (l *LogChannel) Name() string {
	return "log"
}

// Send logs the alert
func (l *LogChannel) Send(ctx context.Context, alert Alert) error {
	fmt.Printf("[ALERT] %s: %s - %s\n", alert.Severity, alert.Name, alert.Message)
	return nil
}

// CreateDefaultRules creates default alert rules
func CreateDefaultRules() []AlertRule {
	return []AlertRule{
		{
			Name: "pod-health-critical",
			Condition: func(result CheckResult) bool {
				return result.Name == "pod-health" && result.Status == HealthStatusUnhealthy
			},
			Severity: AlertSeverityCritical,
			Cooldown: 5 * time.Minute,
			Channel:  "log",
			Template: "Critical pod health issue: %s",
		},
		{
			Name: "node-health-critical",
			Condition: func(result CheckResult) bool {
				return result.Name == "node-health" && result.Status == HealthStatusUnhealthy
			},
			Severity: AlertSeverityCritical,
			Cooldown: 5 * time.Minute,
			Channel:  "log",
			Template: "Critical node health issue: %s",
		},
		{
			Name: "pod-health-warning",
			Condition: func(result CheckResult) bool {
				return result.Name == "pod-health" && result.Status == HealthStatusDegraded
			},
			Severity: AlertSeverityWarning,
			Cooldown: 10 * time.Minute,
			Channel:  "log",
			Template: "Pod health degraded: %s",
		},
	}
}
