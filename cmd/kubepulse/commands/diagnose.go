package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kubepulse/kubepulse/internal/config"
	"github.com/kubepulse/kubepulse/pkg/ai"
	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/health"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	enableHealing    bool
	diagOutputFormat string
	confidenceMin    float64
)

// diagnoseCmd represents the diagnose command
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose [check-name]",
	Short: "AI-powered diagnostic analysis of health check failures",
	Long: `Diagnose uses Claude Code CLI to perform intelligent analysis of health check failures.
It provides detailed diagnostic insights, root cause analysis, and actionable recommendations.

Examples:
  kubepulse diagnose pod-health
  kubepulse diagnose --healing node-health
  kubepulse diagnose --format json pod-health`,
	RunE: runDiagnose,
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)

	diagnoseCmd.Flags().BoolVar(&enableHealing, "healing", false, "Include self-healing suggestions")
	diagnoseCmd.Flags().StringVar(&diagOutputFormat, "format", "text", "Output format (text, json)")
	diagnoseCmd.Flags().Float64Var(&confidenceMin, "confidence", 0.5, "Minimum AI confidence level")
	diagnoseCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to analyze (for pod checks)")
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("health check name required")
	}

	checkName := args[0]

	client := GetK8sClient()
	if client == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	// Create AI config and client
	aiConfig := &config.AIConfig{
		Enabled:    true,
		ClaudePath: "claude",
		MaxTurns:   3,
		Timeout:    120 * time.Second,
	}

	// Create AI client directly for diagnosis
	aiClientConfig := ai.Config{
		ClaudePath: aiConfig.ClaudePath,
		MaxTurns:   aiConfig.MaxTurns,
		Timeout:    aiConfig.Timeout,
	}
	aiClient := ai.NewClient(aiClientConfig)

	// Create monitoring engine to get health check results
	engineConfig := core.EngineConfig{
		KubeClient: client,
		EnableAI:   true,
		AIConfig:   aiConfig,
	}
	engine := core.NewEngine(engineConfig)

	// Run the specific health check
	fmt.Printf("🔍 Running health check: %s\n\n", checkName)

	checkResult, err := runSingleHealthCheck(engine, client, checkName, namespace)
	if err != nil {
		return fmt.Errorf("failed to run health check: %w", err)
	}

	// Display health check result
	displayHealthCheckResult(checkResult)

	// Only run AI analysis if there are issues
	if checkResult.Status == core.HealthStatusHealthy {
		fmt.Println("✅ Health check passed - no AI analysis needed")
		return nil
	}

	fmt.Printf("\n🤖 Running AI diagnostic analysis...\n")

	// Build diagnostic context
	diagnosticContext := buildDiagnosticContextFromCheck(checkResult)

	// Convert to AI types and run diagnostic analysis
	aiCheckResult := convertCoreToAICheckResult(checkResult)
	diagnosisResp, err := aiClient.AnalyzeDiagnostic(cmd.Context(), &aiCheckResult, diagnosticContext)
	if err != nil {
		return fmt.Errorf("AI diagnostic analysis failed: %w", err)
	}

	// Check confidence threshold
	if diagnosisResp.Confidence < confidenceMin {
		fmt.Printf("⚠️  AI confidence (%.2f) below minimum threshold (%.2f)\n",
			diagnosisResp.Confidence, confidenceMin)
	}

	// Display results based on output format
	switch diagOutputFormat {
	case "json":
		return displayJSONOutput(diagnosisResp, nil)
	default:
		displayTextDiagnosis(diagnosisResp)
	}

	// Run healing analysis if requested and confidence is sufficient
	if enableHealing && diagnosisResp.Confidence >= confidenceMin {
		fmt.Printf("\n🩺 Generating self-healing recommendations...\n")

		healingResp, err := aiClient.AnalyzeHealing(cmd.Context(), &aiCheckResult, diagnosticContext)
		if err != nil {
			klog.Errorf("AI healing analysis failed: %v", err)
			return nil // Don't fail the command for healing errors
		}

		switch diagOutputFormat {
		case "json":
			return displayJSONOutput(diagnosisResp, healingResp)
		default:
			displayTextHealing(healingResp)
		}
	}

	return nil
}

// runSingleHealthCheck executes a single health check
func runSingleHealthCheck(engine *core.Engine, client kubernetes.Interface, checkName, namespace string) (core.CheckResult, error) {

	var check core.HealthCheck

	switch checkName {
	case "pod-health":
		podCheck := health.NewPodHealthCheck()
		if namespace != "" {
			if err := podCheck.Configure(map[string]interface{}{
				"namespace": namespace,
			}); err != nil {
				return core.CheckResult{}, fmt.Errorf("failed to configure pod check: %w", err)
			}
		}
		check = podCheck

	case "node-health":
		check = health.NewNodeHealthCheck()

	case "service-health":
		serviceCheck := health.NewServiceHealthCheck()
		if namespace != "" {
			if err := serviceCheck.Configure(map[string]interface{}{
				"namespace": namespace,
			}); err != nil {
				return core.CheckResult{}, fmt.Errorf("failed to configure service check: %w", err)
			}
		}
		check = serviceCheck

	default:
		return core.CheckResult{}, fmt.Errorf("unknown health check: %s", checkName)
	}

	// Run the check
	result, err := check.Check(context.Background(), client)
	if err != nil {
		result.Status = core.HealthStatusUnknown
		result.Error = err
		result.Message = fmt.Sprintf("Check failed: %v", err)
	}

	result.Name = checkName
	return result, nil
}

// buildDiagnosticContextFromCheck creates diagnostic context from check result
func buildDiagnosticContextFromCheck(result core.CheckResult) ai.DiagnosticContext {
	// Convert metrics to AI format
	aiMetrics := make([]ai.Metric, len(result.Metrics))
	for i, metric := range result.Metrics {
		aiMetrics[i] = ai.Metric{
			Name:      metric.Name,
			Value:     metric.Value,
			Unit:      metric.Unit,
			Type:      ai.MetricType(metric.Type),
			Labels:    metric.Labels,
			Timestamp: metric.Timestamp,
		}
	}

	return ai.DiagnosticContext{
		ClusterName:  "default",
		ResourceType: extractResourceType(result.Name),
		ResourceName: extractResourceName(result.Name),
		ErrorLogs:    extractErrorLogs(result),
		Events:       extractEvents(result),
		Metrics:      aiMetrics,
	}
}

// displayHealthCheckResult shows the health check result
func displayHealthCheckResult(result core.CheckResult) {
	statusIcon := getStatusIcon(result.Status)
	fmt.Printf("%s %s: %s\n", statusIcon, result.Name, result.Status)

	if result.Message != "" {
		fmt.Printf("   Message: %s\n", result.Message)
	}

	if result.Error != nil {
		fmt.Printf("   Error: %s\n", result.Error.Error())
	}

	if len(result.Metrics) > 0 {
		fmt.Printf("   Metrics: %d collected\n", len(result.Metrics))
	}
}

// displayTextDiagnosis displays diagnostic analysis in text format
func displayTextDiagnosis(response *ai.AnalysisResponse) {
	fmt.Printf("\n📋 AI Diagnostic Analysis\n")
	fmt.Printf("─────────────────────────────\n")
	fmt.Printf("Summary: %s\n", response.Summary)
	fmt.Printf("Confidence: %.2f (%.0f%%)\n", response.Confidence, response.Confidence*100)
	fmt.Printf("Severity: %s\n", response.Severity)

	if response.Diagnosis != "" {
		fmt.Printf("\n📝 Detailed Diagnosis:\n%s\n", response.Diagnosis)
	}

	if len(response.Recommendations) > 0 {
		fmt.Printf("\n💡 Recommendations:\n")
		for i, rec := range response.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec.Title)
			fmt.Printf("     %s\n", rec.Description)
			if rec.Impact != "" {
				fmt.Printf("     Impact: %s\n", rec.Impact)
			}
			fmt.Println()
		}
	}
}

// displayTextHealing displays healing suggestions in text format
func displayTextHealing(response *ai.AnalysisResponse) {
	fmt.Printf("\n🩺 Self-Healing Suggestions\n")
	fmt.Printf("──────────────────────────────\n")
	fmt.Printf("Summary: %s\n", response.Summary)

	if len(response.Actions) > 0 {
		fmt.Printf("\n🔧 Suggested Actions:\n")
		for i, action := range response.Actions {
			fmt.Printf("  %d. %s (%s)\n", i+1, action.Title, action.Type)
			fmt.Printf("     %s\n", action.Description)

			if action.Command != "" {
				fmt.Printf("     Command: %s\n", action.Command)
			}

			if action.RequiresApproval {
				fmt.Printf("     ⚠️  Requires manual approval\n")
			} else if action.IsAutomatic {
				fmt.Printf("     ✅ Can be automated\n")
			}
			fmt.Println()
		}
	}

	if len(response.Recommendations) > 0 {
		fmt.Printf("\n📋 Additional Recommendations:\n")
		for i, rec := range response.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec.Title)
			fmt.Printf("     %s\n", rec.Description)
		}
	}
}

// displayJSONOutput displays results in JSON format
func displayJSONOutput(diagnosis *ai.AnalysisResponse, healing *ai.AnalysisResponse) error {
	output := map[string]interface{}{
		"diagnosis": diagnosis,
	}

	if healing != nil {
		output["healing"] = healing
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// getStatusIcon returns an icon for the health status
func getStatusIcon(status core.HealthStatus) string {
	switch status {
	case core.HealthStatusHealthy:
		return "✅"
	case core.HealthStatusDegraded:
		return "⚠️"
	case core.HealthStatusUnhealthy:
		return "❌"
	default:
		return "❓"
	}
}

// Helper functions (reuse from engine.go)
func extractResourceType(checkName string) string {
	if strings.Contains(strings.ToLower(checkName), "pod") {
		return "pod"
	}
	if strings.Contains(strings.ToLower(checkName), "node") {
		return "node"
	}
	if strings.Contains(strings.ToLower(checkName), "service") {
		return "service"
	}
	return "unknown"
}

func extractResourceName(checkName string) string {
	parts := strings.Split(checkName, "-")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return checkName
}

func extractErrorLogs(result core.CheckResult) []string {
	logs := []string{}
	if result.Error != nil {
		logs = append(logs, result.Error.Error())
	}
	if result.Message != "" {
		logs = append(logs, result.Message)
	}
	return logs
}

func extractEvents(result core.CheckResult) []string {
	events := []string{}
	if result.Details != nil {
		if eventData, exists := result.Details["events"]; exists {
			if eventSlice, ok := eventData.([]string); ok {
				events = append(events, eventSlice...)
			}
		}
	}
	return events
}

// convertCoreToAICheckResult converts core.CheckResult to ai.CheckResult
func convertCoreToAICheckResult(result core.CheckResult) ai.CheckResult {
	aiMetrics := make([]ai.Metric, len(result.Metrics))
	for i, metric := range result.Metrics {
		aiMetrics[i] = ai.Metric{
			Name:      metric.Name,
			Value:     metric.Value,
			Unit:      metric.Unit,
			Type:      ai.MetricType(metric.Type),
			Labels:    metric.Labels,
			Timestamp: metric.Timestamp,
		}
	}

	aiPredictions := make([]ai.HealthPrediction, len(result.Predictions))
	for i, pred := range result.Predictions {
		aiPredictions[i] = ai.HealthPrediction{
			Timestamp:   pred.Timestamp,
			Status:      ai.HealthStatus(pred.Status),
			Probability: pred.Probability,
			Reason:      pred.Reason,
		}
	}

	return ai.CheckResult{
		Name:        result.Name,
		Status:      ai.HealthStatus(result.Status),
		Message:     result.Message,
		Details:     result.Details,
		Error:       result.Error,
		Timestamp:   result.Timestamp,
		Duration:    result.Duration,
		Metrics:     aiMetrics,
		Predictions: aiPredictions,
	}
}
