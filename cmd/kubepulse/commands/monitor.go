package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/health"
	"github.com/kubepulse/kubepulse/pkg/plugins"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	interval      time.Duration
	outputFormat  string
	watch         bool
	namespace     string
	enabledChecks []string
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start monitoring Kubernetes cluster health",
	Long: `Monitor continuously checks the health of your Kubernetes cluster
and reports issues in real-time. It runs various health checks including
pod status, node conditions, and resource usage.`,
	RunE: runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().DurationVarP(&interval, "interval", "i", 30*time.Second, "Check interval")
	monitorCmd.Flags().StringVarP(&outputFormat, "output", "o", "summary", "Output format (summary, json, yaml)")
	monitorCmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch mode - continuous monitoring")
	monitorCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to monitor (empty for all)")
	monitorCmd.Flags().StringSliceVar(&enabledChecks, "checks", []string{"pod-health", "node-health"}, "Enabled health checks")
}

func runMonitor(cmd *cobra.Command, args []string) error {
	client := GetK8sClient()
	if client == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	// Create channels for alerts and metrics
	alertChan := make(chan core.Alert, 100)
	metricsChan := make(chan core.Metric, 1000)

	// Create monitoring engine
	engineConfig := core.EngineConfig{
		KubeClient:  client,
		Interval:    interval,
		AlertChan:   alertChan,
		MetricsChan: metricsChan,
	}
	engine := core.NewEngine(engineConfig)

	// Create and register health checks
	registry := plugins.NewRegistry()

	// Register built-in checks
	podCheck := health.NewPodHealthCheck()
	if namespace != "" {
		if err := podCheck.Configure(map[string]interface{}{
			"namespace": namespace,
		}); err != nil {
			return fmt.Errorf("failed to configure pod check: %w", err)
		}
	}
	if err := registry.Register(podCheck); err != nil {
		return fmt.Errorf("failed to register pod check: %w", err)
	}

	nodeCheck := health.NewNodeHealthCheck()
	if err := registry.Register(nodeCheck); err != nil {
		return fmt.Errorf("failed to register node check: %w", err)
	}

	// Add enabled checks to the engine
	for _, checkName := range enabledChecks {
		check, err := registry.Get(checkName)
		if err != nil {
			klog.Warningf("Check %s not found: %v", checkName, err)
			continue
		}
		engine.AddCheck(check)
	}

	// Start alert handler
	go handleAlerts(alertChan)

	// Start metrics handler
	go handleMetrics(metricsChan)

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring
	go func() {
		if err := engine.Start(); err != nil {
			klog.Errorf("Engine error: %v", err)
			cancel()
		}
	}()

	// Run once or watch mode
	if !watch {
		// Wait for one check cycle
		time.Sleep(2 * time.Second)
		displayResults(engine)
		return nil
	}

	// Watch mode - continuous monitoring
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Println("Starting continuous monitoring... (Press Ctrl+C to stop)")
	fmt.Println()

	// Initial display
	time.Sleep(2 * time.Second)
	displayResults(engine)

	for {
		select {
		case <-ticker.C:
			displayResults(engine)
		case <-sigChan:
			fmt.Println("\nShutting down...")
			engine.Stop()
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

func displayResults(engine *core.Engine) {
	health := engine.GetClusterHealth("default")

	switch outputFormat {
	case "json":
		// TODO: Implement JSON output
		fmt.Println("JSON output not yet implemented")
	case "yaml":
		// TODO: Implement YAML output
		fmt.Println("YAML output not yet implemented")
	default:
		displaySummary(health)
	}
}

func displaySummary(health core.ClusterHealth) {
	// Clear screen in watch mode
	if watch {
		fmt.Print("\033[H\033[2J")
	}

	// Display header
	fmt.Printf("=== KubePulse Health Report - %s ===\n", health.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Display overall status with color
	statusColor := getStatusColor(health.Status)
	fmt.Printf("Overall Status: %s%s%s\n", statusColor, health.Status, resetColor)
	fmt.Printf("Health Score: %.1f%% (weighted: %.1f%%)\n", health.Score.Raw, health.Score.Weighted)
	fmt.Printf("Confidence: %.1f%%\n", health.Score.Confidence*100)
	fmt.Println()

	// Display individual checks
	fmt.Println("Health Checks:")
	fmt.Println("--------------")
	for _, check := range health.Checks {
		checkColor := getStatusColor(check.Status)
		symbol := getStatusSymbol(check.Status)
		fmt.Printf("%s %s%-15s%s %s\n", symbol, checkColor, check.Name, resetColor, check.Message)

		// Show important details
		if check.Status != core.HealthStatusHealthy && len(check.Details) > 0 {
			if issues, ok := check.Details["issues"].([]string); ok && len(issues) > 0 {
				for i, issue := range issues {
					if i >= 3 {
						fmt.Printf("      ... and %d more issues\n", len(issues)-3)
						break
					}
					fmt.Printf("      - %s\n", issue)
				}
			}
		}
	}

	// Display active alerts
	if len(health.Alerts) > 0 {
		fmt.Println()
		fmt.Println("Active Alerts:")
		fmt.Println("--------------")
		for _, alert := range health.Alerts {
			severityColor := getSeverityColor(alert.Severity)
			fmt.Printf("%s[%s]%s %s: %s\n", severityColor, alert.Severity, resetColor, alert.Name, alert.Message)
		}
	}

	fmt.Println()
}

func handleAlerts(alertChan <-chan core.Alert) {
	for alert := range alertChan {
		// In a real implementation, this would send to configured channels
		klog.V(2).Infof("Alert: [%s] %s - %s", alert.Severity, alert.Name, alert.Message)
	}
}

func handleMetrics(metricsChan <-chan core.Metric) {
	for metric := range metricsChan {
		// In a real implementation, this would export to monitoring systems
		klog.V(3).Infof("Metric: %s = %f %s", metric.Name, metric.Value, metric.Unit)
	}
}

// Color constants
const (
	resetColor  = "\033[0m"
	redColor    = "\033[31m"
	yellowColor = "\033[33m"
	greenColor  = "\033[32m"
	blueColor   = "\033[34m"
)

func getStatusColor(status core.HealthStatus) string {
	switch status {
	case core.HealthStatusHealthy:
		return greenColor
	case core.HealthStatusDegraded:
		return yellowColor
	case core.HealthStatusUnhealthy:
		return redColor
	default:
		return blueColor
	}
}

func getStatusSymbol(status core.HealthStatus) string {
	switch status {
	case core.HealthStatusHealthy:
		return "✓"
	case core.HealthStatusDegraded:
		return "⚠"
	case core.HealthStatusUnhealthy:
		return "✗"
	default:
		return "?"
	}
}

func getSeverityColor(severity core.AlertSeverity) string {
	switch severity {
	case core.AlertSeverityCritical:
		return redColor
	case core.AlertSeverityWarning:
		return yellowColor
	default:
		return blueColor
	}
}
