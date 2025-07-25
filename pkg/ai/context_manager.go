package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// ContextManager manages cluster context and learning data
type ContextManager struct {
	database    *Database
	cache       map[string]*ClusterContext
	cacheMu     sync.RWMutex
	cacheExpiry time.Duration
}

// NewContextManager creates a new context manager
func NewContextManager(database *Database) *ContextManager {
	return &ContextManager{
		database:    database,
		cache:       make(map[string]*ClusterContext),
		cacheExpiry: 5 * time.Minute, // Cache contexts for 5 minutes
	}
}

// GetContext retrieves or creates cluster context
func (c *ContextManager) GetContext(clusterName string) (*ClusterContext, error) {
	// Check cache first
	c.cacheMu.RLock()
	if ctx, exists := c.cache[clusterName]; exists {
		if time.Since(ctx.UpdatedAt) < c.cacheExpiry {
			c.cacheMu.RUnlock()
			return ctx, nil
		}
	}
	c.cacheMu.RUnlock()

	// Load from database
	ctx, err := c.loadFromDatabase(clusterName)
	if err != nil {
		// Create new context if not found
		if err.Error() == "context not found" {
			ctx = c.createNewContext(clusterName)
		} else {
			return nil, fmt.Errorf("failed to load context: %w", err)
		}
	}

	// Update cache
	c.cacheMu.Lock()
	c.cache[clusterName] = ctx
	c.cacheMu.Unlock()

	return ctx, nil
}

// UpdateContext updates cluster context with new information
func (c *ContextManager) UpdateContext(clusterName string, updates map[string]interface{}) error {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Apply updates
	if healthScore, ok := updates["health_score"].(float64); ok {
		ctx.HealthScore = healthScore
	}

	if nodeCount, ok := updates["node_count"].(int); ok {
		ctx.NodeCount = nodeCount
	}

	if namespaceCount, ok := updates["namespace_count"].(int); ok {
		ctx.NamespaceCount = namespaceCount
	}

	if confidence, ok := updates["ai_confidence"].(float64); ok {
		ctx.AIConfidence = confidence
	}

	if baselineMetrics, ok := updates["baseline_metrics"].(map[string]float64); ok {
		ctx.BaselineMetrics = baselineMetrics
	}

	if issues, ok := updates["known_issues"].([]Issue); ok {
		ctx.KnownIssues = issues
	}

	ctx.LastAnalysis = time.Now()
	ctx.UpdatedAt = time.Now()

	// Save to database
	if err := c.saveToDatabase(ctx); err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	// Update cache
	c.cacheMu.Lock()
	c.cache[clusterName] = ctx
	c.cacheMu.Unlock()

	klog.V(3).Infof("Updated context for cluster %s", clusterName)
	return nil
}

// AddKnownIssue adds a known issue to the cluster context
func (c *ContextManager) AddKnownIssue(clusterName string, issue Issue) error {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Check if issue already exists
	for i, existingIssue := range ctx.KnownIssues {
		if existingIssue.Type == issue.Type && existingIssue.Resource == issue.Resource {
			// Update existing issue
			ctx.KnownIssues[i].LastSeen = issue.LastSeen
			ctx.KnownIssues[i].Description = issue.Description
			ctx.KnownIssues[i].Status = issue.Status
			
			return c.saveToDatabase(ctx)
		}
	}

	// Add new issue
	ctx.KnownIssues = append(ctx.KnownIssues, issue)
	ctx.UpdatedAt = time.Now()

	return c.saveToDatabase(ctx)
}

// ResolveIssue marks an issue as resolved
func (c *ContextManager) ResolveIssue(clusterName, issueType, resource string) error {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	for i, issue := range ctx.KnownIssues {
		if issue.Type == issueType && issue.Resource == resource {
			ctx.KnownIssues[i].Status = "resolved"
			ctx.KnownIssues[i].LastSeen = time.Now()
			ctx.UpdatedAt = time.Now()

			return c.saveToDatabase(ctx)
		}
	}

	return fmt.Errorf("issue not found: type=%s, resource=%s", issueType, resource)
}

// GetActiveIssues returns active issues for a cluster
func (c *ContextManager) GetActiveIssues(clusterName string) ([]Issue, error) {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	var activeIssues []Issue
	for _, issue := range ctx.KnownIssues {
		if issue.Status == "active" {
			activeIssues = append(activeIssues, issue)
		}
	}

	return activeIssues, nil
}

// UpdateBaseline updates performance baselines for a cluster
func (c *ContextManager) UpdateBaseline(clusterName string, metrics map[string]float64) error {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Merge new metrics with existing baselines
	if ctx.BaselineMetrics == nil {
		ctx.BaselineMetrics = make(map[string]float64)
	}

	for key, value := range metrics {
		// Use exponential moving average for baseline updates
		if existing, exists := ctx.BaselineMetrics[key]; exists {
			ctx.BaselineMetrics[key] = existing*0.8 + value*0.2
		} else {
			ctx.BaselineMetrics[key] = value
		}
	}

	ctx.UpdatedAt = time.Now()
	return c.saveToDatabase(ctx)
}

// GetBaseline retrieves baseline metrics for a cluster
func (c *ContextManager) GetBaseline(clusterName string) (map[string]float64, error) {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	if ctx.BaselineMetrics == nil {
		return make(map[string]float64), nil
	}

	return ctx.BaselineMetrics, nil
}

// GetContextSummary returns a summary of cluster context
func (c *ContextManager) GetContextSummary(clusterName string) (map[string]interface{}, error) {
	ctx, err := c.GetContext(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get context: %w", err)
	}

	activeIssues := 0
	for _, issue := range ctx.KnownIssues {
		if issue.Status == "active" {
			activeIssues++
		}
	}

	summary := map[string]interface{}{
		"cluster_name":     ctx.ClusterName,
		"last_analysis":    ctx.LastAnalysis,
		"health_score":     ctx.HealthScore,
		"ai_confidence":    ctx.AIConfidence,
		"node_count":       ctx.NodeCount,
		"namespace_count":  ctx.NamespaceCount,
		"active_issues":    activeIssues,
		"total_issues":     len(ctx.KnownIssues),
		"has_baseline":     len(ctx.BaselineMetrics) > 0,
		"baseline_metrics": len(ctx.BaselineMetrics),
		"updated_at":       ctx.UpdatedAt,
	}

	return summary, nil
}

// CleanupOldData removes old context data
func (c *ContextManager) CleanupOldData(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)

	// For SQLite, we need to handle JSON differently
	// First, get all cluster contexts
	rows, err := c.database.DB().QueryContext(ctx, "SELECT cluster_name, known_issues FROM cluster_contexts")
	if err != nil {
		return fmt.Errorf("failed to query cluster contexts: %w", err)
	}
	defer rows.Close()

	// Process each cluster context
	for rows.Next() {
		var clusterName string
		var knownIssuesJSON string
		
		if err := rows.Scan(&clusterName, &knownIssuesJSON); err != nil {
			klog.Errorf("Failed to scan cluster context: %v", err)
			continue
		}

		// Parse the JSON
		var issues []Issue
		if knownIssuesJSON != "" {
			if err := json.Unmarshal([]byte(knownIssuesJSON), &issues); err != nil {
				klog.Errorf("Failed to parse known issues for %s: %v", clusterName, err)
				continue
			}
		}

		// Filter out old resolved issues
		var activeIssues []Issue
		for _, issue := range issues {
			if issue.Status != "resolved" || issue.LastSeen.After(cutoff) {
				activeIssues = append(activeIssues, issue)
			}
		}

		// Update if there were changes
		if len(activeIssues) < len(issues) {
			updatedJSON, err := json.Marshal(activeIssues)
			if err != nil {
				klog.Errorf("Failed to marshal updated issues for %s: %v", clusterName, err)
				continue
			}

			_, err = c.database.DB().ExecContext(ctx, 
				"UPDATE cluster_contexts SET known_issues = ? WHERE cluster_name = ?",
				string(updatedJSON), clusterName)
			if err != nil {
				klog.Errorf("Failed to update cluster context for %s: %v", clusterName, err)
			}
		}
	}

	// Clear cache to force reload
	c.cacheMu.Lock()
	c.cache = make(map[string]*ClusterContext)
	c.cacheMu.Unlock()

	klog.V(2).Infof("Cleaned up context data older than %v", maxAge)
	return nil
}

// ListClusters returns all clusters with contexts
func (c *ContextManager) ListClusters() ([]string, error) {
	query := "SELECT DISTINCT cluster_name FROM cluster_contexts ORDER BY cluster_name"
	
	rows, err := c.database.DB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer rows.Close()

	var clusters []string
	for rows.Next() {
		var clusterName string
		if err := rows.Scan(&clusterName); err != nil {
			return nil, fmt.Errorf("failed to scan cluster name: %w", err)
		}
		clusters = append(clusters, clusterName)
	}

	return clusters, nil
}

// loadFromDatabase loads context from database
func (c *ContextManager) loadFromDatabase(clusterName string) (*ClusterContext, error) {
	query := `
		SELECT cluster_name, last_analysis, baseline_metrics, known_issues, 
		       ai_confidence, health_score, node_count, namespace_count, updated_at
		FROM cluster_contexts 
		WHERE cluster_name = ?
	`

	var ctx ClusterContext
	var baselineMetricsJSON, knownIssuesJSON string

	err := c.database.DB().QueryRow(query, clusterName).Scan(
		&ctx.ClusterName,
		&ctx.LastAnalysis,
		&baselineMetricsJSON,
		&knownIssuesJSON,
		&ctx.AIConfidence,
		&ctx.HealthScore,
		&ctx.NodeCount,
		&ctx.NamespaceCount,
		&ctx.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("context not found")
	}

	// Parse JSON fields
	if baselineMetricsJSON != "" {
		if err := json.Unmarshal([]byte(baselineMetricsJSON), &ctx.BaselineMetrics); err != nil {
			klog.Errorf("Failed to parse baseline metrics for %s: %v", clusterName, err)
			ctx.BaselineMetrics = make(map[string]float64)
		}
	} else {
		ctx.BaselineMetrics = make(map[string]float64)
	}

	if knownIssuesJSON != "" {
		if err := json.Unmarshal([]byte(knownIssuesJSON), &ctx.KnownIssues); err != nil {
			klog.Errorf("Failed to parse known issues for %s: %v", clusterName, err)
			ctx.KnownIssues = []Issue{}
		}
	} else {
		ctx.KnownIssues = []Issue{}
	}

	return &ctx, nil
}

// saveToDatabase saves context to database
func (c *ContextManager) saveToDatabase(ctx *ClusterContext) error {
	baselineMetricsJSON, _ := json.Marshal(ctx.BaselineMetrics)
	knownIssuesJSON, _ := json.Marshal(ctx.KnownIssues)

	query := `
		INSERT OR REPLACE INTO cluster_contexts 
		(cluster_name, last_analysis, baseline_metrics, known_issues, 
		 ai_confidence, health_score, node_count, namespace_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := c.database.DB().Exec(query,
		ctx.ClusterName,
		ctx.LastAnalysis,
		string(baselineMetricsJSON),
		string(knownIssuesJSON),
		ctx.AIConfidence,
		ctx.HealthScore,
		ctx.NodeCount,
		ctx.NamespaceCount,
		ctx.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save context: %w", err)
	}

	return nil
}

// createNewContext creates a new context for a cluster
func (c *ContextManager) createNewContext(clusterName string) *ClusterContext {
	now := time.Now()
	return &ClusterContext{
		ClusterName:     clusterName,
		LastAnalysis:    time.Time{}, // Zero time indicates never analyzed
		BaselineMetrics: make(map[string]float64),
		KnownIssues:     []Issue{},
		AIConfidence:    0.5, // Start with neutral confidence
		HealthScore:     0.0,
		NodeCount:       0,
		NamespaceCount:  0,
		UpdatedAt:       now,
	}
}