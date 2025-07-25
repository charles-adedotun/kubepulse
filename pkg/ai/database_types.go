package ai

import (
	"time"
)

// AnalysisSession represents a stored AI analysis session
type AnalysisSession struct {
	ID             string        `json:"id"`
	ClusterName    string        `json:"cluster_name"`
	AnalysisType   string        `json:"analysis_type"`
	KubectlOutputs string        `json:"kubectl_outputs"` // JSON encoded
	AIResponse     string        `json:"ai_response"`
	Confidence     float64       `json:"confidence"`
	Timestamp      time.Time     `json:"timestamp"`
	Duration       time.Duration `json:"duration"`
	Success        bool          `json:"success"`
	ErrorMessage   string        `json:"error_message,omitempty"`
}

// ClusterPattern represents a recognized pattern in cluster behavior
type ClusterPattern struct {
	ID          string    `json:"id"`
	ClusterName string    `json:"cluster_name"`
	PatternType string    `json:"pattern_type"` // 'anomaly', 'performance', 'failure', 'resource'
	PatternName string    `json:"pattern_name"`
	Indicators  string    `json:"indicators"` // JSON encoded array
	Description string    `json:"description"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Frequency   int       `json:"frequency"`
	Confidence  float64   `json:"confidence"`
}

// Solution represents a stored solution for a known problem
type Solution struct {
	ID                 string    `json:"id"`
	ProblemSignature   string    `json:"problem_signature"`
	ProblemDescription string    `json:"problem_description"`
	Solution           string    `json:"solution"`
	KubectlCommands    string    `json:"kubectl_commands"` // JSON encoded array
	SuccessRate        float64   `json:"success_rate"`
	ApplicationCount   int       `json:"application_count"`
	LastApplied        time.Time `json:"last_applied"`
	ClusterContext     string    `json:"cluster_context"` // JSON encoded
	CreatedAt          time.Time `json:"created_at"`
}

// AIMetric represents AI operation performance metrics
type AIMetric struct {
	ID            int           `json:"id"`
	Timestamp     time.Time     `json:"timestamp"`
	OperationType string        `json:"operation_type"` // 'analysis', 'diagnosis', 'prediction'
	ClusterName   string        `json:"cluster_name"`
	ResponseTime  time.Duration `json:"response_time"`
	Confidence    float64       `json:"confidence"`
	Success       bool          `json:"success"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	TokensUsed    int           `json:"tokens_used"`
	CostEstimate  float64       `json:"cost_estimate"`
}

// KubectlExecution represents a stored kubectl command execution
type KubectlExecution struct {
	ID                int           `json:"id"`
	ClusterName       string        `json:"cluster_name"`
	Command           string        `json:"command"`
	Output            string        `json:"output"`
	Success           bool          `json:"success"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	ExecutionTime     time.Duration `json:"execution_time"`
	Timestamp         time.Time     `json:"timestamp"`
	AnalysisSessionID string        `json:"analysis_session_id,omitempty"`
}

// ClusterContext represents cached cluster context information
type ClusterContext struct {
	ClusterName     string             `json:"cluster_name"`
	LastAnalysis    time.Time          `json:"last_analysis"`
	BaselineMetrics map[string]float64 `json:"baseline_metrics"`
	KnownIssues     []Issue            `json:"known_issues"`
	AIConfidence    float64            `json:"ai_confidence"`
	HealthScore     float64            `json:"health_score"`
	NodeCount       int                `json:"node_count"`
	NamespaceCount  int                `json:"namespace_count"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// Issue represents a known cluster issue
type Issue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Resource    string    `json:"resource,omitempty"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"` // 'active', 'resolved', 'ignored'
}

// MetricsSummary provides aggregated AI performance metrics
type MetricsSummary struct {
	TotalOperations int     `json:"total_operations"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	AvgConfidence   float64 `json:"avg_confidence"`
	SuccessRate     float64 `json:"success_rate_percent"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost_usd"`
}

// AnalysisRequest represents a request for AI analysis
type AnalysisRequestV2 struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // 'comprehensive', 'focused', 'diagnostic', 'predictive'
	ClusterName string                 `json:"cluster_name"`
	Context     string                 `json:"context"`
	KubectlData map[string]interface{} `json:"kubectl_data"`
	HistoryData []AnalysisSession      `json:"history_data,omitempty"`
	PatternData []ClusterPattern       `json:"pattern_data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Timeout     time.Duration          `json:"timeout"`
	Priority    string                 `json:"priority"` // 'low', 'normal', 'high', 'urgent'
}

// AnalysisResult represents the result of AI analysis
type AnalysisResult struct {
	ID                string                 `json:"id"`
	RequestID         string                 `json:"request_id"`
	ClusterName       string                 `json:"cluster_name"`
	AnalysisType      string                 `json:"analysis_type"`
	Summary           string                 `json:"summary"`
	Findings          []Finding              `json:"findings"`
	Recommendations   []Recommendation       `json:"recommendations"`
	KubectlCommands   []string               `json:"kubectl_commands"`
	Confidence        float64                `json:"confidence"`
	Severity          SeverityLevel          `json:"severity"`
	ActionRequired    bool                   `json:"action_required"`
	FollowUpQuestions []string               `json:"follow_up_questions,omitempty"`
	Context           map[string]interface{} `json:"context,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
	Duration          time.Duration          `json:"duration"`
	TokensUsed        int                    `json:"tokens_used"`
	CostEstimate      float64                `json:"cost_estimate"`
}

// Finding represents a specific finding from AI analysis
type Finding struct {
	Type        string        `json:"type"`     // 'issue', 'opportunity', 'observation'
	Category    string        `json:"category"` // 'performance', 'security', 'reliability'
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Severity    SeverityLevel `json:"severity"`
	Evidence    []string      `json:"evidence"` // kubectl commands/outputs that support this finding
	Resource    string        `json:"resource,omitempty"`
	Namespace   string        `json:"namespace,omitempty"`
	Confidence  float64       `json:"confidence"`
}

// KubectlToolResult represents the output from a kubectl tool execution
type KubectlToolResult struct {
	ToolName      string                 `json:"tool_name"`
	Commands      []string               `json:"commands"`
	Outputs       map[string]string      `json:"outputs"` // command -> output
	Errors        map[string]string      `json:"errors"`  // command -> error
	Summary       string                 `json:"summary"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Success       bool                   `json:"success"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Recommendation represents an AI-generated recommendation
type Recommendation struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`   // 'performance', 'security', 'reliability'
	Priority    string   `json:"priority"`   // 'low', 'medium', 'high', 'critical'
	Commands    []string `json:"commands"`   // kubectl commands to execute
	Impact      string   `json:"impact"`     // expected impact description
	References  []string `json:"references"` // documentation references
	Confidence  float64  `json:"confidence"`
	Automated   bool     `json:"automated"` // can be automated
}

// SeverityLevel represents the severity of findings
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// SmartAlert represents an intelligent alert generated by AI analysis
type SmartAlert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Evidence    []string  `json:"evidence"`
	Timestamp   time.Time `json:"timestamp"`
}

// Prediction represents a predictive insight about the cluster
type Prediction struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Likelihood  float64  `json:"likelihood"`
	Timeline    string   `json:"timeline"`
	Impact      string   `json:"impact"`
	Evidence    []string `json:"evidence"`
}
