package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestQueryRequestStruct tests the QueryRequest struct
func TestQueryRequestStruct(t *testing.T) {
	req := QueryRequest{Query: "test query"}
	if req.Query != "test query" {
		t.Errorf("expected query %q, got %q", "test query", req.Query)
	}
}

// TestRemediationRequestStruct tests the RemediationRequest struct
func TestRemediationRequestStruct(t *testing.T) {
	req := RemediationRequest{
		ActionID: "action-123",
		DryRun:   true,
	}
	
	if req.ActionID != "action-123" {
		t.Errorf("expected ActionID %q, got %q", "action-123", req.ActionID)
	}
	
	if !req.DryRun {
		t.Error("expected DryRun to be true")
	}
}

// TestAssistantQueryValidation tests request validation
func TestAssistantQueryValidation(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		body         interface{}
		expectedCode int
		expectedBody string
	}{
		{
			name:         "wrong method",
			method:       "GET",
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "Method not allowed",
		},
		{
			name:         "invalid JSON",
			method:       "POST",
			body:         "invalid json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid request body",
		},
		{
			name:         "empty query",
			method:       "POST",
			body:         QueryRequest{Query: ""},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Query cannot be empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple server without dependencies
			server := &Server{}
			
			var body bytes.Buffer
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body.WriteString(str)
				} else {
					_ = json.NewEncoder(&body).Encode(tt.body)
				}
			}
			
			req := httptest.NewRequest(tt.method, "/api/assistant/query", &body)
			w := httptest.NewRecorder()
			
			// Call handler - it will fail at engine call but that's expected
			server.HandleAssistantQuery(w, req)
			
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
			
			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestRemediationSuggestionsValidation tests path parameter validation
func TestRemediationSuggestionsValidation(t *testing.T) {
	server := &Server{}
	
	// Test empty check name
	req := httptest.NewRequest("GET", "/api/remediation//suggestions", nil)
	// Set empty check name in URL vars (simulating mux behavior)
	w := httptest.NewRecorder()
	
	server.HandleRemediationSuggestions(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
	
	if !strings.Contains(w.Body.String(), "Check name is required") {
		t.Errorf("expected error message about check name, got %q", w.Body.String())
	}
}

// TestExecuteRemediationValidation tests method and body validation
func TestExecuteRemediationValidation(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		body         interface{}
		expectedCode int
		expectedBody string
	}{
		{
			name:         "wrong method",
			method:       "GET",
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "Method not allowed",
		},
		{
			name:         "invalid JSON",
			method:       "POST",
			body:         "invalid json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid request body",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{}
			
			var body bytes.Buffer
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body.WriteString(str)
				} else {
					_ = json.NewEncoder(&body).Encode(tt.body)
				}
			}
			
			req := httptest.NewRequest(tt.method, "/api/remediation/execute", &body)
			w := httptest.NewRecorder()
			
			server.HandleExecuteRemediation(w, req)
			
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
			
			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestJSONStructures tests that our request/response structures work with JSON
func TestJSONStructures(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "QueryRequest",
			data: QueryRequest{Query: "test query"},
		},
		{
			name: "RemediationRequest",
			data: RemediationRequest{ActionID: "action-1", DryRun: true},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("failed to marshal %s: %v", tt.name, err)
			}
			
			// Test JSON unmarshaling
			switch tt.name {
			case "QueryRequest":
				var req QueryRequest
				if err := json.Unmarshal(jsonData, &req); err != nil {
					t.Fatalf("failed to unmarshal %s: %v", tt.name, err)
				}
				original := tt.data.(QueryRequest)
				if req.Query != original.Query {
					t.Errorf("unmarshaled query %q != original %q", req.Query, original.Query)
				}
			case "RemediationRequest":
				var req RemediationRequest
				if err := json.Unmarshal(jsonData, &req); err != nil {
					t.Fatalf("failed to unmarshal %s: %v", tt.name, err)
				}
				original := tt.data.(RemediationRequest)
				if req.ActionID != original.ActionID || req.DryRun != original.DryRun {
					t.Errorf("unmarshaled request doesn't match original")
				}
			}
		})
	}
}

// TestHTTPMethodValidation tests that handlers properly validate HTTP methods
func TestHTTPMethodValidation(t *testing.T) {
	server := &Server{}
	
	// Test that GET request to POST-only endpoints fail
	postEndpoints := []struct {
		name    string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"assistant query", "/api/assistant/query", server.HandleAssistantQuery},
		{"execute remediation", "/api/remediation/execute", server.HandleExecuteRemediation},
	}
	
	for _, endpoint := range postEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint.path, nil)
			w := httptest.NewRecorder()
			
			endpoint.handler(w, req)
			
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for GET to %s, got %d", 
					http.StatusMethodNotAllowed, endpoint.path, w.Code)
			}
		})
	}
}

// TestHandlerExistence tests that handlers exist and don't crash
func TestHandlerExistence(t *testing.T) {
	// We can't test successful responses without mocking the engine properly
	// But we can verify handlers exist and handle nil engine gracefully
	
	server := &Server{}
	
	handlers := []struct {
		name    string
		method  string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"predictive insights", "GET", "/api/insights/predictive", server.HandlePredictiveInsights},
		{"smart alerts", "GET", "/api/alerts/smart", server.HandleSmartAlerts},
	}
	
	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			req := httptest.NewRequest(h.method, h.path, nil)
			w := httptest.NewRecorder()
			
			// Should not panic - may get error due to nil engine but should handle it
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Handler %s panicked: %v", h.name, r)
				}
			}()
			
			h.handler(w, req)
			
			// Should not be 404 (handler not found)
			if w.Code == http.StatusNotFound {
				t.Errorf("Handler %s returned 404, handler may not exist", h.name)
			}
		})
	}
}