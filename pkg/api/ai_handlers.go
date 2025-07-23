package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
)

// QueryRequest represents a natural language query
type QueryRequest struct {
	Query string `json:"query"`
}

// RemediationRequest represents a remediation execution request
type RemediationRequest struct {
	ActionID string `json:"action_id"`
	DryRun   bool   `json:"dry_run"`
}

// HandleAssistantQuery handles natural language queries
func (s *Server) HandleAssistantQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query cannot be empty", http.StatusBadRequest)
		return
	}

	klog.V(2).Infof("Processing assistant query: %s", req.Query)

	response, err := s.engine.QueryAssistant(req.Query)
	if err != nil {
		klog.Errorf("Assistant query failed: %v", err)
		http.Error(w, "Query processing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}

// HandlePredictiveInsights returns AI predictions
func (s *Server) HandlePredictiveInsights(w http.ResponseWriter, r *http.Request) {
	insights, err := s.engine.GetPredictiveInsights()
	if err != nil {
		klog.Errorf("Failed to get predictive insights: %v", err)
		http.Error(w, "Failed to generate predictions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"predictions":  insights,
		"generated_at": time.Now(),
	}); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}

// HandleRemediationSuggestions returns remediation suggestions for a check
func (s *Server) HandleRemediationSuggestions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkName := vars["check"]

	if checkName == "" {
		http.Error(w, "Check name is required", http.StatusBadRequest)
		return
	}

	suggestions, err := s.engine.GetRemediationSuggestions(checkName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		klog.Errorf("Failed to get remediation suggestions: %v", err)
		http.Error(w, "Failed to generate suggestions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"check":       checkName,
		"suggestions": suggestions,
	}); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}

// HandleExecuteRemediation executes a remediation action
func (s *Server) HandleExecuteRemediation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RemediationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	record, err := s.engine.ExecuteRemediation(req.ActionID, req.DryRun)
	if err != nil {
		klog.Errorf("Remediation execution failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(record); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}

// HandleSmartAlerts returns intelligent alert insights
func (s *Server) HandleSmartAlerts(w http.ResponseWriter, r *http.Request) {
	insights, err := s.engine.GetSmartAlertInsights()
	if err != nil {
		klog.Errorf("Failed to get smart alert insights: %v", err)
		http.Error(w, "Failed to get alert insights", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(insights); err != nil {
		klog.Errorf("Failed to encode response: %v", err)
	}
}
