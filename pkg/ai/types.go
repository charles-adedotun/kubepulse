package ai

import (
	"time"
)

// AnalysisRequest represents a request for AI analysis
type AnalysisRequest struct {
	Type        AnalysisType               `json:"type"`
	Context     string                     `json:"context"`
	Data        map[string]interface{}     `json:"data"`
	HealthCheck *CheckResult               `json:"health_check,omitempty"`
	ClusterInfo *ClusterHealth             `json:"cluster_info,omitempty"`
	Timestamp   time.Time                  `json:"timestamp"`
}

// AnalysisType defines the type of AI analysis to perform
type AnalysisType string

const (
	AnalysisTypeDiagnostic    AnalysisType = "diagnostic"
	AnalysisTypeHealing       AnalysisType = "healing"
	AnalysisTypePredictive    AnalysisType = "predictive"
	AnalysisTypeOptimization  AnalysisType = "optimization"
	AnalysisTypeSummary       AnalysisType = "summary"
	AnalysisTypeRootCause     AnalysisType = "root_cause"
)

// AnalysisResponse represents the AI's analysis response
type AnalysisResponse struct {
	ID            string                 `json:"id"`
	Type          AnalysisType           `json:"type"`
	Summary       string                 `json:"summary"`
	Diagnosis     string                 `json:"diagnosis"`
	Confidence    float64                `json:"confidence"`
	Severity      SeverityLevel          `json:"severity"`
	Recommendations []Recommendation     `json:"recommendations"`
	Actions       []SuggestedAction      `json:"actions"`
	Context       map[string]interface{} `json:"context"`
	Timestamp     time.Time              `json:"timestamp"`
	Duration      time.Duration          `json:"duration"`
}

// SeverityLevel represents the severity of an issue
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
	SeverityInfo     SeverityLevel = "info"
)

// Recommendation represents an AI-generated recommendation
type Recommendation struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Priority    int               `json:"priority"`
	Category    string            `json:"category"`
	Impact      string            `json:"impact"`
	Effort      string            `json:"effort"`
	References  []string          `json:"references,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SuggestedAction represents an actionable item
type SuggestedAction struct {
	ID          string            `json:"id"`
	Type        ActionType        `json:"type"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Command     string            `json:"command,omitempty"`
	Script      string            `json:"script,omitempty"`
	IsAutomatic bool              `json:"is_automatic"`
	RequiresApproval bool         `json:"requires_approval"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ActionType defines the type of suggested action
type ActionType string

const (
	ActionTypeKubectl     ActionType = "kubectl"
	ActionTypeScript      ActionType = "script"
	ActionTypeManual      ActionType = "manual"
	ActionTypeRestart     ActionType = "restart"
	ActionTypeScale       ActionType = "scale"
	ActionTypeConfiguration ActionType = "configuration"
	ActionTypeInvestigate ActionType = "investigate"
)

// InsightSummary represents aggregated AI insights
type InsightSummary struct {
	OverallHealth    string                 `json:"overall_health"`
	CriticalIssues   int                    `json:"critical_issues"`
	Recommendations  []Recommendation       `json:"top_recommendations"`
	TrendAnalysis    string                 `json:"trend_analysis"`
	PredictedIssues  []string               `json:"predicted_issues"`
	HealthScore      float64                `json:"health_score"`
	AIConfidence     float64                `json:"ai_confidence"`
	LastAnalyzed     time.Time              `json:"last_analyzed"`
	Context          map[string]interface{} `json:"context"`
}

// DiagnosticContext provides context for AI analysis
type DiagnosticContext struct {
	ClusterName     string                    `json:"cluster_name"`
	Namespace       string                    `json:"namespace,omitempty"`
	ResourceType    string                    `json:"resource_type,omitempty"`
	ResourceName    string                    `json:"resource_name,omitempty"`
	ErrorLogs       []string                  `json:"error_logs,omitempty"`
	Events          []string                  `json:"events,omitempty"`
	Metrics         []Metric                  `json:"metrics,omitempty"`
	RelatedChecks   []CheckResult             `json:"related_checks,omitempty"`
	HistoricalData  []CheckResult             `json:"historical_data,omitempty"`
	ClusterState    map[string]interface{}    `json:"cluster_state,omitempty"`
}

// Local type definitions to avoid import cycles

// HealthStatus represents the health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// MetricType represents the type of a metric
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a metric
type Metric struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit,omitempty"`
	Type      MetricType        `json:"type"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Error       error                  `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Metrics     []Metric               `json:"metrics,omitempty"`
	Predictions []Prediction           `json:"predictions,omitempty"`
}

// Prediction represents an ML prediction
type Prediction struct {
	Timestamp   time.Time    `json:"timestamp"`
	Status      HealthStatus `json:"status"`
	Probability float64      `json:"probability"`
	Reason      string       `json:"reason"`
}

// HealthScore represents health scoring information
type HealthScore struct {
	Raw        float64 `json:"raw"`
	Weighted   float64 `json:"weighted"`
	Trend      string  `json:"trend"`
	Confidence float64 `json:"confidence"`
	Forecast   string  `json:"forecast"`
}

// ClusterHealth represents overall cluster health
type ClusterHealth struct {
	ClusterName string        `json:"cluster_name"`
	Status      HealthStatus  `json:"status"`
	Score       HealthScore   `json:"score"`
	Checks      []CheckResult `json:"checks"`
	Timestamp   time.Time     `json:"timestamp"`
}