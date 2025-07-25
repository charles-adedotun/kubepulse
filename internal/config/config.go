package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Kubernetes settings
	Kubernetes KubernetesConfig `yaml:"kubernetes" mapstructure:"kubernetes"`

	// Monitoring settings
	Monitoring MonitoringConfig `yaml:"monitoring" mapstructure:"monitoring"`

	// Alert settings
	Alerts AlertsConfig `yaml:"alerts" mapstructure:"alerts"`

	// SLO definitions
	SLOs map[string]SLOConfig `yaml:"slos" mapstructure:"slos"`

	// ML settings
	ML MLConfig `yaml:"ml" mapstructure:"ml"`

	// Server settings
	Server ServerConfig `yaml:"server" mapstructure:"server"`

	// UI settings
	UI UIConfig `yaml:"ui" mapstructure:"ui"`

	// AI settings
	AI AIConfig `yaml:"ai" mapstructure:"ai"`
}

// KubernetesConfig holds Kubernetes-related configuration
type KubernetesConfig struct {
	Kubeconfig string   `yaml:"kubeconfig" mapstructure:"kubeconfig"`
	Context    string   `yaml:"context" mapstructure:"context"`
	Namespaces []string `yaml:"namespaces" mapstructure:"namespaces"`
}

// MonitoringConfig holds monitoring-related configuration
type MonitoringConfig struct {
	Interval      time.Duration `yaml:"interval" mapstructure:"interval"`
	EnabledChecks []string      `yaml:"enabled_checks" mapstructure:"enabled_checks"`
	MaxHistory    int           `yaml:"max_history" mapstructure:"max_history"`
	Timeout       time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

// AlertsConfig holds alert-related configuration
type AlertsConfig struct {
	Enabled  bool                       `yaml:"enabled" mapstructure:"enabled"`
	Channels map[string]ChannelConfig   `yaml:"channels" mapstructure:"channels"`
	Rules    map[string]AlertRuleConfig `yaml:"rules" mapstructure:"rules"`
}

// ChannelConfig represents a notification channel configuration
type ChannelConfig struct {
	Type     string                 `yaml:"type" mapstructure:"type"`
	Enabled  bool                   `yaml:"enabled" mapstructure:"enabled"`
	Settings map[string]interface{} `yaml:"settings" mapstructure:"settings"`
}

// AlertRuleConfig represents an alert rule configuration
type AlertRuleConfig struct {
	Condition string        `yaml:"condition" mapstructure:"condition"`
	Severity  string        `yaml:"severity" mapstructure:"severity"`
	Cooldown  time.Duration `yaml:"cooldown" mapstructure:"cooldown"`
	Channels  []string      `yaml:"channels" mapstructure:"channels"`
	Template  string        `yaml:"template" mapstructure:"template"`
}

// SLOConfig represents an SLO configuration
type SLOConfig struct {
	Description  string               `yaml:"description" mapstructure:"description"`
	SLI          string               `yaml:"sli" mapstructure:"sli"`
	Target       float64              `yaml:"target" mapstructure:"target"`
	Window       time.Duration        `yaml:"window" mapstructure:"window"`
	BudgetPolicy []BudgetPolicyConfig `yaml:"budget_policy" mapstructure:"budget_policy"`
}

// BudgetPolicyConfig represents error budget policy configuration
type BudgetPolicyConfig struct {
	Threshold float64 `yaml:"threshold" mapstructure:"threshold"`
	Action    string  `yaml:"action" mapstructure:"action"`
}

// MLConfig holds ML-related configuration
type MLConfig struct {
	Enabled         bool          `yaml:"enabled" mapstructure:"enabled"`
	AnomalyEngine   string        `yaml:"anomaly_engine" mapstructure:"anomaly_engine"`
	Threshold       float64       `yaml:"threshold" mapstructure:"threshold"`
	LearningPeriod  time.Duration `yaml:"learning_period" mapstructure:"learning_period"`
	PredictionHours int           `yaml:"prediction_hours" mapstructure:"prediction_hours"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         int           `yaml:"port" mapstructure:"port"`
	Host         string        `yaml:"host" mapstructure:"host"`
	EnableWeb    bool          `yaml:"enable_web" mapstructure:"enable_web"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
}

// UIConfig holds UI-related configuration
type UIConfig struct {
	RefreshInterval      time.Duration `yaml:"refresh_interval" mapstructure:"refresh_interval"`
	AIInsightsInterval   time.Duration `yaml:"ai_insights_interval" mapstructure:"ai_insights_interval"`
	MaxReconnectAttempts int           `yaml:"max_reconnect_attempts" mapstructure:"max_reconnect_attempts"`
	ReconnectDelay       time.Duration `yaml:"reconnect_delay" mapstructure:"reconnect_delay"`
	Theme                string        `yaml:"theme" mapstructure:"theme"`
	Features             UIFeatures    `yaml:"features" mapstructure:"features"`
}

// UIFeatures holds UI feature flags
type UIFeatures struct {
	AIInsights          bool `yaml:"ai_insights" mapstructure:"ai_insights"`
	PredictiveAnalytics bool `yaml:"predictive_analytics" mapstructure:"predictive_analytics"`
	SmartAlerts         bool `yaml:"smart_alerts" mapstructure:"smart_alerts"`
	NodeDetails         bool `yaml:"node_details" mapstructure:"node_details"`
}

// AIConfig holds AI-related configuration
type AIConfig struct {
	Enabled      bool          `yaml:"enabled" mapstructure:"enabled"`
	ClaudePath   string        `yaml:"claude_path" mapstructure:"claude_path"`
	MaxTurns     int           `yaml:"max_turns" mapstructure:"max_turns"`
	Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout"`
	SystemPrompt string        `yaml:"system_prompt" mapstructure:"system_prompt"`
	DatabasePath string        `yaml:"database_path" mapstructure:"database_path"`
	Features     AIFeatures    `yaml:"features" mapstructure:"features"`
}

// AIFeatures holds AI feature flags
type AIFeatures struct {
	ContextAwareAnalysis bool `yaml:"context_aware_analysis" mapstructure:"context_aware_analysis"`
	PredictiveAnalysis   bool `yaml:"predictive_analysis" mapstructure:"predictive_analysis"`
	AutoRemediation      bool `yaml:"auto_remediation" mapstructure:"auto_remediation"`
	SmartAlerts          bool `yaml:"smart_alerts" mapstructure:"smart_alerts"`
}

// LoadConfig loads configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	// Set defaults
	config := &Config{
		Kubernetes: KubernetesConfig{
			Kubeconfig: "~/.kube/config",
		},
		Monitoring: MonitoringConfig{
			Interval:      30 * time.Second,
			EnabledChecks: []string{"pod-health", "node-health", "service-health"},
			MaxHistory:    1000,
			Timeout:       30 * time.Second,
		},
		Alerts: AlertsConfig{
			Enabled: true,
			Channels: map[string]ChannelConfig{
				"log": {
					Type:    "log",
					Enabled: true,
				},
			},
		},
		ML: MLConfig{
			Enabled:         true,
			AnomalyEngine:   "statistical",
			Threshold:       2.0,
			LearningPeriod:  24 * time.Hour,
			PredictionHours: 24,
		},
		Server: ServerConfig{
			Port:         8080,
			Host:         "",
			EnableWeb:    true,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		},
		UI: UIConfig{
			RefreshInterval:      10 * time.Second,
			AIInsightsInterval:   30 * time.Second,
			MaxReconnectAttempts: 5,
			ReconnectDelay:       3 * time.Second,
			Theme:                "system",
			Features: UIFeatures{
				AIInsights:          true,
				PredictiveAnalytics: true,
				SmartAlerts:         true,
				NodeDetails:         true,
			},
		},
		AI: AIConfig{
			Enabled:      false, // Disabled by default, must be explicitly enabled
			ClaudePath:   "claude",
			MaxTurns:     3,
			Timeout:      120 * time.Second,
			DatabasePath: "~/.kubepulse/ai.db",
			Features: AIFeatures{
				ContextAwareAnalysis: true,
				PredictiveAnalysis:   true,
				AutoRemediation:      false, // Disabled by default for safety
				SmartAlerts:          true,
			},
		},
	}

	// Load from file if specified
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Load from environment
	if err := loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from YAML file
func loadFromFile(config *Config, path string) error {
	// Validate path to prevent directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid config path: path traversal detected")
	}

	data, err := os.ReadFile(path) // #nosec G304 -- path is validated above
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) error {
	viper.SetEnvPrefix("KUBEPULSE")
	viper.AutomaticEnv()

	// Bind environment variables
	_ = viper.BindEnv("kubernetes.kubeconfig", "KUBEPULSE_KUBECONFIG")
	_ = viper.BindEnv("monitoring.interval", "KUBEPULSE_INTERVAL")
	_ = viper.BindEnv("ml.enabled", "KUBEPULSE_ML_ENABLED")
	_ = viper.BindEnv("server.port", "KUBEPULSE_PORT")
	_ = viper.BindEnv("server.host", "KUBEPULSE_HOST")
	_ = viper.BindEnv("server.enable_web", "KUBEPULSE_WEB_ENABLED")
	_ = viper.BindEnv("ui.refresh_interval", "KUBEPULSE_UI_REFRESH")
	_ = viper.BindEnv("ui.theme", "KUBEPULSE_UI_THEME")

	// Override with environment values if set
	if viper.IsSet("kubernetes.kubeconfig") {
		config.Kubernetes.Kubeconfig = viper.GetString("kubernetes.kubeconfig")
	}
	if viper.IsSet("monitoring.interval") {
		config.Monitoring.Interval = viper.GetDuration("monitoring.interval")
	}
	if viper.IsSet("ml.enabled") {
		config.ML.Enabled = viper.GetBool("ml.enabled")
	}
	if viper.IsSet("server.port") {
		config.Server.Port = viper.GetInt("server.port")
	}
	if viper.IsSet("server.host") {
		config.Server.Host = viper.GetString("server.host")
	}
	if viper.IsSet("server.enable_web") {
		config.Server.EnableWeb = viper.GetBool("server.enable_web")
	}
	if viper.IsSet("ui.refresh_interval") {
		config.UI.RefreshInterval = viper.GetDuration("ui.refresh_interval")
	}
	if viper.IsSet("ui.theme") {
		config.UI.Theme = viper.GetString("ui.theme")
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate monitoring settings
	if config.Monitoring.Interval <= 0 {
		return fmt.Errorf("monitoring.interval must be positive")
	}
	if config.Monitoring.Timeout <= 0 {
		return fmt.Errorf("monitoring.timeout must be positive")
	}
	if config.Monitoring.MaxHistory <= 0 {
		config.Monitoring.MaxHistory = 1000
	}

	// Validate ML settings
	if config.ML.Threshold <= 0 {
		return fmt.Errorf("ml.threshold must be positive")
	}
	if config.ML.PredictionHours <= 0 {
		config.ML.PredictionHours = 24
	}

	return nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	config, _ := LoadConfig("")
	return config
}
