package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PromptBuilder creates contextual AI prompts based on kubectl data and runbook knowledge
type PromptBuilder struct {
	systemPrompts map[string]string
	runbookKnowledge *RunbookKnowledge
}

// RunbookKnowledge contains kubectl runbook expertise
type RunbookKnowledge struct {
	categories         map[string][]string // category -> commands
	troubleshootingMap map[string][]string // issue -> diagnostic commands  
	metricsCommands    []string
	debuggingCommands  []string
}

// PromptContext contains all context needed for prompt building
type PromptContext struct {
	AnalysisType   string
	ClusterName    string
	KubectlResults map[string]*KubectlToolResult
	ClusterContext *ClusterContext
	HistoryData    []AnalysisSession
	Patterns       []ClusterPattern
	UserContext    string
}

// NewPromptBuilder creates a new prompt builder with runbook knowledge
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		systemPrompts: map[string]string{
			"comprehensive": getComprehensiveSystemPrompt(),
			"diagnostic":    getDiagnosticSystemPrompt(),
			"optimization":  getOptimizationSystemPrompt(),
			"query":         getQuerySystemPrompt(),
			"infrastructure": getInfrastructureSystemPrompt(),
		},
		runbookKnowledge: initializeRunbookKnowledge(),
	}
}

// initializeRunbookKnowledge sets up the kubectl runbook knowledge base
func initializeRunbookKnowledge() *RunbookKnowledge {
	return &RunbookKnowledge{
		categories: map[string][]string{
			"control-plane": {
				"kubectl get --raw='/livez?verbose'",
				"kubectl get --raw='/readyz?verbose'", 
				"kubectl get componentstatuses",
				"kubectl get --raw='/metrics' | head -200",
				"kubectl cluster-info",
			},
			"nodes": {
				"kubectl get nodes -o wide",
				"kubectl top nodes", 
				"kubectl describe nodes",
				"kubectl get nodes -o custom-columns=NAME:.metadata.name,STATUS:.status.conditions[?(@.status==\"True\")].reason",
			},
			"workloads": {
				"kubectl get pods -A -o wide",
				"kubectl get pods -A --field-selector status.phase!=Running",
				"kubectl get deployments -A",
				"kubectl top pods -A --sort-by=cpu",
				"kubectl rollout status deployment/<name> -n <ns>",
			},
			"events": {
				"kubectl get events -A --sort-by=.lastTimestamp",
				"kubectl get events -A --sort-by=.lastTimestamp | tail -40",
				"kubectl get events -A --watch",
			},
			"networking": {
				"kubectl get svc -A",
				"kubectl get ep -A", 
				"kubectl get ingress -A",
			},
			"storage": {
				"kubectl get pvc -A",
				"kubectl get pv",
			},
		},
		troubleshootingMap: map[string][]string{
			"pod-crash": {
				"kubectl describe pod <pod> -n <ns>",
				"kubectl logs <pod> -n <ns> --tail=100",
				"kubectl logs <pod> -n <ns> --previous",
				"kubectl get events -n <ns> --field-selector involvedObject.name=<pod>",
			},
			"node-issues": {
				"kubectl describe node <node>",
				"kubectl get events -A --field-selector involvedObject.name=<node>",
				"kubectl top node <node>",
			},
			"resource-pressure": {
				"kubectl top nodes",
				"kubectl top pods -A --sort-by=cpu",
				"kubectl top pods -A --sort-by=memory",
				"kubectl describe nodes | grep -A 5 -B 5 Pressure",
			},
			"networking": {
				"kubectl get svc -A",
				"kubectl get ep -A",
				"kubectl describe svc <service> -n <ns>",
				"kubectl get networkpolicies -A",
			},
		},
		metricsCommands: []string{
			"kubectl get --raw \"/apis/metrics.k8s.io/v1beta1/nodes\" | jq .",
			"kubectl get --raw \"/apis/metrics.k8s.io/v1beta1/pods\" | jq .",
			"kubectl top nodes",
			"kubectl top pods -A",
		},
		debuggingCommands: []string{
			"kubectl describe pod <pod> -n <ns>",
			"kubectl logs <pod> -n <ns> --tail=100",
			"kubectl logs -f <pod> -n <ns>",
			"kubectl exec -it <pod> -n <ns> -- printenv",
			"kubectl exec -it <pod> -n <ns> -- ls /path",
		},
	}
}

// BuildAnalysisPrompt builds a complete AI prompt for cluster analysis
func (p *PromptBuilder) BuildAnalysisPrompt(ctx *PromptContext) (string, error) {
	var promptBuilder strings.Builder

	// Add system context header
	promptBuilder.WriteString(fmt.Sprintf("KUBERNETES CLUSTER ANALYSIS REQUEST\n"))
	promptBuilder.WriteString(fmt.Sprintf("Analysis Type: %s\n", ctx.AnalysisType))
	promptBuilder.WriteString(fmt.Sprintf("Cluster: %s\n", ctx.ClusterName))
	promptBuilder.WriteString(fmt.Sprintf("Timestamp: %s\n\n", time.Now().Format(time.RFC3339)))

	// Add user context if provided
	if ctx.UserContext != "" {
		promptBuilder.WriteString(fmt.Sprintf("USER REQUEST: %s\n\n", ctx.UserContext))
	}

	// Add kubectl results
	if err := p.addKubectlResults(&promptBuilder, ctx.KubectlResults); err != nil {
		return "", fmt.Errorf("failed to add kubectl results: %w", err)
	}

	// Add cluster context
	if ctx.ClusterContext != nil {
		p.addClusterContext(&promptBuilder, ctx.ClusterContext)
	}

	// Add historical context
	if len(ctx.HistoryData) > 0 {
		p.addHistoricalContext(&promptBuilder, ctx.HistoryData)
	}

	// Add pattern context
	if len(ctx.Patterns) > 0 {
		p.addPatternContext(&promptBuilder, ctx.Patterns)
	}

	// Add analysis instructions
	p.addAnalysisInstructions(&promptBuilder, ctx.AnalysisType)

	return promptBuilder.String(), nil
}

// addKubectlResults adds kubectl command outputs to the prompt
func (p *PromptBuilder) addKubectlResults(builder *strings.Builder, results map[string]*KubectlToolResult) error {
	builder.WriteString("=== KUBECTL COMMAND OUTPUTS ===\n\n")

	if len(results) == 0 {
		builder.WriteString("No kubectl data available.\n\n")
		return nil
	}

	for toolName, result := range results {
		builder.WriteString(fmt.Sprintf("## Tool: %s\n", toolName))
		builder.WriteString(fmt.Sprintf("Execution Time: %v\n", result.ExecutionTime))
		builder.WriteString(fmt.Sprintf("Success: %t\n", result.Success))

		if result.Summary != "" {
			builder.WriteString(fmt.Sprintf("Summary: %s\n", result.Summary))
		}

		// Add command outputs
		for cmd, output := range result.Outputs {
			builder.WriteString(fmt.Sprintf("\n### Command: %s\n", cmd))
			if len(output) > 5000 {
				// Truncate very long outputs but preserve structure
				lines := strings.Split(output, "\n")
				if len(lines) > 50 {
					truncated := append(lines[:25], "... [truncated] ...")
				truncated = append(truncated, lines[len(lines)-25:]...)
					output = strings.Join(truncated, "\n")
				}
			}
			builder.WriteString(fmt.Sprintf("```\n%s\n```\n", output))
		}

		// Add errors if any
		if len(result.Errors) > 0 {
			builder.WriteString("\n### Errors:\n")
			for cmd, errMsg := range result.Errors {
				builder.WriteString(fmt.Sprintf("- %s: %s\n", cmd, errMsg))
			}
		}

		// Add metadata if available
		if result.Metadata != nil && len(result.Metadata) > 0 {
			if extractedMetrics, ok := result.Metadata["extracted_metrics"]; ok {
				metricJSON, _ := json.MarshalIndent(extractedMetrics, "", "  ")
				builder.WriteString(fmt.Sprintf("\n### Extracted Metrics:\n```json\n%s\n```\n", string(metricJSON)))
			}
		}

		builder.WriteString("\n---\n\n")
	}

	return nil
}

// addClusterContext adds cluster context information to the prompt
func (p *PromptBuilder) addClusterContext(builder *strings.Builder, ctx *ClusterContext) {
	builder.WriteString("=== CLUSTER CONTEXT ===\n\n")
	
	builder.WriteString(fmt.Sprintf("Last Analysis: %s\n", ctx.LastAnalysis.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("Health Score: %.2f\n", ctx.HealthScore))
	builder.WriteString(fmt.Sprintf("AI Confidence: %.2f\n", ctx.AIConfidence))
	builder.WriteString(fmt.Sprintf("Node Count: %d\n", ctx.NodeCount))
	builder.WriteString(fmt.Sprintf("Namespace Count: %d\n", ctx.NamespaceCount))

	// Add baseline metrics
	if len(ctx.BaselineMetrics) > 0 {
		builder.WriteString("\n## Baseline Performance Metrics:\n")
		for metric, value := range ctx.BaselineMetrics {
			builder.WriteString(fmt.Sprintf("- %s: %.2f\n", metric, value))
		}
	}

	// Add known issues
	if len(ctx.KnownIssues) > 0 {
		builder.WriteString("\n## Known Issues:\n")
		for _, issue := range ctx.KnownIssues {
			if issue.Status == "active" {
				builder.WriteString(fmt.Sprintf("- [%s] %s: %s (Resource: %s, Since: %s)\n",
					issue.Severity, issue.Type, issue.Description, issue.Resource, 
					issue.FirstSeen.Format("2006-01-02")))
			}
		}
	}

	builder.WriteString("\n")
}

// addHistoricalContext adds historical analysis data to the prompt
func (p *PromptBuilder) addHistoricalContext(builder *strings.Builder, history []AnalysisSession) {
	builder.WriteString("=== ANALYSIS HISTORY ===\n\n")
	
	builder.WriteString(fmt.Sprintf("Recent analysis sessions (%d):\n", len(history)))
	
	for i, session := range history {
		if i >= 3 { // Limit to most recent 3 for brevity
			break
		}
		
		builder.WriteString(fmt.Sprintf("\n## Session %d (%s):\n", i+1, session.Timestamp.Format("2006-01-02 15:04")))
		builder.WriteString(fmt.Sprintf("Type: %s\n", session.AnalysisType))
		builder.WriteString(fmt.Sprintf("Confidence: %.2f\n", session.Confidence))
		builder.WriteString(fmt.Sprintf("Duration: %v\n", session.Duration))
		
		if session.AIResponse != "" && len(session.AIResponse) < 500 {
			builder.WriteString(fmt.Sprintf("Summary: %s\n", session.AIResponse))
		}
		
		if !session.Success && session.ErrorMessage != "" {
			builder.WriteString(fmt.Sprintf("Error: %s\n", session.ErrorMessage))
		}
	}
	
	builder.WriteString("\n")
}

// addPatternContext adds recognized patterns to the prompt
func (p *PromptBuilder) addPatternContext(builder *strings.Builder, patterns []ClusterPattern) {
	builder.WriteString("=== RECOGNIZED PATTERNS ===\n\n")
	
	recentPatterns := patterns
	if len(recentPatterns) > 5 {
		recentPatterns = patterns[:5] // Limit to 5 most recent
	}
	
	for _, pattern := range recentPatterns {
		builder.WriteString(fmt.Sprintf("## Pattern: %s\n", pattern.PatternName))
		builder.WriteString(fmt.Sprintf("Type: %s\n", pattern.PatternType))
		builder.WriteString(fmt.Sprintf("Description: %s\n", pattern.Description))
		builder.WriteString(fmt.Sprintf("Frequency: %d occurrences\n", pattern.Frequency))
		builder.WriteString(fmt.Sprintf("Confidence: %.2f\n", pattern.Confidence))
		builder.WriteString(fmt.Sprintf("Last Seen: %s\n", pattern.LastSeen.Format("2006-01-02 15:04")))
		
		if pattern.Indicators != "" {
			builder.WriteString(fmt.Sprintf("Indicators: %s\n", pattern.Indicators))
		}
		
		builder.WriteString("\n")
	}
}

// addAnalysisInstructions adds specific instructions based on analysis type
func (p *PromptBuilder) addAnalysisInstructions(builder *strings.Builder, analysisType string) {
	builder.WriteString("=== ANALYSIS INSTRUCTIONS ===\n\n")
	
	// Get type-specific system prompt
	if systemPrompt, exists := p.systemPrompts[analysisType]; exists {
		builder.WriteString(systemPrompt)
	} else {
		// Default instructions
		builder.WriteString(getDefaultAnalysisInstructions())
	}
	
	builder.WriteString("\n")
	
	// Add common requirements
	builder.WriteString(`
RESPONSE REQUIREMENTS:
1. Provide a clear executive summary (2-3 sentences)
2. List specific findings with evidence from kubectl outputs
3. Include actionable recommendations with priority levels
4. Suggest specific kubectl commands for further investigation
5. Assess confidence level (0.0-1.0) in your analysis
6. Identify any urgent issues requiring immediate attention

OUTPUT FORMAT:
Please structure your response with clear sections and use the kubectl command outputs as evidence for your analysis. Focus on actionable insights that can help improve cluster health and performance.
`)
}

// System prompt templates for different analysis types
func getComprehensiveSystemPrompt() string {
	return `You are an expert Kubernetes cluster analyst equipped with comprehensive kubectl runbook knowledge. Review all kubectl outputs to provide:

## KUBECTL RUNBOOK EXPERTISE
You have access to the complete "Kubernetes Read-Only kubectl Run Book" with expertise in:
- Control-Plane Vitality: API server health, component status, metrics analysis
- Node Condition & Capacity: Node health, resource usage, capacity planning
- Workload Health: Pod status, deployment rollouts, restart patterns
- Events & Recent Changes: Event correlation, timeline analysis
- Networking: Service discovery, endpoint health, ingress analysis
- Storage: PVC/PV status, volume health
- On-the-Spot Debugging: Log analysis, environment inspection
- Raw Metrics API: Direct metrics interpretation

## ANALYSIS FRAMEWORK
1. EXECUTIVE SUMMARY: Overall cluster health status and key concerns
2. INFRASTRUCTURE ANALYSIS: 
   - Control plane health (API server liveness/readiness, component status)
   - Node health and capacity (resource pressure, conditions)
3. WORKLOAD ANALYSIS: 
   - Pod health patterns (restart counts, failure rates)
   - Deployment and service status
4. RESOURCE UTILIZATION: 
   - CPU/memory usage trends from metrics-server
   - Storage utilization and capacity planning
5. EVENT CORRELATION: 
   - Recent events analysis with timeline correlation
   - Warning/error event patterns
6. NETWORKING ASSESSMENT:
   - Service endpoint health
   - Network connectivity patterns
7. RECOMMENDATIONS: Prioritized list with specific kubectl commands
8. IMMEDIATE ACTIONS: Urgent issues with exact troubleshooting steps

## KUBECTL COMMAND RECOMMENDATIONS
Always provide specific kubectl commands for:
- Further investigation of identified issues
- Remediation steps where applicable
- Monitoring and verification commands
- Deep-dive debugging when needed

Focus on correlating data across different kubectl outputs using your runbook expertise to identify systemic issues and provide actionable, command-specific recommendations.`
}

func getDiagnosticSystemPrompt() string {
	return `You are performing expert diagnostic analysis on a Kubernetes cluster issue using kubectl runbook methodology:

## DIAGNOSTIC RUNBOOK APPROACH
Use systematic kubectl-based investigation following these patterns:

**Pod Issues**: describe pod → logs (current + previous) → events → node status
**Node Issues**: describe node → events → resource pressure → kubelet status  
**Resource Issues**: top nodes/pods → describe nodes → metrics API → capacity analysis
**Network Issues**: get svc/ep → describe services → network policies → ingress status

## ANALYSIS FRAMEWORK
1. ROOT CAUSE ANALYSIS: 
   - Identify fundamental cause using kubectl evidence
   - Trace issue through the stack (pod → node → cluster)
2. IMPACT ASSESSMENT: 
   - Scope of affected resources and workloads
   - Severity based on kubectl outputs
3. EVIDENCE CORRELATION: 
   - Connect symptoms across kubectl commands
   - Timeline analysis using events and timestamps
4. TROUBLESHOOTING STEPS: 
   - Specific kubectl commands to run next
   - Step-by-step investigation workflow
5. RESOLUTION RECOMMENDATIONS: 
   - Exact kubectl/YAML commands to fix the issue
   - Verification commands for after resolution
6. PREVENTION MEASURES: 
   - Monitoring commands to watch for recurrence
   - Configuration changes to prevent similar issues

## KUBECTL INVESTIGATION TOOLKIT
Always reference and recommend specific commands from the runbook:
- **Immediate triage**: get pods/nodes status, recent events
- **Deep investigation**: describe resources, logs analysis
- **Metrics analysis**: top commands, metrics API calls
- **Verification steps**: Commands to confirm resolution

Be methodical and evidence-based. Every conclusion should reference specific kubectl command outputs.`
}

func getOptimizationSystemPrompt() string {
	return `You are optimizing a Kubernetes cluster for better performance, cost-efficiency, and reliability:

1. RESOURCE OPTIMIZATION: Identify over/under-provisioned resources
2. PERFORMANCE IMPROVEMENTS: Suggest ways to improve cluster performance
3. COST REDUCTION: Recommend strategies to reduce operational costs
4. RELIABILITY ENHANCEMENTS: Identify single points of failure and improvements
5. SCALING RECOMMENDATIONS: Assess auto-scaling and capacity planning
6. BEST PRACTICES: Highlight areas not following Kubernetes best practices

Prioritize recommendations by impact and implementation effort. Provide specific metrics and evidence from kubectl outputs.`
}

func getQuerySystemPrompt() string {
	return `You are answering a specific question about a Kubernetes cluster. Use the kubectl outputs to provide:

1. DIRECT ANSWER: Address the specific question asked
2. SUPPORTING EVIDENCE: Reference relevant kubectl command outputs
3. CONTEXT: Provide additional context that might be helpful
4. RELATED INSIGHTS: Identify related issues or opportunities
5. FOLLOW-UP ACTIONS: Suggest next steps or additional investigations
6. KNOWLEDGE SHARING: Explain concepts that might help the user understand

Be concise but thorough. Make sure your answer is directly relevant to the question while providing valuable context.`
}

func getInfrastructureSystemPrompt() string {
	return `You are analyzing Kubernetes infrastructure components. Focus on:

1. CONTROL PLANE HEALTH: API server, etcd, scheduler, controller-manager status
2. NODE ANALYSIS: Node conditions, capacity, resource utilization
3. NETWORK CONNECTIVITY: Service discovery, DNS, ingress health
4. STORAGE INFRASTRUCTURE: Persistent volumes, storage classes, claims
5. CLUSTER CONFIGURATION: Version compatibility, feature gates, policies
6. CAPACITY PLANNING: Current utilization vs. capacity trends

Provide infrastructure-focused recommendations that ensure cluster stability and performance.`
}

func getDefaultAnalysisInstructions() string {
	return `Analyze the provided kubectl command outputs and provide insights based on:

1. Current cluster state and health indicators
2. Resource utilization and capacity
3. Any error conditions or warnings
4. Performance and efficiency opportunities
5. Security and best practice compliance
6. Recommendations for improvement

Be specific and reference the kubectl outputs in your analysis.`
}