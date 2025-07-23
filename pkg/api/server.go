package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kubepulse/kubepulse/pkg/core"
	"k8s.io/klog/v2"
)

// Server handles HTTP API requests
type Server struct {
	engine   *core.Engine
	router   *mux.Router
	server   *http.Server
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
}

// spaHandler implements a single-page application handler
type spaHandler struct {
	handler http.Handler
	path    string
}

// Config holds server configuration
type Config struct {
	Port   int
	Engine *core.Engine
}

// NewServer creates a new API server
func NewServer(config Config) *Server {
	router := mux.NewRouter()
	
	server := &Server{
		engine: config.Engine,
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", config.Port),
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients: make(map[*websocket.Conn]bool),
	}
	
	server.setupRoutes()
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
	clusterName := r.URL.Query().Get("cluster")
	if clusterName == "" {
		clusterName = "default"
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
			"id":       "example-1",
			"severity": "warning",
			"message":  "Example alert",
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
			
			fmt.Fprintf(w, "# TYPE %s %s\n", metric.Name, string(metric.Type))
			fmt.Fprintf(w, "%s%s %f %d\n", 
				metric.Name, 
				labels, 
				metric.Value, 
				metric.Timestamp.Unix()*1000)
		}
	}
}

// handleWebSocket handles WebSocket connections for real-time updates
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		klog.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	s.clients[conn] = true
	klog.Info("New WebSocket client connected")
	
	// Send initial data
	health := s.engine.GetClusterHealth("default")
	if err := conn.WriteJSON(health); err != nil {
		klog.Errorf("Failed to send initial data: %v", err)
		delete(s.clients, conn)
		return
	}
	
	// Keep connection alive and handle disconnect
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			klog.Info("WebSocket client disconnected")
			delete(s.clients, conn)
			break
		}
	}
}

// BroadcastUpdate sends updates to all connected WebSocket clients
func (s *Server) BroadcastUpdate(data interface{}) {
	for client := range s.clients {
		if err := client.WriteJSON(data); err != nil {
			klog.Errorf("Failed to broadcast to client: %v", err)
			client.Close()
			delete(s.clients, client)
		}
	}
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
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

// writeJSON writes JSON response
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
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