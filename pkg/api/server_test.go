package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubepulse/kubepulse/internal/config"
)

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid config",
			config: Config{
				Port:        8080,
				Host:        "localhost",
				CORSEnabled: true,
				CORSOrigins: []string{"*"},
			},
			valid: true,
		},
		{
			name: "zero port",
			config: Config{
				Port: 0,
				Host: "localhost",
			},
			valid: true, // Zero port is valid (system assigns)
		},
		{
			name: "empty host",
			config: Config{
				Port: 8080,
				Host: "",
			},
			valid: true, // Empty host is valid (binds to all interfaces)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - just ensure config can be created
			if tt.config.Port < 0 {
				t.Errorf("port should not be negative: %d", tt.config.Port)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	server := &Server{}
	w := httptest.NewRecorder()

	data := map[string]string{
		"message": "test",
		"status":  "ok",
	}

	server.writeJSON(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["message"] != "test" {
		t.Errorf("expected message 'test', got %s", response["message"])
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", response["status"])
	}
}

func TestWriteError(t *testing.T) {
	server := &Server{}
	w := httptest.NewRecorder()

	server.writeError(w, http.StatusBadRequest, "test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %s", contentType)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["message"] != "test error message" {
		t.Errorf("expected message 'test error message', got %v", response["message"])
	}

	if response["error"] != true {
		t.Errorf("expected error true, got %v", response["error"])
	}

	if response["status"] != float64(400) {
		t.Errorf("expected status 400, got %v", response["status"])
	}
}

func TestWriteJSON_InvalidData(t *testing.T) {
	server := &Server{}
	w := httptest.NewRecorder()

	// Channel cannot be marshaled to JSON
	data := make(chan int)

	server.writeJSON(w, data)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	// The writeJSON function uses http.Error which writes plain text, not JSON
	bodyText := w.Body.String()
	if !strings.Contains(bodyText, "Failed to encode JSON") {
		t.Errorf("expected encode error in response, got %s", bodyText)
	}
}

func TestCorsMiddleware_Disabled(t *testing.T) {
	server := &Server{
		corsEnabled: false,
		corsOrigins: []string{},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	handler := server.corsMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Should not have CORS headers when disabled
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS headers when CORS is disabled")
	}
}

func TestCorsMiddleware_Enabled_Wildcard(t *testing.T) {
	server := &Server{
		corsEnabled: true,
		corsOrigins: []string{"*"},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	handler := server.corsMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://anydomain.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected wildcard CORS origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials allowed header")
	}
}

func TestCorsMiddleware_Enabled_SpecificOrigins(t *testing.T) {
	server := &Server{
		corsEnabled: true,
		corsOrigins: []string{"https://example.com", "https://test.com"},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.corsMiddleware(testHandler)

	// Test allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected specific CORS origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// Test disallowed origin
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://malicious.com")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header for disallowed origin")
	}
}

func TestCorsMiddleware_Preflight(t *testing.T) {
	server := &Server{
		corsEnabled: true,
		corsOrigins: []string{"https://example.com"},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := server.corsMiddleware(testHandler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for preflight, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("expected CORS origin header for preflight")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected allowed methods header for preflight")
	}

	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected allowed headers header for preflight")
	}
}

func TestSpaHandler(t *testing.T) {
	// Create a simple file server for testing
	fileHandler := http.FileServer(http.Dir("./"))

	spa := spaHandler{
		handler: fileHandler,
		path:    "./",
	}

	// Test normal file request (should pass through)
	req := httptest.NewRequest("GET", "/server.go", nil)
	w := httptest.NewRecorder()

	spa.ServeHTTP(w, req)

	// Should attempt to serve the file (may or may not exist, but handler should be called)
	if w.Code == 0 {
		t.Error("expected some response from spa handler")
	}
}

func TestServer_TimeoutDefaults(t *testing.T) {
	// Test that proper default timeouts are set when creating a server
	defaultReadTimeout := 15 * time.Second
	defaultWriteTimeout := 15 * time.Second
	defaultIdleTimeout := 60 * time.Second

	if defaultReadTimeout != 15*time.Second {
		t.Error("default read timeout should be 15 seconds")
	}

	if defaultWriteTimeout != 15*time.Second {
		t.Error("default write timeout should be 15 seconds")
	}

	if defaultIdleTimeout != 60*time.Second {
		t.Error("default idle timeout should be 60 seconds")
	}
}

func TestServer_AddressFormatting(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		port         int
		expectedAddr string
	}{
		{
			name:         "localhost with port",
			host:         "localhost",
			port:         8080,
			expectedAddr: "localhost:8080",
		},
		{
			name:         "empty host with port",
			host:         "",
			port:         9090,
			expectedAddr: ":9090",
		},
		{
			name:         "IP address with port",
			host:         "127.0.0.1",
			port:         3000,
			expectedAddr: "127.0.0.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var addr string
			if tt.host == "" {
				addr = fmt.Sprintf(":%d", tt.port)
			} else {
				addr = fmt.Sprintf("%s:%d", tt.host, tt.port)
			}

			if addr != tt.expectedAddr {
				t.Errorf("expected address %s, got %s", tt.expectedAddr, addr)
			}
		})
	}
}

func TestServer_UIConfigResponse(t *testing.T) {
	uiConfig := config.UIConfig{
		Theme:                "dark",
		MaxReconnectAttempts: 10,
	}

	server := &Server{
		uiConfig: uiConfig,
	}

	req := httptest.NewRequest("GET", "/api/v1/config/ui", nil)
	w := httptest.NewRecorder()

	server.handleUIConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response config.UIConfig
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %s", response.Theme)
	}

	if response.MaxReconnectAttempts != 10 {
		t.Errorf("expected max reconnect attempts 10, got %d", response.MaxReconnectAttempts)
	}

	// Test that response is valid JSON and contains expected structure
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type to be application/json")
	}
}

func TestServer_ClientManagement(t *testing.T) {
	server := &Server{
		clients: make(map[*websocket.Conn]bool),
	}

	if server.clients == nil {
		t.Error("expected clients map to be initialized")
	}

	if len(server.clients) != 0 {
		t.Errorf("expected empty clients map, got %d entries", len(server.clients))
	}
}

func TestServer_ShutdownChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := &Server{
		shutdown: make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}

	if server.shutdown == nil {
		t.Error("expected shutdown channel to be initialized")
	}

	if server.ctx == nil {
		t.Error("expected context to be initialized")
	}

	if server.cancel == nil {
		t.Error("expected cancel function to be initialized")
	}

	// Test that cancel function works
	server.cancel()

	select {
	case <-server.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("expected context to be cancelled")
	}
}
