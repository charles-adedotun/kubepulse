package alerts

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockNotificationChannel implements NotificationChannel for testing
type mockNotificationChannel struct {
	name      string
	sentAlert *Alert
	sendError error
}

func (m *mockNotificationChannel) Send(ctx context.Context, alert Alert) error {
	m.sentAlert = &alert
	return m.sendError
}

func (m *mockNotificationChannel) Name() string {
	return m.name
}

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.channels == nil {
		t.Fatal("expected initialized channels map")
	}

	if manager.rules == nil {
		t.Fatal("expected initialized rules slice")
	}

	if manager.silences == nil {
		t.Fatal("expected initialized silences map")
	}

	if manager.history == nil {
		t.Fatal("expected initialized history slice")
	}

	if manager.maxHistory != 1000 {
		t.Errorf("expected maxHistory 1000, got %d", manager.maxHistory)
	}
}

func TestManager_RegisterChannel(t *testing.T) {
	manager := NewManager()
	channel := &mockNotificationChannel{name: "test-channel"}

	manager.RegisterChannel(channel)

	if len(manager.channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(manager.channels))
	}

	retrievedChannel := manager.channels["test-channel"]
	if retrievedChannel != channel {
		t.Error("retrieved channel is not the same instance as registered")
	}
}

func TestManager_AddRule(t *testing.T) {
	manager := NewManager()
	rule := AlertRule{
		Name:     "test-rule",
		Severity: AlertSeverityCritical,
		Cooldown: 5 * time.Minute,
		Channel:  "test-channel",
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}

	manager.AddRule(rule)

	if len(manager.rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(manager.rules))
	}

	if manager.rules[0].Name != "test-rule" {
		t.Errorf("expected rule name 'test-rule', got %s", manager.rules[0].Name)
	}
}

func TestManager_ProcessCheckResult(t *testing.T) {
	manager := NewManager()
	channel := &mockNotificationChannel{name: "test-channel"}
	manager.RegisterChannel(channel)

	rule := AlertRule{
		Name:     "test-rule",
		Severity: AlertSeverityCritical,
		Cooldown: 5 * time.Minute,
		Channel:  "test-channel",
		Template: "Test alert: %s",
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}
	manager.AddRule(rule)

	// Test result that should trigger alert
	result := CheckResult{
		Name:      "test-check",
		Status:    HealthStatusUnhealthy,
		Message:   "Test failure",
		Timestamp: time.Now(),
	}

	err := manager.ProcessCheckResult(context.Background(), result)
	if err != nil {
		t.Errorf("unexpected error processing check result: %v", err)
	}

	// Verify alert was sent
	if channel.sentAlert == nil {
		t.Fatal("expected alert to be sent")
	}

	if channel.sentAlert.Name != "test-rule" {
		t.Errorf("expected alert name 'test-rule', got %s", channel.sentAlert.Name)
	}

	if channel.sentAlert.Severity != AlertSeverityCritical {
		t.Errorf("expected alert severity critical, got %s", channel.sentAlert.Severity)
	}

	if channel.sentAlert.Status != AlertStatusFiring {
		t.Errorf("expected alert status firing, got %s", channel.sentAlert.Status)
	}

	// Verify alert is in history
	history := manager.GetHistory(10)
	if len(history) != 1 {
		t.Errorf("expected 1 alert in history, got %d", len(history))
	}
}

func TestManager_ProcessCheckResult_NoMatch(t *testing.T) {
	manager := NewManager()
	channel := &mockNotificationChannel{name: "test-channel"}
	manager.RegisterChannel(channel)

	rule := AlertRule{
		Name:     "test-rule",
		Severity: AlertSeverityCritical,
		Cooldown: 5 * time.Minute,
		Channel:  "test-channel",
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}
	manager.AddRule(rule)

	// Test result that should NOT trigger alert
	result := CheckResult{
		Name:      "test-check",
		Status:    HealthStatusHealthy,
		Message:   "All good",
		Timestamp: time.Now(),
	}

	err := manager.ProcessCheckResult(context.Background(), result)
	if err != nil {
		t.Errorf("unexpected error processing check result: %v", err)
	}

	// Verify no alert was sent
	if channel.sentAlert != nil {
		t.Error("expected no alert to be sent")
	}

	// Verify no alert in history
	history := manager.GetHistory(10)
	if len(history) != 0 {
		t.Errorf("expected 0 alerts in history, got %d", len(history))
	}
}

func TestManager_ProcessCheckResult_Cooldown(t *testing.T) {
	manager := NewManager()
	channel := &mockNotificationChannel{name: "test-channel"}
	manager.RegisterChannel(channel)

	rule := AlertRule{
		Name:      "test-rule",
		Severity:  AlertSeverityCritical,
		Cooldown:  5 * time.Minute,
		Channel:   "test-channel",
		LastFired: time.Now(), // Just fired
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}
	manager.AddRule(rule)

	result := CheckResult{
		Name:      "test-check",
		Status:    HealthStatusUnhealthy,
		Message:   "Test failure",
		Timestamp: time.Now(),
	}

	err := manager.ProcessCheckResult(context.Background(), result)
	if err != nil {
		t.Errorf("unexpected error processing check result: %v", err)
	}

	// Verify no alert was sent due to cooldown
	if channel.sentAlert != nil {
		t.Error("expected no alert to be sent due to cooldown")
	}
}

func TestManager_ProcessCheckResult_ChannelError(t *testing.T) {
	manager := NewManager()
	channel := &mockNotificationChannel{
		name:      "test-channel",
		sendError: fmt.Errorf("channel send error"),
	}
	manager.RegisterChannel(channel)

	rule := AlertRule{
		Name:     "test-rule",
		Severity: AlertSeverityCritical,
		Cooldown: 5 * time.Minute,
		Channel:  "test-channel",
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}
	manager.AddRule(rule)

	result := CheckResult{
		Name:      "test-check",
		Status:    HealthStatusUnhealthy,
		Message:   "Test failure",
		Timestamp: time.Now(),
	}

	err := manager.ProcessCheckResult(context.Background(), result)
	if err == nil {
		t.Error("expected error when channel send fails")
	}

	if err.Error() != "failed to send alert: channel send error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestManager_ProcessCheckResult_MissingChannel(t *testing.T) {
	manager := NewManager()

	rule := AlertRule{
		Name:     "test-rule",
		Severity: AlertSeverityCritical,
		Cooldown: 5 * time.Minute,
		Channel:  "missing-channel",
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}
	manager.AddRule(rule)

	result := CheckResult{
		Name:      "test-check",
		Status:    HealthStatusUnhealthy,
		Message:   "Test failure",
		Timestamp: time.Now(),
	}

	err := manager.ProcessCheckResult(context.Background(), result)
	if err == nil {
		t.Error("expected error when channel is missing")
	}

	if err.Error() != "failed to send alert: channel missing-channel not found" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestManager_SilenceAlert(t *testing.T) {
	manager := NewManager()
	channel := &mockNotificationChannel{name: "test-channel"}
	manager.RegisterChannel(channel)

	rule := AlertRule{
		Name:     "test-rule",
		Severity: AlertSeverityCritical,
		Cooldown: 0, // No cooldown
		Channel:  "test-channel",
		Condition: func(result CheckResult) bool {
			return result.Status == HealthStatusUnhealthy
		},
	}
	manager.AddRule(rule)

	// Silence the alert
	fingerprint := manager.generateFingerprint("test-rule", CheckResult{Name: "test-check"})
	manager.SilenceAlert(fingerprint, 1*time.Hour)

	// Try to trigger alert
	result := CheckResult{
		Name:      "test-check",
		Status:    HealthStatusUnhealthy,
		Message:   "Test failure",
		Timestamp: time.Now(),
	}

	err := manager.ProcessCheckResult(context.Background(), result)
	if err != nil {
		t.Errorf("unexpected error processing check result: %v", err)
	}

	// Verify no alert was sent due to silence
	if channel.sentAlert != nil {
		t.Error("expected no alert to be sent due to silence")
	}

	// Verify alert is still in history (rules fire but don't send)
	history := manager.GetHistory(10)
	if len(history) != 1 {
		t.Errorf("expected 1 alert in history, got %d", len(history))
	}
}

func TestManager_GetHistory(t *testing.T) {
	manager := NewManager()

	// Add some alerts to history manually
	for i := 0; i < 5; i++ {
		alert := Alert{
			ID:        fmt.Sprintf("alert-%d", i),
			Name:      fmt.Sprintf("alert-%d", i),
			Severity:  AlertSeverityInfo,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
		manager.addToHistory(alert)
	}

	// Test getting all history
	history := manager.GetHistory(0)
	if len(history) != 5 {
		t.Errorf("expected 5 alerts in history, got %d", len(history))
	}

	// Test getting limited history
	history = manager.GetHistory(3)
	if len(history) != 3 {
		t.Errorf("expected 3 alerts in limited history, got %d", len(history))
	}

	// Verify we get the most recent alerts
	if history[0].ID != "alert-2" {
		t.Errorf("expected first alert to be alert-2, got %s", history[0].ID)
	}
}

func TestManager_AddToHistory_MaxLimit(t *testing.T) {
	manager := NewManager()
	manager.maxHistory = 3 // Set small limit for testing

	// Add more alerts than the limit
	for i := 0; i < 5; i++ {
		alert := Alert{
			ID:        fmt.Sprintf("alert-%d", i),
			Name:      fmt.Sprintf("alert-%d", i),
			Severity:  AlertSeverityInfo,
			Timestamp: time.Now(),
		}
		manager.addToHistory(alert)
	}

	// Verify history is limited
	if len(manager.history) != 3 {
		t.Errorf("expected history length 3, got %d", len(manager.history))
	}

	// Verify we kept the most recent alerts
	if manager.history[0].ID != "alert-2" {
		t.Errorf("expected first alert to be alert-2, got %s", manager.history[0].ID)
	}
	if manager.history[2].ID != "alert-4" {
		t.Errorf("expected last alert to be alert-4, got %s", manager.history[2].ID)
	}
}

func TestManager_FormatMessage(t *testing.T) {
	manager := NewManager()
	result := CheckResult{
		Name:    "test-check",
		Status:  HealthStatusUnhealthy,
		Message: "Test failure message",
	}

	// Test empty template
	message := manager.formatMessage("", result)
	if message != "Test failure message" {
		t.Errorf("expected original message, got %s", message)
	}

	// Test template formatting
	template := "Alert for %s with status %s: %s"
	message = manager.formatMessage(template, result)
	expected := "Alert for test-check with status unhealthy: Test failure message"
	if message != expected {
		t.Errorf("expected '%s', got '%s'", expected, message)
	}
}

func TestManager_GenerateFingerprint(t *testing.T) {
	manager := NewManager()
	result := CheckResult{Name: "test-check"}

	fingerprint := manager.generateFingerprint("test-rule", result)
	expected := "test-rule-test-check"
	if fingerprint != expected {
		t.Errorf("expected fingerprint '%s', got '%s'", expected, fingerprint)
	}
}

func TestManager_IsSilenced(t *testing.T) {
	manager := NewManager()

	// Test non-existent silence
	if manager.isSilenced("non-existent") {
		t.Error("expected non-existent fingerprint to not be silenced")
	}

	// Test active silence
	fingerprint := "test-fingerprint"
	manager.silences[fingerprint] = time.Now().Add(1 * time.Hour)
	if !manager.isSilenced(fingerprint) {
		t.Error("expected active silence to be detected")
	}

	// Test expired silence
	manager.silences[fingerprint] = time.Now().Add(-1 * time.Hour)
	if manager.isSilenced(fingerprint) {
		t.Error("expected expired silence to be removed")
	}

	// Verify expired silence was cleaned up
	if _, exists := manager.silences[fingerprint]; exists {
		t.Error("expected expired silence to be removed from map")
	}
}

func TestLogChannel(t *testing.T) {
	channel := NewLogChannel()

	if channel == nil {
		t.Fatal("expected non-nil log channel")
	}

	if channel.Name() != "log" {
		t.Errorf("expected channel name 'log', got %s", channel.Name())
	}

	// Test sending an alert (should not error)
	alert := Alert{
		ID:       "test-alert",
		Name:     "test",
		Severity: AlertSeverityInfo,
		Message:  "Test message",
	}

	err := channel.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("unexpected error sending alert: %v", err)
	}
}

func TestCreateDefaultRules(t *testing.T) {
	rules := CreateDefaultRules()

	if len(rules) != 3 {
		t.Errorf("expected 3 default rules, got %d", len(rules))
	}

	// Test pod-health-critical rule
	podCritical := rules[0]
	if podCritical.Name != "pod-health-critical" {
		t.Errorf("expected first rule name 'pod-health-critical', got %s", podCritical.Name)
	}
	if podCritical.Severity != AlertSeverityCritical {
		t.Errorf("expected critical severity, got %s", podCritical.Severity)
	}
	if podCritical.Cooldown != 5*time.Minute {
		t.Errorf("expected 5 minute cooldown, got %v", podCritical.Cooldown)
	}

	// Test condition
	criticalResult := CheckResult{Name: "pod-health", Status: HealthStatusUnhealthy}
	if !podCritical.Condition(criticalResult) {
		t.Error("expected pod-health-critical condition to match unhealthy pod-health")
	}

	healthyResult := CheckResult{Name: "pod-health", Status: HealthStatusHealthy}
	if podCritical.Condition(healthyResult) {
		t.Error("expected pod-health-critical condition to not match healthy pod-health")
	}
}
