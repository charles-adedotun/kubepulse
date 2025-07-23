package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// RemediationEngine provides AI-powered auto-remediation
type RemediationEngine struct {
	client      *Client
	executor    CommandExecutor
	safetyCheck SafetyChecker
	history     *RemediationHistory
}

// CommandExecutor interface for executing remediation commands
type CommandExecutor interface {
	Execute(ctx context.Context, command string) (string, error)
	DryRun(ctx context.Context, command string) (string, error)
}

// SafetyChecker validates remediation safety
type SafetyChecker interface {
	IsSafe(action *RemediationAction) (bool, string)
	ValidateCommand(command string) error
}

// RemediationHistory tracks remediation actions
type RemediationHistory struct {
	actions    []RemediationRecord
	maxHistory int
}

// RemediationRecord represents a remediation attempt
type RemediationRecord struct {
	ID          string
	Timestamp   time.Time
	Problem     string
	Action      RemediationAction
	Result      string
	Success     bool
	RollbackCmd string
}

// RemediationAction represents an AI-suggested remediation
type RemediationAction struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Description      string    `json:"description"`
	Commands         []string  `json:"commands"`
	Risk             RiskLevel `json:"risk"`
	Confidence       float64   `json:"confidence"`
	Impact           string    `json:"impact"`
	Rollback         string    `json:"rollback_command"`
	RequiresApproval bool      `json:"requires_approval"`
}

// RiskLevel represents the risk of a remediation action
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// NewRemediationEngine creates a new remediation engine
func NewRemediationEngine(client *Client, executor CommandExecutor, checker SafetyChecker) *RemediationEngine {
	return &RemediationEngine{
		client:      client,
		executor:    executor,
		safetyCheck: checker,
		history: &RemediationHistory{
			actions:    []RemediationRecord{},
			maxHistory: 1000,
		},
	}
}

// GenerateRemediation creates AI-powered remediation plan
func (r *RemediationEngine) GenerateRemediation(ctx context.Context, issue CheckResult, context DiagnosticContext) ([]RemediationAction, error) {
	klog.V(2).Infof("Generating remediation for issue: %s", issue.Name)

	prompt := fmt.Sprintf(`Generate remediation actions for this Kubernetes issue:

Issue: %s
Status: %s
Message: %s
Error Logs: %v
Events: %v

Cluster Context:
- Resource Type: %s
- Resource Name: %s

For each remediation action provide:
1. Type (restart, scale, configure, patch, rollback)
2. Risk level (low, medium, high)
3. Exact kubectl commands
4. Rollback command
5. Expected impact
6. Confidence score

Prioritize:
- Minimal disruption
- Quick recovery
- Root cause fixes over symptoms
- Data preservation

Format as JSON array.`,
		issue.Name, issue.Status, issue.Message,
		context.ErrorLogs, context.Events,
		context.ResourceType, context.ResourceName)

	request := AnalysisRequest{
		Type:        AnalysisTypeHealing,
		Context:     prompt,
		HealthCheck: &issue,
		Data: map[string]interface{}{
			"diagnostic_context": context,
		},
		Timestamp: time.Now(),
	}

	response, err := r.client.Analyze(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("remediation generation failed: %w", err)
	}

	// Parse remediation actions
	actions := r.parseRemediationActions(response)

	// Validate safety of each action
	validatedActions := []RemediationAction{}
	for _, action := range actions {
		if safe, reason := r.safetyCheck.IsSafe(&action); safe {
			validatedActions = append(validatedActions, action)
		} else {
			klog.Warningf("Rejected unsafe remediation: %s, reason: %s", action.Description, reason)
		}
	}

	return validatedActions, nil
}

// ExecuteRemediation executes a remediation action
func (r *RemediationEngine) ExecuteRemediation(ctx context.Context, action RemediationAction, dryRun bool) (*RemediationRecord, error) {
	record := RemediationRecord{
		ID:        fmt.Sprintf("rem-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Action:    action,
	}

	// Safety validation
	if err := r.validateAction(action); err != nil {
		record.Success = false
		record.Result = fmt.Sprintf("Validation failed: %v", err)
		r.history.actions = append(r.history.actions, record)
		return &record, err
	}

	// Execute commands
	results := []string{}
	for _, cmd := range action.Commands {
		var result string
		var err error

		if dryRun {
			result, err = r.executor.DryRun(ctx, cmd)
		} else {
			result, err = r.executor.Execute(ctx, cmd)
		}

		if err != nil {
			record.Success = false
			record.Result = fmt.Sprintf("Command failed: %s, error: %v", cmd, err)

			// Attempt rollback if not in dry-run
			if !dryRun && record.RollbackCmd != "" {
				r.attemptRollback(ctx, record.RollbackCmd)
			}

			r.history.actions = append(r.history.actions, record)
			return &record, err
		}

		results = append(results, result)
	}

	record.Success = true
	record.Result = strings.Join(results, "\n")
	r.history.actions = append(r.history.actions, record)

	klog.Infof("Successfully executed remediation: %s", action.Description)

	return &record, nil
}

// GetSmartRecommendations provides context-aware remediation suggestions
func (r *RemediationEngine) GetSmartRecommendations(ctx context.Context, health *ClusterHealth) ([]RemediationAction, error) {
	// Analyze patterns in failures
	patterns := r.analyzeFailurePatterns(health)

	prompt := fmt.Sprintf(`Based on these Kubernetes cluster patterns, suggest proactive remediations:

Failure Patterns:
%+v

Recent Remediation History:
%+v

Suggest preventive actions that:
1. Address root causes not symptoms
2. Prevent future occurrences
3. Optimize resource usage
4. Improve reliability

Focus on automation-friendly actions.`,
		patterns, r.getRecentHistory(10))

	request := AnalysisRequest{
		Type:        AnalysisTypeHealing,
		Context:     prompt,
		ClusterInfo: health,
		Timestamp:   time.Now(),
	}

	response, err := r.client.Analyze(ctx, request)
	if err != nil {
		return nil, err
	}

	return r.parseRemediationActions(response), nil
}

// LearnFromOutcome updates AI knowledge based on remediation outcomes
func (r *RemediationEngine) LearnFromOutcome(ctx context.Context, recordID string, outcome string, successful bool) {
	record := r.findRecord(recordID)
	if record == nil {
		return
	}

	// Update AI context with outcome
	feedback := map[string]interface{}{
		"action":     record.Action,
		"outcome":    outcome,
		"successful": successful,
		"timestamp":  time.Now(),
	}

	prompt := fmt.Sprintf(`Learn from this remediation outcome:
Action: %+v
Success: %v
Outcome: %s

Update your knowledge to:
1. Improve future recommendations
2. Adjust confidence scores
3. Refine risk assessments`,
		record.Action, successful, outcome)

	request := AnalysisRequest{
		Type:    "learning",
		Context: prompt,
		Data: map[string]interface{}{
			"feedback": feedback,
		},
		Timestamp: time.Now(),
	}

	// Fire and forget - learning happens async
	go func() {
		if _, err := r.client.Analyze(context.Background(), request); err != nil {
			klog.Errorf("Failed to learn from outcome: %v", err)
		}
	}()
}

// Helper methods

func (r *RemediationEngine) parseRemediationActions(response *AnalysisResponse) []RemediationAction {
	actions := []RemediationAction{}

	for i, suggestedAction := range response.Actions {
		action := RemediationAction{
			ID:               fmt.Sprintf("action-%d-%d", time.Now().Unix(), i),
			Type:             string(suggestedAction.Type),
			Description:      suggestedAction.Description,
			Commands:         r.extractCommands(suggestedAction),
			Risk:             r.assessRisk(suggestedAction),
			Confidence:       response.Confidence,
			Impact:           r.assessImpact(suggestedAction),
			RequiresApproval: suggestedAction.RequiresApproval,
		}

		// Generate rollback command
		if rollback := r.generateRollback(action); rollback != "" {
			action.Rollback = rollback
		}

		actions = append(actions, action)
	}

	return actions
}

func (r *RemediationEngine) extractCommands(action SuggestedAction) []string {
	if action.Command != "" {
		return []string{action.Command}
	}
	if action.Script != "" {
		return strings.Split(action.Script, "\n")
	}
	return []string{}
}

func (r *RemediationEngine) assessRisk(action SuggestedAction) RiskLevel {
	switch action.Type {
	case ActionTypeRestart, ActionTypeScale:
		return RiskMedium
	case ActionTypeConfiguration:
		return RiskHigh
	case ActionTypeInvestigate:
		return RiskLow
	default:
		return RiskMedium
	}
}

func (r *RemediationEngine) assessImpact(action SuggestedAction) string {
	switch action.Type {
	case ActionTypeRestart:
		return "Temporary service disruption during restart"
	case ActionTypeScale:
		return "Resource usage will change"
	case ActionTypeConfiguration:
		return "Configuration changes may affect behavior"
	default:
		return "Minimal impact expected"
	}
}

func (r *RemediationEngine) generateRollback(action RemediationAction) string {
	// Generate inverse commands for common operations
	for _, cmd := range action.Commands {
		if strings.Contains(cmd, "scale") {
			// Extract current scale and generate rollback
			return strings.Replace(cmd, "replicas=", "replicas=<previous>", 1)
		}
		if strings.Contains(cmd, "set image") {
			return strings.Replace(cmd, "=", "=<previous>", 1)
		}
		if strings.Contains(cmd, "apply") {
			return strings.Replace(cmd, "apply", "delete", 1)
		}
	}
	return ""
}

func (r *RemediationEngine) validateAction(action RemediationAction) error {
	// Validate commands
	for _, cmd := range action.Commands {
		if err := r.safetyCheck.ValidateCommand(cmd); err != nil {
			return fmt.Errorf("invalid command: %w", err)
		}
	}

	// Check risk vs confidence
	if action.Risk == RiskHigh && action.Confidence < 0.8 {
		return fmt.Errorf("high risk action with low confidence (%.2f)", action.Confidence)
	}

	return nil
}

func (r *RemediationEngine) attemptRollback(ctx context.Context, rollbackCmd string) {
	klog.Warningf("Attempting rollback: %s", rollbackCmd)
	if _, err := r.executor.Execute(ctx, rollbackCmd); err != nil {
		klog.Errorf("Rollback failed: %v", err)
	}
}

func (r *RemediationEngine) analyzeFailurePatterns(health *ClusterHealth) map[string]int {
	patterns := make(map[string]int)

	for _, check := range health.Checks {
		if check.Status != HealthStatusHealthy {
			patterns[check.Name]++

			// Extract pattern from message
			if strings.Contains(check.Message, "OOMKilled") {
				patterns["memory_issues"]++
			}
			if strings.Contains(check.Message, "CrashLoopBackOff") {
				patterns["crash_loops"]++
			}
			if strings.Contains(check.Message, "ImagePullBackOff") {
				patterns["image_issues"]++
			}
		}
	}

	return patterns
}

func (r *RemediationEngine) getRecentHistory(limit int) []RemediationRecord {
	start := len(r.history.actions) - limit
	if start < 0 {
		start = 0
	}
	return r.history.actions[start:]
}

func (r *RemediationEngine) findRecord(id string) *RemediationRecord {
	for _, record := range r.history.actions {
		if record.ID == id {
			return &record
		}
	}
	return nil
}
