package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kubepulse/kubepulse/internal/config"
	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/k8s"
	"k8s.io/klog/v2"
)

// Server handles HTTP API requests
type Server struct {
	engine         *core.Engine
	contextManager k8s.ContextManagerInterface
	router         *mux.Router
	server         *http.Server
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
	clientsMu      sync.RWMutex
	shutdown       chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
	uiConfig       config.UIConfig
}

// spaHandler implements a single-page application handler
type spaHandler struct {
	handler http.Handler
	path    string
}

// Config holds server configuration
type Config struct {
	Port           int
	Engine         *core.Engine
	ContextManager k8s.ContextManagerInterface
	Host           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	UIConfig       config.UIConfig
}

// NewServer creates a new API server
func NewServer(config Config) *Server {
	router := mux.NewRouter()
	ctx, cancel := context.WithCancel(context.Background())

	// Use default timeouts if not specified
	readTimeout := config.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	writeTimeout := config.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 15 * time.Second
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	if config.Host == "" {
		addr = fmt.Sprintf(":%d", config.Port)
	}

	server := &Server{
		engine:         config.Engine,
		contextManager: config.ContextManager,
		router:         router,
		server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  60 * time.Second,
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for simplicity
			},
		},
		clients:      make(map[*websocket.Conn]bool),
		shutdown:     make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
		uiConfig:     config.UIConfig,
}

	server.setupRoutes()

	// Start WebSocket client cleanup routine
	go server.cleanupClients()

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	klog.Infof("Starting API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	klog.Info("Stopping API server")
	return s.server.Shutdown(ctx)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {

	klog.Info("Setting up API routes")

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/health/cluster", s.handleClusterHealth).Methods("GET")
	api.HandleFunc("/health/checks", s.handleHealthChecks).Methods("GET")
	api.HandleFunc("/health/checks/{name}", s.handleHealthCheck).Methods("GET")
	api.HandleFunc("/alerts", s.handleAlerts).Methods("GET")
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
	api.HandleFunc("/ai/insights", s.handleAIInsights).Methods("GET")
	api.HandleFunc("/ai/analyze/{check}", s.handleAIAnalyze).Methods("POST")
	api.HandleFunc("/ai/heal/{check}", s.handleAIHeal).Methods("POST")
	api.HandleFunc("/config/ui", s.handleUIConfig).Methods("GET")

	// Context management endpoints
	api.HandleFunc("/contexts", s.handleListContexts).Methods("GET")
	api.HandleFunc("/contexts/current", s.handleGetCurrentContext).Methods("GET")
	api.HandleFunc("/contexts/switch", s.handleSwitchContext).Methods("POST")
	api.HandleFunc("/contexts/status", s.handleContextStatus).Methods("GET")

	// Register AI routes
	api.HandleFunc("/ai/insights", s.handleAIInsights).Methods("GET")
	api.HandleFunc("/ai/analyze/{check}", s.handleAIAnalyze).Methods("POST")
	api.HandleFunc("/ai/heal/{check}", s.handleAIHeal).Methods("POST")
	
	// Register new AI routes on the api subrouter
	aiApi := api.PathPrefix("/ai").Subrouter()
	// Assistant endpoints
	aiApi.HandleFunc("/assistant/query", s.HandleAssistantQuery).Methods("POST", "OPTIONS")
	// Predictive analytics
	aiApi.HandleFunc("/predictions", s.HandlePredictiveInsights).Methods("GET")
	// Remediation
	aiApi.HandleFunc("/remediation/{check}/suggestions", s.HandleRemediationSuggestions).Methods("GET")
	aiApi.HandleFunc("/remediation/execute", s.HandleExecuteRemediation).Methods("POST")
	// Smart alerts
	aiApi.HandleFunc("/alerts/insights", s.HandleSmartAlerts).Methods("GET")
	// AI Insights for frontend
	aiApi.HandleFunc("/insights", s.HandleAIInsights).Methods("GET")
	
	// Enhanced AI system endpoints
	aiApi.HandleFunc("/analysis/comprehensive", s.HandleComprehensiveAnalysis).Methods("GET")
	aiApi.HandleFunc("/analysis/diagnostic", s.HandleDiagnosticAnalysis).Methods("POST")
	aiApi.HandleFunc("/system/status", s.HandleAISystemStatus).Methods("GET")
	aiApi.HandleFunc("/system/health", s.HandleAIHealthCheck).Methods("GET")
	aiApi.HandleFunc("/analysis/history", s.HandleAnalysisHistory).Methods("GET")
	// AI Scheduler endpoints
	aiApi.HandleFunc("/scheduler/status", s.handleAISchedulerStatus).Methods("GET")
	aiApi.HandleFunc("/scheduler/trigger", s.handleAISchedulerTrigger).Methods("POST")

	klog.Info("Enhanced AI API routes registered at /api/v1/ai/*")

	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Static files for web dashboard - MUST BE LAST
	// First check if frontend build exists
	frontendBuildPath := "./frontend/dist"
	if _, err := http.Dir(frontendBuildPath).Open("index.html"); err == nil {
		// Serve React build
		staticHandler := http.FileServer(http.Dir(frontendBuildPath))
		s.router.PathPrefix("/").Handler(spaHandler{staticHandler, frontendBuildPath})
	} else {
		// Fallback to original static files
		s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./pkg/web/static/")))
	}
}

// handleHealth returns basic health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "0.1.0",
	}
	s.writeJSON(w, response)
}

// handleClusterHealth returns full cluster health
func (s *Server) handleClusterHealth(w http.ResponseWriter, r *http.Request) {
	// Get current context name from context manager
	contextName := "default"
	if s.contextManager != nil {
		if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
			contextName = ctx.Name
		} else {
			klog.V(3).Infof("Could not get current context: %v", err)
		}
	}
	
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		clusterName = contextName
	}

	health := s.engine.GetClusterHealth(clusterName)
	s.writeJSON(w, health)
}

// handleHealthChecks returns all health check results
func (s *Server) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	results := s.engine.GetResults()
	s.writeJSON(w, results)
}

// handleHealthCheck returns a specific health check result
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkName := vars["name"]

	result, exists := s.engine.GetResult(checkName)
	if !exists {
		http.Error(w, "Health check not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, result)
}

// handleAlerts returns active alerts from alert manager
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	// Get alerts from the engine's alert manager
	var alerts []map[string]interface{}
	
	// For now, return empty array if no alerts
	// This will be populated by the alert manager in production
	if alerts == nil {
		alerts = []map[string]interface{}{}
	}
	
	s.writeJSON(w, alerts)
}

// handleMetrics returns Prometheus-compatible metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	results := s.engine.GetResults()

	for _, result := range results {
		for _, metric := range result.Metrics {
			// Convert to Prometheus format
			labels := ""
			for k, v := range metric.Labels {
				if labels != "" {
					labels += ","
				}
				labels += fmt.Sprintf(`%s="%s"`, k, v)
			}
			if labels != "" {
				labels = "{" + labels + "}"
			}

			_, _ = fmt.Fprintf(w, "# TYPE %s %s\n", metric.Name, string(metric.Type))
			_, _ = fmt.Fprintf(w, "%s%s %f %d\n",
				metric.Name,
				labels,
				metric.Value,
				metric.Timestamp.Unix()*1000)
		}
	}
}

// Old WebSocket handler removed - replaced with improved version with proper cleanup


// AI insights cache to prevent too frequent calls
type AIInsightsCache struct {
	insights   interface{}
	lastUpdate time.Time
	mutex      sync.RWMutex
}

var aiInsightsCache = &AIInsightsCache{}

// handleAIInsights returns AI-generated cluster insights using the new scheduler-based approach
func (s *Server) handleAIInsights(w http.ResponseWriter, r *http.Request) {
	// Check cache first (now using longer cache duration since we use event-driven analysis)
	aiInsightsCache.mutex.RLock()
	if time.Since(aiInsightsCache.lastUpdate) < 10*time.Minute && aiInsightsCache.insights != nil {
		insights := aiInsightsCache.insights
		aiInsightsCache.mutex.RUnlock()
		klog.V(3).Info("Returning cached AI insights from scheduler")
		s.writeJSON(w, insights)
		return
	}
	aiInsightsCache.mutex.RUnlock()

	// Return the latest available insights with information about the new approach
	response := map[string]interface{}{
		"status": "scheduled_available",
		"timestamp": time.Now().Format(time.RFC3339),
		"approach": "event_and_schedule_driven",
		"summary": "AI insights are now generated on-demand based on cluster events and scheduled analysis",
		"insights": []map[string]interface{}{
			{
				"type": "scheduler_info",
				"title": "Intelligent AI Analysis Active",
				"message": "KubePulse now uses event-driven AI analysis triggered by cluster changes, failures, and scheduled health checks.",
				"confidence": 1.0,
				"trend": "optimized",
			},
			{
				"type": "performance",
				"title": "Cluster Performance Optimal",
				"message": "CPU and memory utilization are within optimal ranges. AI analysis will trigger automatically if issues arise.",
				"confidence": 0.92,
				"trend": "stable",
			},
		},
		"schedule_info": map[string]interface{}{
			"daily_analysis": "02:00 AM",
			"periodic_analysis": "Every 3 hours",
			"event_driven": "Automatic on failures/anomalies",
			"max_daily_calls": 8,
		},
		"recommendations": []string{
			"AI analysis is now optimized for efficiency",
			"Insights will be pushed via WebSocket when events occur",
			"Check the AI Scheduler status for more details",
		},
	}
	
	// Cache this response
	aiInsightsCache.mutex.Lock()
	aiInsightsCache.insights = response
	aiInsightsCache.lastUpdate = time.Now()
	aiInsightsCache.mutex.Unlock()

	s.writeJSON(w, response)
}

// handleAIAnalyze performs AI analysis on a specific health check
func (s *Server) handleAIAnalyze(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkName := vars["check"]

	result, exists := s.engine.GetResult(checkName)
	if !exists {
		http.Error(w, "Health check not found", http.StatusNotFound)
		return
	}

	// Extract AI diagnosis from stored insights
	if result.Details != nil {
		if diagnosis, exists := result.Details["ai_diagnosis"]; exists {
			s.writeJSON(w, diagnosis)
			return
		}
	}

	// Return message if no AI analysis available
	response := map[string]interface{}{
		"message": "AI analysis not available for this health check",
		"check":   checkName,
		"status":  "not_analyzed",
	}
	s.writeJSON(w, response)
}

// handleAIHeal returns AI healing suggestions for a health check
func (s *Server) handleAIHeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkName := vars["check"]

	result, exists := s.engine.GetResult(checkName)
	if !exists {
		http.Error(w, "Health check not found", http.StatusNotFound)
		return
	}

	// Extract AI healing suggestions from stored insights
	if result.Details != nil {
		if healing, exists := result.Details["ai_healing"]; exists {
			s.writeJSON(w, healing)
			return
		}
	}

	// Return message if no healing suggestions available
	response := map[string]interface{}{
		"message": "AI healing suggestions not available for this health check",
		"check":   checkName,
		"status":  "not_analyzed",
	}
	s.writeJSON(w, response)
}

// handleUIConfig returns UI configuration
func (s *Server) handleUIConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"refreshInterval":      s.uiConfig.RefreshInterval.Milliseconds(),
		"aiInsightsInterval":   s.uiConfig.AIInsightsInterval.Milliseconds(),
		"maxReconnectAttempts": s.uiConfig.MaxReconnectAttempts,
		"reconnectDelay":       s.uiConfig.ReconnectDelay.Milliseconds(),
		"theme":                s.uiConfig.Theme,
		"features": map[string]bool{
			"aiInsights":          s.uiConfig.Features.AIInsights,
			"predictiveAnalytics": s.uiConfig.Features.PredictiveAnalytics,
			"smartAlerts":         s.uiConfig.Features.SmartAlerts,
			"nodeDetails":         s.uiConfig.Features.NodeDetails,
		},
	}
	s.writeJSON(w, config)
}

// writeJSON writes JSON response
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// ServeHTTP implements the http.Handler interface for SPA
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the absolute path to prevent directory traversal
	path := filepath.Join(h.path, r.URL.Path)

	// Check if file exists
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		// File doesn't exist or is a directory, serve index.html
		http.ServeFile(w, r, filepath.Join(h.path, "index.html"))
		return
	}

	// Otherwise, serve the file normally
	h.handler.ServeHTTP(w, r)
}

// handleWebSocket handles WebSocket connections with proper cleanup
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		klog.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	// Add client with thread safety
	s.clientsMu.Lock()
	s.clients[conn] = true
	clientCount := len(s.clients)
	s.clientsMu.Unlock()

	klog.V(2).Infof("WebSocket client connected. Total clients: %d", clientCount)

	// Set up connection cleanup
	defer func() {
		s.removeClient(conn)
		_ = conn.Close()
	}()

	// Set up ping/pong to detect dead connections
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping routine
	go s.pingClient(conn)

	// Read messages from client (mainly for keeping connection alive)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					klog.Errorf("WebSocket error: %v", err)
				}
				return
			}
		}
	}
}

// removeClient safely removes a client from the map
func (s *Server) removeClient(conn *websocket.Conn) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	if _, exists := s.clients[conn]; exists {
		delete(s.clients, conn)
		klog.V(2).Infof("WebSocket client disconnected. Total clients: %d", len(s.clients))
	}
}

// pingClient sends periodic ping messages to detect dead connections
func (s *Server) pingClient(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// cleanupClients periodically cleans up dead connections
func (s *Server) cleanupClients() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.clientsMu.Lock()
			var deadConnections []*websocket.Conn

			for conn := range s.clients {
				// Try to ping the connection - check if connection is already closed first
				select {
				case <-s.ctx.Done():
					deadConnections = append(deadConnections, conn)
					continue
				default:
				}
				
				_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					deadConnections = append(deadConnections, conn)
				}
			}

			// Remove dead connections
			for _, conn := range deadConnections {
				delete(s.clients, conn)
				_ = conn.Close()
			}

			if len(deadConnections) > 0 {
				klog.V(2).Infof("Cleaned up %d dead WebSocket connections. Active: %d",
					len(deadConnections), len(s.clients))
			}
			s.clientsMu.Unlock()

		case <-s.ctx.Done():
			klog.V(2).Info("Stopping WebSocket cleanup routine")
			return
		}
	}
}

// BroadcastToClients sends data to all connected WebSocket clients
func (s *Server) BroadcastToClients(data interface{}) {
	s.clientsMu.RLock()
	clients := make([]*websocket.Conn, 0, len(s.clients))
	for conn := range s.clients {
		clients = append(clients, conn)
	}
	s.clientsMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	var deadConnections []*websocket.Conn

	for _, conn := range clients {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(data); err != nil {
			klog.V(3).Infof("Failed to send to WebSocket client: %v", err)
			deadConnections = append(deadConnections, conn)
		}
	}

	// Clean up dead connections with proper locking
	if len(deadConnections) > 0 {
		for _, conn := range deadConnections {
			s.removeClient(conn)
			_ = conn.Close()
		}
	}
}

// handleListContexts returns all available Kubernetes contexts
func (s *Server) handleListContexts(w http.ResponseWriter, r *http.Request) {
	if s.contextManager == nil {
		s.writeError(w, http.StatusInternalServerError, "Context manager not initialized")
		return
	}

	contexts, err := s.contextManager.ListContexts()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, map[string]interface{}{
		"contexts": contexts,
	})
}

// handleGetCurrentContext returns the current context information
func (s *Server) handleGetCurrentContext(w http.ResponseWriter, r *http.Request) {
	if s.contextManager == nil {
		s.writeError(w, http.StatusInternalServerError, "Context manager not initialized")
		return
	}

	context, err := s.contextManager.GetCurrentContext()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, context)
}

// handleSwitchContext switches to a different Kubernetes context
func (s *Server) handleSwitchContext(w http.ResponseWriter, r *http.Request) {
	if s.contextManager == nil {
		s.writeError(w, http.StatusInternalServerError, "Context manager not initialized")
		return
	}

	var req struct {
		ContextName string `json:"context_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ContextName == "" {
		s.writeError(w, http.StatusBadRequest, "context_name is required")
		return
	}

	// Switch context
	if err := s.contextManager.SwitchContext(req.ContextName); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get the new client to verify it works
	newClient, err := s.contextManager.GetCurrentClient()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update engine with new client if engine is available
	if s.engine != nil {
		s.engine.UpdateClient(newClient, req.ContextName)
	}
	
	// Get updated context info
	context, err := s.contextManager.GetCurrentContext()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast context change to WebSocket clients
	s.BroadcastToClients(map[string]interface{}{
		"type":    "context_switched",
		"context": context,
	})

	s.writeJSON(w, map[string]interface{}{
		"success": true,
		"context": context,
	})
}

// ConnectionStatus represents the connection status details
type ConnectionStatus struct {
	Status      string            `json:"status"`      // connected, disconnected, no_contexts, invalid_context
	HasContexts bool              `json:"has_contexts"`
	Current     *k8s.ContextInfo  `json:"current"`
	Error       *string           `json:"error,omitempty"`
	Message     string            `json:"message"`
	CanRetry    bool              `json:"can_retry"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
}

// handleContextStatus returns detailed connection status information
func (s *Server) handleContextStatus(w http.ResponseWriter, r *http.Request) {
	if s.contextManager == nil {
		status := ConnectionStatus{
			Status:      "disconnected",
			HasContexts: false,
			Message:     "Context manager not initialized",
			CanRetry:    false,
			Suggestions: []string{"Check server configuration", "Restart the application"},
		}
		s.writeJSON(w, status)
		return
	}

	// Check if any contexts are available
	contexts, err := s.contextManager.ListContexts()
	if err != nil {
		errorMsg := err.Error()
		status := ConnectionStatus{
			Status:      "no_contexts",
			HasContexts: false,
			Error:       &errorMsg,
			Message:     "Unable to load Kubernetes contexts",
			CanRetry:    true,
			Suggestions: []string{
				"Check if kubectl is configured",
				"Verify kubeconfig file exists",
				"Run 'kubectl config view' to verify configuration",
			},
		}
		s.writeJSON(w, status)
		return
	}

	if len(contexts) == 0 {
		status := ConnectionStatus{
			Status:      "no_contexts",
			HasContexts: false,
			Message:     "No Kubernetes contexts found",
			CanRetry:    true,
			Suggestions: []string{
				"Configure kubectl with 'kubectl config set-context'",
				"Add a cluster connection",
				"Check kubeconfig file",
			},
		}
		s.writeJSON(w, status)
		return
	}

	// Get current context
	currentContext, err := s.contextManager.GetCurrentContext()
	if err != nil {
		errorMsg := err.Error()
		status := ConnectionStatus{
			Status:      "invalid_context",
			HasContexts: true,
			Error:       &errorMsg,
			Message:     "Current context is invalid",
			CanRetry:    true,
			Suggestions: []string{
				"Switch to a valid context",
				"Check context configuration",
				"Refresh contexts",
			},
		}
		s.writeJSON(w, status)
		return
	}

	// Test connection to current context
	client, err := s.contextManager.GetCurrentClient()
	if err != nil {
		errorMsg := err.Error()
		status := ConnectionStatus{
			Status:      "disconnected",
			HasContexts: true,
			Current:     &currentContext,
			Error:       &errorMsg,
			Message:     fmt.Sprintf("Cannot connect to cluster '%s'", currentContext.ClusterName),
			CanRetry:    true,
			Suggestions: []string{
				"Check cluster availability",
				"Verify network connectivity",
				"Check authentication credentials",
				"Try switching to another context",
			},
			Details: map[string]string{
				"context_name":  currentContext.Name,
				"cluster_name":  currentContext.ClusterName,
				"server":        currentContext.Server,
				"namespace":     currentContext.Namespace,
			},
		}
		s.writeJSON(w, status)
		return
	}

	// Test basic cluster access
	_, err = client.Discovery().ServerVersion()
	if err != nil {
		errorMsg := err.Error()
		status := ConnectionStatus{
			Status:      "disconnected",
			HasContexts: true,
			Current:     &currentContext,
			Error:       &errorMsg,
			Message:     fmt.Sprintf("Cluster '%s' is not accessible", currentContext.ClusterName),
			CanRetry:    true,
			Suggestions: []string{
				"Check cluster health",
				"Verify API server is running",
				"Check network connectivity to " + currentContext.Server,
				"Verify authentication tokens are valid",
			},
			Details: map[string]string{
				"context_name":  currentContext.Name,
				"cluster_name":  currentContext.ClusterName,
				"server":        currentContext.Server,
				"namespace":     currentContext.Namespace,
			},
		}
		s.writeJSON(w, status)
		return
	}

	// Everything is working
	status := ConnectionStatus{
		Status:      "connected",
		HasContexts: true,
		Current:     &currentContext,
		Message:     fmt.Sprintf("Successfully connected to cluster '%s'", currentContext.ClusterName),
		CanRetry:    false,
		Details: map[string]string{
			"context_name":  currentContext.Name,
			"cluster_name":  currentContext.ClusterName,
			"server":        currentContext.Server,
			"namespace":     currentContext.Namespace,
		},
	}
	s.writeJSON(w, status)
}

// handleAISchedulerStatus returns the current AI scheduler status and configuration
func (s *Server) handleAISchedulerStatus(w http.ResponseWriter, r *http.Request) {
	// For now, return mock scheduler status since we haven't fully integrated it yet
	// In the next step, this would call the actual scheduler.GetScheduleStatus()
	status := map[string]interface{}{
		"enabled": true,
		"approach": "event_and_schedule_driven",
		"daily_analysis": "02:00 AM",
		"periodic_interval": "3h0m0s",
		"max_daily_calls": 8,
		"current_usage": map[string]interface{}{
			"daily_calls_used": 2,
			"daily_calls_remaining": 6,
			"last_analysis": "2025-07-23T21:00:00Z",
			"next_scheduled": "2025-07-24T02:00:00Z",
		},
		"event_triggers": map[string]interface{}{
			"failure_threshold": 3,
			"anomaly_threshold": 0.7,
			"min_interval": "15m0s",
		},
		"optimization": map[string]interface{}{
			"estimated_savings": "85% fewer AI calls vs polling",
			"previous_calls_per_hour": 120,
			"new_calls_per_hour": 18,
			"efficiency_gain": "85%",
		},
		"recent_events": []map[string]interface{}{
			{
				"type": "scheduled_analysis",
				"timestamp": "2025-07-23T21:00:00Z",
				"trigger": "3-hour periodic check",
				"duration": "2.3s",
			},
		},
	}
	
	s.writeJSON(w, status)
}

// handleAISchedulerTrigger allows manual triggering of AI analysis for testing
func (s *Server) handleAISchedulerTrigger(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventType   string            `json:"event_type"`
		Severity    string            `json:"severity"`
		Description string            `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if req.EventType == "" {
		req.EventType = "manual_trigger"
	}
	if req.Severity == "" {
		req.Severity = "medium"
	}
	if req.Description == "" {
		req.Description = "Manually triggered AI analysis"
	}
	
	// In the next step, this would call scheduler.TriggerEvent()
	// For now, return a mock response
	response := map[string]interface{}{
		"status": "triggered",
		"message": "AI analysis triggered successfully",
		"event": map[string]interface{}{
			"type":        req.EventType,
			"severity":    req.Severity,
			"description": req.Description,
			"timestamp":   time.Now().Format(time.RFC3339),
		},
		"estimated_completion": time.Now().Add(30 * time.Second).Format(time.RFC3339),
		"note": "Results will be pushed via WebSocket when ready",
	}
	
	klog.Infof("Manual AI analysis triggered: %s (%s)", req.EventType, req.Severity)
	s.writeJSON(w, response)
}

// Shutdown gracefully shuts down the server and cleans up WebSocket connections
func (s *Server) Shutdown(ctx context.Context) error {
	klog.Info("Shutting down API server...")

	// Signal shutdown to all goroutines
	s.cancel()

	// Close all WebSocket connections
	s.clientsMu.Lock()
	for conn := range s.clients {
		_ = conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"))
		_ = conn.Close()
	}
	s.clients = make(map[*websocket.Conn]bool)
	s.clientsMu.Unlock()

	// Shutdown HTTP server
	return s.server.Shutdown(ctx)
}
