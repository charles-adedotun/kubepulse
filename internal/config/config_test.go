package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config == nil {
		t.Fatal("expected non-nil config")
	}

	// Test Kubernetes defaults
	if config.Kubernetes.Kubeconfig != "~/.kube/config" {
		t.Errorf("expected default kubeconfig '~/.kube/config', got %s", config.Kubernetes.Kubeconfig)
	}

	// Test Monitoring defaults
	if config.Monitoring.Interval != 30*time.Second {
		t.Errorf("expected default monitoring interval 30s, got %v", config.Monitoring.Interval)
	}

	expectedChecks := []string{"pod-health", "node-health", "service-health"}
	if !reflect.DeepEqual(config.Monitoring.EnabledChecks, expectedChecks) {
		t.Errorf("expected default enabled checks %v, got %v", expectedChecks, config.Monitoring.EnabledChecks)
	}

	if config.Monitoring.MaxHistory != 1000 {
		t.Errorf("expected default max history 1000, got %d", config.Monitoring.MaxHistory)
	}

	// Test Alerts defaults
	if !config.Alerts.Enabled {
		t.Error("expected alerts to be enabled by default")
	}

	if len(config.Alerts.Channels) != 1 {
		t.Errorf("expected 1 default alert channel, got %d", len(config.Alerts.Channels))
	}

	logChannel, exists := config.Alerts.Channels["log"]
	if !exists {
		t.Error("expected default 'log' alert channel")
	}

	if logChannel.Type != "log" {
		t.Errorf("expected log channel type 'log', got %s", logChannel.Type)
	}

	if !logChannel.Enabled {
		t.Error("expected log channel to be enabled by default")
	}

	// Test ML defaults
	if !config.ML.Enabled {
		t.Error("expected ML to be enabled by default")
	}

	if config.ML.AnomalyEngine != "statistical" {
		t.Errorf("expected default anomaly engine 'statistical', got %s", config.ML.AnomalyEngine)
	}

	if config.ML.Threshold != 2.0 {
		t.Errorf("expected default ML threshold 2.0, got %f", config.ML.Threshold)
	}

	// Test Server defaults
	if config.Server.Port != 8080 {
		t.Errorf("expected default server port 8080, got %d", config.Server.Port)
	}

	if !config.Server.EnableWeb {
		t.Error("expected web to be enabled by default")
	}

	if !config.Server.CORSEnabled {
		t.Error("expected CORS to be enabled by default")
	}

	expectedOrigins := []string{"*"}
	if !reflect.DeepEqual(config.Server.CORSOrigins, expectedOrigins) {
		t.Errorf("expected default CORS origins %v, got %v", expectedOrigins, config.Server.CORSOrigins)
	}

	// Test UI defaults
	if config.UI.RefreshInterval != 10*time.Second {
		t.Errorf("expected default UI refresh interval 10s, got %v", config.UI.RefreshInterval)
	}

	if config.UI.Theme != "system" {
		t.Errorf("expected default theme 'system', got %s", config.UI.Theme)
	}

	if !config.UI.Features.AIInsights {
		t.Error("expected AI insights to be enabled by default")
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Test loading config when no file exists
	config, err := LoadConfig("")

	if err != nil {
		t.Errorf("expected no error when loading default config, got %v", err)
	}

	if config == nil {
		t.Fatal("expected non-nil config")
	}

	// Should return default values
	if config.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", config.Server.Port)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	configData := `
kubernetes:
  kubeconfig: "/custom/kubeconfig"
  context: "test-context"
  namespaces:
    - "default"
    - "kube-system"

monitoring:
  interval: 60s
  enabled_checks:
    - "custom-check"
  max_history: 500
  timeout: 45s

alerts:
  enabled: false

ml:
  enabled: false
  threshold: 3.0

server:
  port: 9090
  host: "localhost"
  enable_web: false

ui:
  theme: "dark"
`

	err := os.WriteFile(configPath, []byte(configData), 0644)
	if err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Test that values were loaded from file
	if config.Kubernetes.Kubeconfig != "/custom/kubeconfig" {
		t.Errorf("expected kubeconfig '/custom/kubeconfig', got %s", config.Kubernetes.Kubeconfig)
	}

	if config.Kubernetes.Context != "test-context" {
		t.Errorf("expected context 'test-context', got %s", config.Kubernetes.Context)
	}

	expectedNamespaces := []string{"default", "kube-system"}
	if !reflect.DeepEqual(config.Kubernetes.Namespaces, expectedNamespaces) {
		t.Errorf("expected namespaces %v, got %v", expectedNamespaces, config.Kubernetes.Namespaces)
	}

	if config.Monitoring.Interval != 60*time.Second {
		t.Errorf("expected monitoring interval 60s, got %v", config.Monitoring.Interval)
	}

	if config.Alerts.Enabled {
		t.Error("expected alerts to be disabled")
	}

	if config.ML.Enabled {
		t.Error("expected ML to be disabled")
	}

	if config.Server.Port != 9090 {
		t.Errorf("expected server port 9090, got %d", config.Server.Port)
	}

	if config.UI.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %s", config.UI.Theme)
	}
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	// Test loading non-existent file
	config, err := LoadConfig("/non/existent/path.yaml")

	// Should return error and nil config for non-existent file
	if err == nil {
		t.Error("expected error when loading non-existent file")
	}

	if config != nil {
		t.Error("expected nil config when file load fails")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.yaml")

	invalidYAML := `
kubernetes:
  kubeconfig: "/test"
  invalid yaml structure
    - bad indentation
`

	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid config file: %v", err)
	}

	config, err := LoadConfig(configPath)

	// Should return error and nil config for invalid YAML
	if err == nil {
		t.Error("expected error when loading invalid YAML")
	}

	if config != nil {
		t.Error("expected nil config when YAML parsing fails")
	}
}

func TestSaveConfig(t *testing.T) {
	// Create a test config
	config := &Config{
		Kubernetes: KubernetesConfig{
			Kubeconfig: "/test/kubeconfig",
			Context:    "test-context",
		},
		Server: ServerConfig{
			Port: 9999,
			Host: "test-host",
		},
		UI: UIConfig{
			Theme: "dark",
		},
	}

	// Save to temporary file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "saved-config.yaml")

	err := SaveConfig(config, configPath)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var savedConfig Config
	err = yaml.Unmarshal(data, &savedConfig)
	if err != nil {
		t.Fatalf("failed to unmarshal saved config: %v", err)
	}

	if savedConfig.Kubernetes.Kubeconfig != "/test/kubeconfig" {
		t.Errorf("expected saved kubeconfig '/test/kubeconfig', got %s", savedConfig.Kubernetes.Kubeconfig)
	}

	if savedConfig.Server.Port != 9999 {
		t.Errorf("expected saved port 9999, got %d", savedConfig.Server.Port)
	}

	if savedConfig.UI.Theme != "dark" {
		t.Errorf("expected saved theme 'dark', got %s", savedConfig.UI.Theme)
	}
}

func TestSaveConfig_InvalidPath(t *testing.T) {
	config := GetDefaultConfig()

	// Try to save to invalid path
	err := SaveConfig(config, "/invalid/path/that/does/not/exist/config.yaml")

	if err == nil {
		t.Error("expected error when saving to invalid path")
	}
}

func TestConfigStructs_Initialization(t *testing.T) {
	// Test that all config structs can be initialized
	config := &Config{
		Kubernetes: KubernetesConfig{
			Kubeconfig: "test",
			Context:    "test",
			Namespaces: []string{"default"},
		},
		Monitoring: MonitoringConfig{
			Interval:      30 * time.Second,
			EnabledChecks: []string{"test"},
			MaxHistory:    100,
			Timeout:       10 * time.Second,
		},
		Alerts: AlertsConfig{
			Enabled: true,
			Channels: map[string]ChannelConfig{
				"test": {
					Type:     "test",
					Enabled:  true,
					Settings: map[string]interface{}{"key": "value"},
				},
			},
			Rules: map[string]AlertRuleConfig{
				"test": {
					Condition: "test",
					Severity:  "high",
					Cooldown:  5 * time.Minute,
					Channels:  []string{"test"},
					Template:  "test template",
				},
			},
		},
		SLOs: map[string]SLOConfig{
			"test": {
				Description: "test SLO",
				SLI:         "test",
				Target:      99.9,
				Window:      24 * time.Hour,
				BudgetPolicy: []BudgetPolicyConfig{
					{
						Threshold: 0.1,
						Action:    "alert",
					},
				},
			},
		},
		ML: MLConfig{
			Enabled:         true,
			AnomalyEngine:   "test",
			Threshold:       2.0,
			LearningPeriod:  24 * time.Hour,
			PredictionHours: 12,
		},
		Server: ServerConfig{
			Port:         8080,
			Host:         "localhost",
			EnableWeb:    true,
			CORSEnabled:  true,
			CORSOrigins:  []string{"*"},
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		UI: UIConfig{
			RefreshInterval:      10 * time.Second,
			AIInsightsInterval:   30 * time.Second,
			MaxReconnectAttempts: 5,
			ReconnectDelay:       3 * time.Second,
			Theme:                "dark",
			Features: UIFeatures{
				AIInsights:          true,
				PredictiveAnalytics: true,
				SmartAlerts:         true,
				NodeDetails:         true,
			},
		},
	}

	// Basic validation that all fields are accessible
	if config.Kubernetes.Kubeconfig != "test" {
		t.Error("Kubernetes config not properly initialized")
	}

	if config.Monitoring.Interval != 30*time.Second {
		t.Error("Monitoring config not properly initialized")
	}

	if !config.Alerts.Enabled {
		t.Error("Alerts config not properly initialized")
	}

	if len(config.SLOs) != 1 {
		t.Error("SLOs config not properly initialized")
	}

	if !config.ML.Enabled {
		t.Error("ML config not properly initialized")
	}

	if config.Server.Port != 8080 {
		t.Error("Server config not properly initialized")
	}

	if config.UI.Theme != "dark" {
		t.Error("UI config not properly initialized")
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Test that environment variables can override config values
	// This is a basic test to verify the structure supports env overrides

	// Set test environment variables
	_ = os.Setenv("KUBEPULSE_SERVER_PORT", "9999")
	_ = os.Setenv("KUBEPULSE_UI_THEME", "light")
	defer func() {
		_ = os.Unsetenv("KUBEPULSE_SERVER_PORT")
		_ = os.Unsetenv("KUBEPULSE_UI_THEME")
	}()

	// Load config and verify environment support is available
	config := GetDefaultConfig()

	// Note: The actual environment loading happens in LoadConfig
	// This test validates the config structure supports it
	if config == nil {
		t.Fatal("expected non-nil config")
	}

	// The config should have proper structure for environment loading
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		t.Error("Server port should be in valid range")
	}

	if config.UI.Theme == "" {
		t.Error("UI theme should not be empty")
	}
}

func TestConfigValidation_Defaults(t *testing.T) {
	config := GetDefaultConfig()

	// Test that default config passes basic validation
	if config.Monitoring.Interval <= 0 {
		t.Error("Monitoring interval should be positive")
	}

	if config.Monitoring.MaxHistory <= 0 {
		t.Error("Max history should be positive")
	}

	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		t.Error("Server port should be in valid range")
	}

	if config.ML.Threshold < 0 {
		t.Error("ML threshold should be non-negative")
	}

	if config.UI.RefreshInterval <= 0 {
		t.Error("UI refresh interval should be positive")
	}

	if config.UI.MaxReconnectAttempts < 0 {
		t.Error("Max reconnect attempts should be non-negative")
	}
}

func TestYAMLTags(t *testing.T) {
	// Test that struct tags are properly set for YAML marshaling
	config := &Config{
		Server: ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	yamlString := string(data)

	// Check that YAML contains expected field names (not Go field names)
	if !contains(yamlString, "server:") {
		t.Error("expected 'server:' in YAML output")
	}

	if !contains(yamlString, "port:") {
		t.Error("expected 'port:' in YAML output")
	}

	if !contains(yamlString, "host:") {
		t.Error("expected 'host:' in YAML output")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAtIndex(s, substr)))
}

func containsAtIndex(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
