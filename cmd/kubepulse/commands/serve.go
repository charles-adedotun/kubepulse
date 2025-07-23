package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kubepulse/kubepulse/internal/config"
	"github.com/kubepulse/kubepulse/pkg/ai"
	"github.com/kubepulse/kubepulse/pkg/api"
	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/health"
	"github.com/kubepulse/kubepulse/pkg/k8s"
	"github.com/kubepulse/kubepulse/pkg/plugins"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

var (
	port       int
	apiOnly    bool
	webEnabled bool
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start KubePulse web server with dashboard and API",
	Long: `Serve starts the KubePulse web server providing:
- REST API endpoints for health data
- Real-time web dashboard with live updates
- WebSocket connections for streaming updates
- Prometheus-compatible metrics endpoint`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for web server")
	serveCmd.Flags().BoolVar(&apiOnly, "api-only", false, "Serve API only (no web dashboard)")
	serveCmd.Flags().BoolVar(&webEnabled, "web", true, "Enable web dashboard")
	serveCmd.Flags().DurationVarP(&interval, "interval", "i", 10*time.Second, "Health check interval")
	serveCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to monitor (empty for all)")
}

func runServe(cmd *cobra.Command, args []string) error {
	client := GetK8sClient()
	if client == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	// Create context manager
	kubeconfigPath := viper.GetString("kubeconfig")
	contextManager, err := k8s.NewContextManager(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to create context manager: %w", err)
	}

	// Get current context name
	currentContext := ""
	if ctx, err := contextManager.GetCurrentContext(); err == nil {
		currentContext = ctx.Name
	}

	// Load configuration
	var cfg *config.Config
	var err error
	
	// Check if config file is specified
	configFile := viper.GetString("config")
	if configFile != "" {
		cfg, err = config.LoadConfig(configFile)
	} else {
		cfg, err = config.LoadConfig("")
	}
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with command line flags if provided
	if cmd.Flags().Changed("port") {
		cfg.Server.Port = port
	}
	if cmd.Flags().Changed("web") {
		cfg.Server.EnableWeb = webEnabled
	}
	if cmd.Flags().Changed("interval") {
		cfg.Monitoring.Interval = interval
	}

	// Create channels for alerts and metrics
	alertChan := make(chan core.Alert, 100)
	metricsChan := make(chan core.Metric, 1000)

	// Create monitoring engine with AI enabled
	aiConfig := ai.Config{
		ClaudePath: "claude", // Assume claude is in PATH
		MaxTurns:   3,
	}

	engineConfig := core.EngineConfig{
		KubeClient:     client,
		ContextName:    currentContext,
		Interval:       interval,
		AlertChan:      alertChan,
		MetricsChan:    metricsChan,
		EnableAI:       true,
		AIConfig:       &aiConfig,
	}
	engine := core.NewEngine(engineConfig)

	// Register health checks
	registry := plugins.NewRegistry()

	// Add pod health check
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

	// Add node health check
	nodeCheck := health.NewNodeHealthCheck()
	if err := registry.Register(nodeCheck); err != nil {
		return fmt.Errorf("failed to register node check: %w", err)
	}

	// Add service health check
	serviceCheck := health.NewServiceHealthCheck()
	if namespace != "" {
		if err := serviceCheck.Configure(map[string]interface{}{
			"namespace": namespace,
		}); err != nil {
			return fmt.Errorf("failed to configure service check: %w", err)
		}
	}
	if err := registry.Register(serviceCheck); err != nil {
		return fmt.Errorf("failed to register service check: %w", err)
	}

	// Add all checks to engine
	for _, check := range registry.List() {
		engine.AddCheck(check)
	}

	// Create API server with configuration
	serverConfig := api.Config{
		Port:           cfg.Server.Port,
		Engine:         engine,
		ContextManager: contextManager,
		Host:           cfg.Server.Host,
		CORSEnabled:    cfg.Server.CORSEnabled,
		CORSOrigins:    cfg.Server.CORSOrigins,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		UIConfig:     cfg.UI,
	}
	apiServer := api.NewServer(serverConfig)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start monitoring engine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := engine.Start(); err != nil {
			klog.Errorf("Monitoring engine error: %v", err)
		}
	}()

	// Start API server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.Start(); err != nil {
			klog.Errorf("API server error: %v", err)
		}
	}()

	// Start broadcasting updates to WebSocket clients
	wg.Add(1)
	go func() {
		defer wg.Done()
		broadcastTicker := time.NewTicker(cfg.UI.RefreshInterval)
		defer broadcastTicker.Stop()

		for {
			select {
			case <-broadcastTicker.C:
				health := engine.GetClusterHealth("default")
				apiServer.BroadcastToClients(health)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Handle alert and metrics channels
	go handleAlerts(alertChan)
	go handleMetrics(metricsChan)

	// Display startup information
	displayStartupInfo(cfg)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n🛑 Shutting down KubePulse server...")

	// Stop components gracefully
	cancel()

	// Stop API server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		klog.Errorf("Error stopping API server: %v", err)
	}

	// Stop monitoring engine
	engine.Stop()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("✅ KubePulse server stopped gracefully")
	case <-time.After(10 * time.Second):
		fmt.Println("⚠️  Timeout waiting for graceful shutdown")
	}

	return nil
}

func displayStartupInfo(cfg *config.Config) {
	fmt.Printf("\n🚀 KubePulse Server Starting...\n")
	fmt.Printf("┌─────────────────────────────────────────┐\n")
	fmt.Printf("│  Port: %d                               │\n", cfg.Server.Port)
	fmt.Printf("│  Mode: %s                          │\n", getServerMode(!cfg.Server.EnableWeb, cfg.Server.EnableWeb))
	fmt.Printf("│  Monitoring Interval: %s                │\n", cfg.Monitoring.Interval.String())
	fmt.Printf("│  UI Refresh Interval: %s               │\n", cfg.UI.RefreshInterval.String())
	fmt.Printf("│  CORS: %v                              │\n", cfg.Server.CORSEnabled)
	fmt.Printf("└─────────────────────────────────────────┘\n\n")

	// Display feature flags
	fmt.Printf("📊 UI Features:\n")
	fmt.Printf("   • AI Insights:           %v\n", cfg.UI.Features.AIInsights)
	fmt.Printf("   • Predictive Analytics:  %v\n", cfg.UI.Features.PredictiveAnalytics)
	fmt.Printf("   • Smart Alerts:          %v\n", cfg.UI.Features.SmartAlerts)
	fmt.Printf("   • Node Details:          %v\n", cfg.UI.Features.NodeDetails)

	fmt.Printf("\n📡 API Endpoints:\n")
	fmt.Printf("   • Health Status:     http://localhost:%d/api/v1/health\n", cfg.Server.Port)
	fmt.Printf("   • Cluster Health:    http://localhost:%d/api/v1/health/cluster\n", cfg.Server.Port)
	fmt.Printf("   • Health Checks:     http://localhost:%d/api/v1/health/checks\n", cfg.Server.Port)
	fmt.Printf("   • Prometheus Metrics: http://localhost:%d/api/v1/metrics\n", cfg.Server.Port)

	if cfg.Server.EnableWeb {
		fmt.Printf("\n🌐 Web Dashboard:\n")
		fmt.Printf("   • Dashboard:         http://localhost:%d\n", cfg.Server.Port)
		fmt.Printf("   • WebSocket:         ws://localhost:%d/ws\n", cfg.Server.Port)
	}

	fmt.Printf("\n💡 Press Ctrl+C to stop the server\n\n")
}

func getServerMode(apiOnly, webEnabled bool) string {
	if apiOnly {
		return "API Only          "
	}
	if webEnabled {
		return "API + Web Dashboard"
	}
	return "API Only          "
}
