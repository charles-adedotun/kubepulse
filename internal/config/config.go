package config

import (
	"fmt"
	"os"
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
	data, err := os.ReadFile(path)
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

	return os.WriteFile(path, data, 0644)
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	config, _ := LoadConfig("")
	return config
}
