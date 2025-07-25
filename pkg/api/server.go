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
	contextManager *k8s.ContextManager
	router         *mux.Router
	server         *http.Server
	upgrader       websocket.Upgrader
	clients        map[*websocket.Conn]bool
	clientsMu      sync.RWMutex
	shutdown       chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
	corsEnabled    bool
	corsOrigins    []string
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
	ContextManager *k8s.ContextManager
	Host           string
	CORSEnabled    bool
	CORSOrigins    []string
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
				if !config.CORSEnabled {
					return false
				}
				if len(config.CORSOrigins) == 0 || config.CORSOrigins[0] == "*" {
					return true
				}
				// Check specific origins
				origin := r.Header.Get("Origin")
				for _, allowed := range config.CORSOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
		},
		clients:     make(map[*websocket.Conn]bool),
		shutdown:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		corsEnabled: config.CORSEnabled,
		corsOrigins: config.CORSOrigins,
		uiConfig:    config.UIConfig,
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
	// Add CORS middleware first
	s.router.Use(s.corsMiddleware)

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

	klog.Info("AI API routes registered at /api/v1/ai/*")

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
	contextName := ""
	if s.contextManager != nil {
		if ctx, err := s.contextManager.GetCurrentContext(); err == nil {
			contextName = ctx.Name
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

// handleAlerts returns active alerts (placeholder)
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement alert history from alert manager
	alerts := []map[string]interface{}{
		{
			"id":        "example-1",
			"severity":  "warning",
			"message":   "Example alert",
			"timestamp": time.Now(),
		},
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

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.corsEnabled {
			next.ServeHTTP(w, r)
			return
		}

		// Set CORS headers based on configuration
		if len(s.corsOrigins) == 0 || s.corsOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			origin := r.Header.Get("Origin")
			for _, allowed := range s.corsOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleAIInsights returns AI-generated cluster insights
func (s *Server) handleAIInsights(w http.ResponseWriter, r *http.Request) {
	insights, err := s.engine.GetAIInsights()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get AI insights: %v", err), http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, insights)
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

// writeError writes an error response with the given status code and message
func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"status":  statusCode,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
	}
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
				// Try to ping the connection
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
			return
		}
	}
}

// BroadcastToClients sends data to all connected WebSocket clients
func (s *Server) BroadcastToClients(data interface{}) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	if len(s.clients) == 0 {
		return
	}

	var deadConnections []*websocket.Conn

	for conn := range s.clients {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(data); err != nil {
			klog.V(3).Infof("Failed to send to WebSocket client: %v", err)
			deadConnections = append(deadConnections, conn)
		}
	}

	// Clean up dead connections (but don't modify map during read lock)
	if len(deadConnections) > 0 {
		go func() {
			for _, conn := range deadConnections {
				s.removeClient(conn)
				_ = conn.Close()
			}
		}()
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

	// Get the new client
	client, err := s.contextManager.GetCurrentClient()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Update engine with new client
	// Note: This requires adding a method to update the engine's client
	// For now, we'll just return success

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
