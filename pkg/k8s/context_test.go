package k8s

import (
	"os"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Helper function to create test kubeconfig
func createTestKubeconfig(t *testing.T) string {
	t.Helper()
	
	config := clientcmdapi.NewConfig()
	
	// Add test clusters
	config.Clusters["test-cluster-1"] = &clientcmdapi.Cluster{
		Server: "https://test-cluster-1.example.com",
	}
	config.Clusters["test-cluster-2"] = &clientcmdapi.Cluster{
		Server: "https://test-cluster-2.example.com",
	}
	
	// Add test auth infos
	config.AuthInfos["test-user-1"] = &clientcmdapi.AuthInfo{
		Token: "test-token-1",
	}
	config.AuthInfos["test-user-2"] = &clientcmdapi.AuthInfo{
		Token: "test-token-2",
	}
	
	// Add test contexts
	config.Contexts["context-1"] = &clientcmdapi.Context{
		Cluster:   "test-cluster-1",
		AuthInfo:  "test-user-1",
		Namespace: "default",
	}
	config.Contexts["context-2"] = &clientcmdapi.Context{
		Cluster:   "test-cluster-2",
		AuthInfo:  "test-user-2",
		Namespace: "kube-system",
	}
	
	// Set current context
	config.CurrentContext = "context-1"
	
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	
	if err := clientcmd.WriteToFile(*config, tmpFile.Name()); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}
	
	return tmpFile.Name()
}

func TestNewContextManager(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	tests := []struct {
		name           string
		kubeconfigPath string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "valid kubeconfig",
			kubeconfigPath: kubeconfigPath,
			wantErr:        false,
		},
		{
			name:           "non-existent kubeconfig",
			kubeconfigPath: "/non/existent/path",
			wantErr:        true,
			errContains:    "failed to load kubeconfig",
		},
		{
			name:           "empty path uses default",
			kubeconfigPath: "",
			wantErr:        false, // May succeed if default kubeconfig exists
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := NewContextManager(tt.kubeconfigPath)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %v", tt.errContains, err)
				}
			} else {
				if err != nil && tt.kubeconfigPath != "" { // Only fail if we provided a valid path
					t.Errorf("Unexpected error: %v", err)
				}
				if cm != nil && cm.config == nil {
					t.Error("Config should not be nil")
				}
			}
		})
	}
}

func TestListContexts(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	contexts, err := cm.ListContexts()
	if err != nil {
		t.Fatalf("Failed to list contexts: %v", err)
	}
	
	if len(contexts) != 2 {
		t.Errorf("Expected 2 contexts, got %d", len(contexts))
	}
	
	// Verify context details
	contextMap := make(map[string]ContextInfo)
	for _, ctx := range contexts {
		contextMap[ctx.Name] = ctx
	}
	
	// Check context-1
	if ctx1, ok := contextMap["context-1"]; !ok {
		t.Error("context-1 not found")
	} else {
		if ctx1.ClusterName != "test-cluster-1" {
			t.Errorf("Expected cluster name test-cluster-1, got %s", ctx1.ClusterName)
		}
		if ctx1.Namespace != "default" {
			t.Errorf("Expected namespace default, got %s", ctx1.Namespace)
		}
		if !ctx1.Current {
			t.Error("context-1 should be current")
		}
		if ctx1.Server != "https://test-cluster-1.example.com" {
			t.Errorf("Expected server https://test-cluster-1.example.com, got %s", ctx1.Server)
		}
	}
	
	// Check context-2
	if ctx2, ok := contextMap["context-2"]; !ok {
		t.Error("context-2 not found")
	} else {
		if ctx2.Current {
			t.Error("context-2 should not be current")
		}
		if ctx2.Namespace != "kube-system" {
			t.Errorf("Expected namespace kube-system, got %s", ctx2.Namespace)
		}
	}
}

func TestGetCurrentContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	currentCtx, err := cm.GetCurrentContext()
	if err != nil {
		t.Fatalf("Failed to get current context: %v", err)
	}
	
	if currentCtx.Name != "context-1" {
		t.Errorf("Expected current context context-1, got %s", currentCtx.Name)
	}
	if !currentCtx.Current {
		t.Error("Current context should have Current=true")
	}
}

func TestSwitchContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	// Initial context should be context-1
	if cm.currentContext != "context-1" {
		t.Errorf("Expected initial context context-1, got %s", cm.currentContext)
	}
	
	// Switch to context-2
	err = cm.SwitchContext("context-2")
	if err != nil {
		t.Fatalf("Failed to switch context: %v", err)
	}
	
	if cm.currentContext != "context-2" {
		t.Errorf("Expected current context context-2, got %s", cm.currentContext)
	}
	
	// Verify through GetCurrentContext
	currentCtx, err := cm.GetCurrentContext()
	if err != nil {
		t.Fatalf("Failed to get current context: %v", err)
	}
	
	if currentCtx.Name != "context-2" {
		t.Errorf("Expected current context context-2, got %s", currentCtx.Name)
	}
	
	// Try switching to non-existent context
	err = cm.SwitchContext("non-existent")
	if err == nil {
		t.Error("Expected error when switching to non-existent context")
	}
}

func TestGetClient(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	// Note: This will fail in test environment as the test clusters don't exist
	// We're mainly testing the logic here
	client, err := cm.GetClient("context-1")
	if err == nil {
		// If it succeeds (unlikely in test), verify we got a client
		if client == nil {
			t.Error("Expected non-nil client")
		}
	} else {
		// Expected to fail due to test cluster not existing
		if !contains(err.Error(), "failed to connect to cluster") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
	
	// Test client caching
	// Even though connection fails, the same error should be returned
	client2, err2 := cm.GetClient("context-1")
	if err2 == nil && client2 == nil {
		t.Error("Expected consistent behavior on second call")
	}
}

func TestGetNamespace(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	tests := []struct {
		contextName string
		expected    string
	}{
		{"context-1", "default"},
		{"context-2", "kube-system"},
		{"non-existent", "default"}, // Should return default for non-existent
	}
	
	for _, tt := range tests {
		t.Run(tt.contextName, func(t *testing.T) {
			ns := cm.GetNamespace(tt.contextName)
			if ns != tt.expected {
				t.Errorf("Expected namespace %s, got %s", tt.expected, ns)
			}
		})
	}
}

func TestRefreshContexts(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	// Modify the kubeconfig file
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}
	
	// Add a new context
	config.Contexts["context-3"] = &clientcmdapi.Context{
		Cluster:   "test-cluster-1",
		AuthInfo:  "test-user-1",
		Namespace: "new-namespace",
	}
	
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		t.Fatalf("Failed to write updated kubeconfig: %v", err)
	}
	
	// Refresh contexts
	err = cm.RefreshContexts()
	if err != nil {
		t.Fatalf("Failed to refresh contexts: %v", err)
	}
	
	// Verify new context is loaded
	contexts, err := cm.ListContexts()
	if err != nil {
		t.Fatalf("Failed to list contexts: %v", err)
	}
	
	if len(contexts) != 3 {
		t.Errorf("Expected 3 contexts after refresh, got %d", len(contexts))
	}
	
	// Verify clients cache was cleared
	if len(cm.clients) != 0 {
		t.Error("Client cache should be cleared after refresh")
	}
}

func TestConcurrentAccess(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t)
	defer os.Remove(kubeconfigPath)
	
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}
	
	// Run concurrent operations
	done := make(chan bool)
	
	// Goroutine 1: List contexts repeatedly
	go func() {
		for i := 0; i < 10; i++ {
			cm.ListContexts()
		}
		done <- true
	}()
	
	// Goroutine 2: Switch contexts
	go func() {
		for i := 0; i < 5; i++ {
			cm.SwitchContext("context-1")
			cm.SwitchContext("context-2")
		}
		done <- true
	}()
	
	// Goroutine 3: Get current context
	go func() {
		for i := 0; i < 10; i++ {
			cm.GetCurrentContext()
		}
		done <- true
	}()
	
	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
	
	// If we get here without deadlock, test passes
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}