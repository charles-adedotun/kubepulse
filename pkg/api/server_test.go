package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kubepulse/kubepulse/internal/config"
	"github.com/kubepulse/kubepulse/pkg/k8s"
	"k8s.io/client-go/kubernetes"
)

// Mock context manager for testing
type mockContextManager struct {
	contexts       []k8s.ContextInfo
	currentContext string
	switchError    error
	listError      error
}

func (m *mockContextManager) ListContexts() ([]k8s.ContextInfo, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.contexts, nil
}

func (m *mockContextManager) GetCurrentContext() (k8s.ContextInfo, error) {
	for _, ctx := range m.contexts {
		if ctx.Name == m.currentContext {
			return ctx, nil
		}
	}
	return k8s.ContextInfo{}, nil
}

func (m *mockContextManager) SwitchContext(contextName string) error {
	if m.switchError != nil {
		return m.switchError
	}
	m.currentContext = contextName
	return nil
}

func (m *mockContextManager) GetClient(contextName string) (kubernetes.Interface, error) {
	return nil, nil
}

func (m *mockContextManager) GetCurrentClient() (kubernetes.Interface, error) {
	return nil, nil
}

func (m *mockContextManager) RefreshContexts() error {
	return nil
}

func (m *mockContextManager) GetNamespace(contextName string) string {
	return "default"
}

// Helper to create test server
func createTestServer(contextManager k8s.ContextManagerInterface) *Server {
	router := mux.NewRouter()
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		router:         router,
		contextManager: contextManager,
		clients:        make(map[*websocket.Conn]bool),
		uiConfig:       config.UIConfig{},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		shutdown: make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Register routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/contexts", server.handleListContexts).Methods("GET")
	api.HandleFunc("/contexts/current", server.handleGetCurrentContext).Methods("GET")
	api.HandleFunc("/contexts/switch", server.handleSwitchContext).Methods("POST")

	return server
}

func TestHandleListContexts(t *testing.T) {
	tests := []struct {
		name           string
		contextManager *mockContextManager
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful list",
			contextManager: &mockContextManager{
				contexts: []k8s.ContextInfo{
					{Name: "context-1", ClusterName: "cluster-1", Namespace: "default", Current: true},
					{Name: "context-2", ClusterName: "cluster-2", Namespace: "kube-system", Current: false},
				},
				currentContext: "context-1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "empty contexts",
			contextManager: &mockContextManager{
				contexts: []k8s.ContextInfo{},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error listing contexts",
			contextManager: &mockContextManager{
				listError: fmt.Errorf("failed to list contexts"),
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to list contexts",
		},
		{
			name:           "nil context manager",
			contextManager: nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Context manager not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *Server
			if tt.contextManager != nil {
				server = createTestServer(tt.contextManager)
			} else {
				server = &Server{
					router: mux.NewRouter(),
				}
			}

			req := httptest.NewRequest("GET", "/api/v1/contexts", nil)
			w := httptest.NewRecorder()

			server.handleListContexts(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" {
				var response map[string]string
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if !strings.Contains(response["error"], tt.expectedBody) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedBody, response["error"])
				}
			} else if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if _, ok := response["contexts"]; !ok {
					t.Error("Response should contain 'contexts' field")
				}
			}
		})
	}
}

func TestHandleGetCurrentContext(t *testing.T) {
	tests := []struct {
		name           string
		contextManager *mockContextManager
		expectedStatus int
		expectedName   string
	}{
		{
			name: "successful get current",
			contextManager: &mockContextManager{
				contexts: []k8s.ContextInfo{
					{Name: "context-1", ClusterName: "cluster-1", Current: true},
				},
				currentContext: "context-1",
			},
			expectedStatus: http.StatusOK,
			expectedName:   "context-1",
		},
		{
			name:           "nil context manager",
			contextManager: nil,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *Server
			if tt.contextManager != nil {
				server = createTestServer(tt.contextManager)
			} else {
				server = &Server{
					router: mux.NewRouter(),
				}
			}

			req := httptest.NewRequest("GET", "/api/v1/contexts/current", nil)
			w := httptest.NewRecorder()

			server.handleGetCurrentContext(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response k8s.ContextInfo
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Name != tt.expectedName {
					t.Errorf("Expected context name %s, got %s", tt.expectedName, response.Name)
				}
			}
		})
	}
}

func TestHandleSwitchContext(t *testing.T) {
	tests := []struct {
		name           string
		contextManager *mockContextManager
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful switch",
			contextManager: &mockContextManager{
				contexts: []k8s.ContextInfo{
					{Name: "context-1", ClusterName: "cluster-1"},
					{Name: "context-2", ClusterName: "cluster-2"},
				},
				currentContext: "context-1",
			},
			requestBody: map[string]string{
				"context_name": "context-2",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing context name",
			contextManager: &mockContextManager{
				contexts: []k8s.ContextInfo{
					{Name: "context-1", ClusterName: "cluster-1"},
				},
			},
			requestBody:    map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "context_name is required",
		},
		{
			name:           "invalid request body",
			contextManager: &mockContextManager{},
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name: "switch error",
			contextManager: &mockContextManager{
				switchError: fmt.Errorf("context not found"),
			},
			requestBody: map[string]string{
				"context_name": "non-existent",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "context not found",
		},
		{
			name:           "nil context manager",
			contextManager: nil,
			requestBody:    map[string]string{"context_name": "test"},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Context manager not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *Server
			if tt.contextManager != nil {
				server = createTestServer(tt.contextManager)
			} else {
				server = &Server{
					router:  mux.NewRouter(),
					clients: make(map[*websocket.Conn]bool),
				}
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/contexts/switch", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.handleSwitchContext(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response map[string]string
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if !strings.Contains(response["error"], tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, response["error"])
				}
			} else if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response["success"] != true {
					t.Error("Response should have success=true")
				}
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			message:    "Invalid input",
		},
		{
			name:       "internal error",
			statusCode: http.StatusInternalServerError,
			message:    "Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			server.writeError(w, tt.statusCode, tt.message)

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("Expected error message %q, got %q", tt.message, response["error"])
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "simple map",
			data: map[string]string{"key": "value"},
		},
		{
			name: "struct",
			data: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{Name: "test", Value: 42},
		},
		{
			name: "array",
			data: []string{"one", "two", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			server.writeJSON(w, tt.data)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Verify we can decode the response
			var decoded interface{}
			if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
				t.Errorf("Failed to decode JSON response: %v", err)
			}
		})
	}
}
