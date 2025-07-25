package ai

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"k8s.io/klog/v2"
)

// Database provides persistent storage for AI analysis data
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new AI database connection
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{db: db}
	if err := database.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	klog.Infof("AI Database initialized at %s", dbPath)
	return database, nil
}

// initSchema creates the database tables
func (d *Database) initSchema() error {
	schema := `
	-- Analysis History
	CREATE TABLE IF NOT EXISTS analysis_sessions (
		id TEXT PRIMARY KEY,
		cluster_name TEXT NOT NULL,
		analysis_type TEXT NOT NULL,
		kubectl_outputs TEXT NOT NULL, -- JSON of command results
		ai_response TEXT NOT NULL,
		confidence REAL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		duration INTEGER, -- milliseconds
		success BOOLEAN DEFAULT TRUE,
		error_message TEXT
	);

	-- Learned Patterns
	CREATE TABLE IF NOT EXISTS cluster_patterns (
		id TEXT PRIMARY KEY,
		cluster_name TEXT NOT NULL,
		pattern_type TEXT NOT NULL, -- 'anomaly', 'performance', 'failure', 'resource'
		pattern_name TEXT NOT NULL,
		indicators TEXT NOT NULL,   -- JSON array of pattern indicators
		description TEXT,
		first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		frequency INTEGER DEFAULT 1,
		confidence REAL DEFAULT 0.5
	);

	-- Solution Knowledge Base
	CREATE TABLE IF NOT EXISTS solutions (
		id TEXT PRIMARY KEY,
		problem_signature TEXT NOT NULL, -- Hash of problem characteristics
		problem_description TEXT NOT NULL,
		solution TEXT NOT NULL,
		kubectl_commands TEXT, -- JSON array of commands
		success_rate REAL DEFAULT 0.0,
		application_count INTEGER DEFAULT 0,
		last_applied DATETIME,
		cluster_context TEXT, -- JSON of cluster characteristics
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Performance Metrics
	CREATE TABLE IF NOT EXISTS ai_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		operation_type TEXT NOT NULL, -- 'analysis', 'diagnosis', 'prediction'
		cluster_name TEXT,
		response_time INTEGER, -- milliseconds
		confidence REAL,
		success BOOLEAN,
		error_message TEXT,
		tokens_used INTEGER,
		cost_estimate REAL
	);

	-- Kubectl Command Executions
	CREATE TABLE IF NOT EXISTS kubectl_executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cluster_name TEXT NOT NULL,
		command TEXT NOT NULL,
		output TEXT NOT NULL,
		success BOOLEAN NOT NULL,
		error_message TEXT,
		execution_time INTEGER, -- milliseconds
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		analysis_session_id TEXT,
		FOREIGN KEY(analysis_session_id) REFERENCES analysis_sessions(id)
	);

	-- Cluster Context Cache
	CREATE TABLE IF NOT EXISTS cluster_contexts (
		cluster_name TEXT PRIMARY KEY,
		last_analysis DATETIME,
		baseline_metrics TEXT, -- JSON of performance baselines
		known_issues TEXT,     -- JSON array of current issues
		ai_confidence REAL DEFAULT 0.5,
		health_score REAL,
		node_count INTEGER,
		namespace_count INTEGER,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes separately
	CREATE INDEX IF NOT EXISTS idx_analysis_sessions_cluster_timestamp ON analysis_sessions(cluster_name, timestamp);
	CREATE INDEX IF NOT EXISTS idx_analysis_sessions_type ON analysis_sessions(analysis_type);
	CREATE INDEX IF NOT EXISTS idx_cluster_patterns_cluster_type ON cluster_patterns(cluster_name, pattern_type);
	CREATE INDEX IF NOT EXISTS idx_cluster_patterns_last_seen ON cluster_patterns(last_seen);
	CREATE INDEX IF NOT EXISTS idx_solutions_signature ON solutions(problem_signature);
	CREATE INDEX IF NOT EXISTS idx_solutions_success_rate ON solutions(success_rate);
	CREATE INDEX IF NOT EXISTS idx_ai_metrics_timestamp_op ON ai_metrics(timestamp, operation_type);
	CREATE INDEX IF NOT EXISTS idx_ai_metrics_cluster ON ai_metrics(cluster_name);
	CREATE INDEX IF NOT EXISTS idx_kubectl_executions_cluster_timestamp ON kubectl_executions(cluster_name, timestamp);
	CREATE INDEX IF NOT EXISTS idx_kubectl_executions_command ON kubectl_executions(command);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	klog.V(2).Info("AI database schema initialized successfully")
	return nil
}

// StoreAnalysisSession stores an analysis session
func (d *Database) StoreAnalysisSession(session *AnalysisSession) error {
	query := `
		INSERT INTO analysis_sessions 
		(id, cluster_name, analysis_type, kubectl_outputs, ai_response, confidence, duration, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		session.ID,
		session.ClusterName,
		session.AnalysisType,
		session.KubectlOutputs,
		session.AIResponse,
		session.Confidence,
		session.Duration.Milliseconds(),
		session.Success,
		session.ErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to store analysis session: %w", err)
	}

	klog.V(3).Infof("Stored analysis session %s for cluster %s", session.ID, session.ClusterName)
	return nil
}

// GetAnalysisHistory retrieves recent analysis history for a cluster
func (d *Database) GetAnalysisHistory(clusterName string, since time.Time) ([]AnalysisSession, error) {
	query := `
		SELECT id, cluster_name, analysis_type, kubectl_outputs, ai_response, 
		       confidence, timestamp, duration, success, error_message
		FROM analysis_sessions 
		WHERE cluster_name = ? AND timestamp >= ?
		ORDER BY timestamp DESC
		LIMIT 50
	`

	rows, err := d.db.Query(query, clusterName, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var sessions []AnalysisSession
	for rows.Next() {
		var session AnalysisSession
		var timestamp time.Time
		var duration int64

		err := rows.Scan(
			&session.ID,
			&session.ClusterName,
			&session.AnalysisType,
			&session.KubectlOutputs,
			&session.AIResponse,
			&session.Confidence,
			&timestamp,
			&duration,
			&session.Success,
			&session.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analysis session: %w", err)
		}

		session.Timestamp = timestamp
		session.Duration = time.Duration(duration) * time.Millisecond
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// StorePattern stores a recognized pattern
func (d *Database) StorePattern(pattern *ClusterPattern) error {
	// Check if pattern already exists
	var existingID string
	err := d.db.QueryRow(
		"SELECT id FROM cluster_patterns WHERE cluster_name = ? AND pattern_type = ? AND pattern_name = ?",
		pattern.ClusterName, pattern.PatternType, pattern.PatternName,
	).Scan(&existingID)

	switch err {
	case sql.ErrNoRows:
		// Insert new pattern
		query := `
			INSERT INTO cluster_patterns 
			(id, cluster_name, pattern_type, pattern_name, indicators, description, confidence)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err = d.db.Exec(query,
			pattern.ID,
			pattern.ClusterName,
			pattern.PatternType,
			pattern.PatternName,
			pattern.Indicators,
			pattern.Description,
			pattern.Confidence,
		)
		if err != nil {
			return fmt.Errorf("failed to insert pattern: %w", err)
		}
	case nil:
		// Update existing pattern
		query := `
			UPDATE cluster_patterns 
			SET last_seen = CURRENT_TIMESTAMP, frequency = frequency + 1,
			    indicators = ?, description = ?, confidence = ?
			WHERE id = ?
		`
		_, err = d.db.Exec(query,
			pattern.Indicators,
			pattern.Description,
			pattern.Confidence,
			existingID,
		)
		if err != nil {
			return fmt.Errorf("failed to update pattern: %w", err)
		}
	default:
		return fmt.Errorf("failed to check existing pattern: %w", err)
	}

	return nil
}

// GetPatterns retrieves patterns for a cluster
func (d *Database) GetPatterns(clusterName string, patternType string) ([]ClusterPattern, error) {
	query := `
		SELECT id, cluster_name, pattern_type, pattern_name, indicators, 
		       description, first_seen, last_seen, frequency, confidence
		FROM cluster_patterns 
		WHERE cluster_name = ? AND (? = '' OR pattern_type = ?)
		ORDER BY last_seen DESC, frequency DESC
	`

	rows, err := d.db.Query(query, clusterName, patternType, patternType)
	if err != nil {
		return nil, fmt.Errorf("failed to query patterns: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var patterns []ClusterPattern
	for rows.Next() {
		var pattern ClusterPattern
		err := rows.Scan(
			&pattern.ID,
			&pattern.ClusterName,
			&pattern.PatternType,
			&pattern.PatternName,
			&pattern.Indicators,
			&pattern.Description,
			&pattern.FirstSeen,
			&pattern.LastSeen,
			&pattern.Frequency,
			&pattern.Confidence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pattern: %w", err)
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// RecordMetric records AI operation metrics
func (d *Database) RecordMetric(metric *AIMetric) error {
	query := `
		INSERT INTO ai_metrics 
		(operation_type, cluster_name, response_time, confidence, success, error_message, tokens_used, cost_estimate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		metric.OperationType,
		metric.ClusterName,
		metric.ResponseTime.Milliseconds(),
		metric.Confidence,
		metric.Success,
		metric.ErrorMessage,
		metric.TokensUsed,
		metric.CostEstimate,
	)

	if err != nil {
		return fmt.Errorf("failed to record AI metric: %w", err)
	}

	return nil
}

// GetMetricsSummary returns AI performance metrics summary
func (d *Database) GetMetricsSummary(since time.Time) (*MetricsSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_operations,
			AVG(response_time) as avg_response_time,
			AVG(confidence) as avg_confidence,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as success_rate,
			COALESCE(SUM(tokens_used), 0) as total_tokens,
			COALESCE(SUM(cost_estimate), 0) as total_cost
		FROM ai_metrics 
		WHERE timestamp >= ?
	`

	var summary MetricsSummary
	err := d.db.QueryRow(query, since).Scan(
		&summary.TotalOperations,
		&summary.AvgResponseTime,
		&summary.AvgConfidence,
		&summary.SuccessRate,
		&summary.TotalTokens,
		&summary.TotalCost,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics summary: %w", err)
	}

	return &summary, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// Health check for database
func (d *Database) Ping() error {
	return d.db.Ping()
}

// DB returns the underlying database connection for internal use
// This should only be used by other internal database-related methods
func (d *Database) DB() *sql.DB {
	return d.db
}