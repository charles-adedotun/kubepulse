package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestContextInfo(t *testing.T) {
	info := ContextInfo{
		Name:        "test-context",
		ClusterName: "test-cluster",
		Namespace:   "test-namespace",
		Server:      "https://test-server.com",
		User:        "test-user",
		Current:     true,
	}

	if info.Name != "test-context" {
		t.Errorf("expected name 'test-context', got %s", info.Name)
	}

	if info.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got %s", info.ClusterName)
	}

	if info.Namespace != "test-namespace" {
		t.Errorf("expected namespace 'test-namespace', got %s", info.Namespace)
	}

	if info.Server != "https://test-server.com" {
		t.Errorf("expected server 'https://test-server.com', got %s", info.Server)
	}

	if info.User != "test-user" {
		t.Errorf("expected user 'test-user', got %s", info.User)
	}

	if !info.Current {
		t.Error("expected Current to be true")
	}
}

func TestNewContextManager_InvalidPath(t *testing.T) {
	_, err := NewContextManager("/non/existent/path")
	if err == nil {
		t.Error("expected error for non-existent kubeconfig path")
	}
}

func TestNewContextManager_EmptyPath(t *testing.T) {
	// This should use the default path, which might not exist in test environment
	// We'll expect an error since we're in a test environment without a real kubeconfig
	_, err := NewContextManager("")
	if err == nil {
		t.Log("Warning: NewContextManager with empty path succeeded (real kubeconfig found)")
	}
}

func TestContextManager_WithMockConfig(t *testing.T) {
	// Create a temporary kubeconfig file for testing
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")

	// Create a minimal valid kubeconfig
	config := &clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": {
				Server: "https://test-server.com",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test-user": {
				Token: "test-token",
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster:   "test-cluster",
				AuthInfo:  "test-user",
				Namespace: "test-namespace",
			},
			"other-context": {
				Cluster:   "test-cluster",
				AuthInfo:  "test-user",
				Namespace: "default",
			},
		},
		CurrentContext: "test-context",
	}

	// Write the config to file
	err := writeKubeConfig(kubeconfigPath, config)
	if err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	// Test creating context manager
	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// Test ListContexts
	contexts, err := cm.ListContexts()
	if err != nil {
		t.Fatalf("failed to list contexts: %v", err)
	}

	if len(contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(contexts))
	}

	// Find the test-context
	var testContext *ContextInfo
	for _, ctx := range contexts {
		if ctx.Name == "test-context" {
			testContext = &ctx
			break
		}
	}

	if testContext == nil {
		t.Fatal("test-context not found")
	}

	if testContext.ClusterName != "test-cluster" {
		t.Errorf("expected cluster name 'test-cluster', got %s", testContext.ClusterName)
	}

	if testContext.Namespace != "test-namespace" {
		t.Errorf("expected namespace 'test-namespace', got %s", testContext.Namespace)
	}

	if testContext.Server != "https://test-server.com" {
		t.Errorf("expected server 'https://test-server.com', got %s", testContext.Server)
	}

	if testContext.User != "test-user" {
		t.Errorf("expected user 'test-user', got %s", testContext.User)
	}

	if !testContext.Current {
		t.Error("expected test-context to be current")
	}

	// Test GetCurrentContext
	currentCtx, err := cm.GetCurrentContext()
	if err != nil {
		t.Fatalf("failed to get current context: %v", err)
	}

	if currentCtx.Name != "test-context" {
		t.Errorf("expected current context 'test-context', got %s", currentCtx.Name)
	}

	// Test GetNamespace
	namespace := cm.GetNamespace("test-context")
	if namespace != "test-namespace" {
		t.Errorf("expected namespace 'test-namespace', got %s", namespace)
	}

	namespace = cm.GetNamespace("other-context")
	if namespace != "default" {
		t.Errorf("expected namespace 'default', got %s", namespace)
	}

	namespace = cm.GetNamespace("non-existent")
	if namespace != "default" {
		t.Errorf("expected default namespace 'default' for non-existent context, got %s", namespace)
	}
}

func TestContextManager_GetCurrentContext_NoContext(t *testing.T) {
	// Create a temporary kubeconfig file without current context
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")

	config := &clientcmdapi.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		Clusters:       map[string]*clientcmdapi.Cluster{},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
		Contexts:       map[string]*clientcmdapi.Context{},
		CurrentContext: "", // No current context
	}

	err := writeKubeConfig(kubeconfigPath, config)
	if err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	_, err = cm.GetCurrentContext()
	if err == nil {
		t.Error("expected error when no current context is set")
	}
}

func TestContextManager_RefreshContexts(t *testing.T) {
	// Create initial kubeconfig
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")

	config := &clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": {Server: "https://test-server.com"},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test-user": {Token: "test-token"},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster:  "test-cluster",
				AuthInfo: "test-user",
			},
		},
		CurrentContext: "test-context",
	}

	err := writeKubeConfig(kubeconfigPath, config)
	if err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// Modify the kubeconfig file
	config.Contexts["new-context"] = &clientcmdapi.Context{
		Cluster:  "test-cluster",
		AuthInfo: "test-user",
	}

	err = writeKubeConfig(kubeconfigPath, config)
	if err != nil {
		t.Fatalf("failed to write updated kubeconfig: %v", err)
	}

	// Refresh contexts
	err = cm.RefreshContexts()
	if err != nil {
		t.Fatalf("failed to refresh contexts: %v", err)
	}

	// Check that new context is available
	contexts, err := cm.ListContexts()
	if err != nil {
		t.Fatalf("failed to list contexts: %v", err)
	}

	found := false
	for _, ctx := range contexts {
		if ctx.Name == "new-context" {
			found = true
			break
		}
	}

	if !found {
		t.Error("new-context not found after refresh")
	}
}

func TestContextManager_DefaultNamespace(t *testing.T) {
	// Test context with empty namespace should default to "default"
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")

	config := &clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": {Server: "https://test-server.com"},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test-user": {Token: "test-token"},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster:   "test-cluster",
				AuthInfo:  "test-user",
				Namespace: "", // Empty namespace
			},
		},
		CurrentContext: "test-context",
	}

	err := writeKubeConfig(kubeconfigPath, config)
	if err != nil {
		t.Fatalf("failed to write test kubeconfig: %v", err)
	}

	cm, err := NewContextManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	contexts, err := cm.ListContexts()
	if err != nil {
		t.Fatalf("failed to list contexts: %v", err)
	}

	if len(contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(contexts))
	}

	if contexts[0].Namespace != "default" {
		t.Errorf("expected namespace 'default', got %s", contexts[0].Namespace)
	}
}

// writeKubeConfig writes a kubeconfig to a file in YAML format
func writeKubeConfig(path string, config *clientcmdapi.Config) error {
	// Create a simple YAML representation of the config
	content := fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: %s
clusters:
`, config.CurrentContext)

	for name, cluster := range config.Clusters {
		content += fmt.Sprintf(`- cluster:
    server: %s
  name: %s
`, cluster.Server, name)
	}

	content += "contexts:\n"
	for name, ctx := range config.Contexts {
		content += fmt.Sprintf(`- context:
    cluster: %s
    user: %s
`, ctx.Cluster, ctx.AuthInfo)
		if ctx.Namespace != "" {
			content += fmt.Sprintf("    namespace: %s\n", ctx.Namespace)
		}
		content += fmt.Sprintf("  name: %s\n", name)
	}

	content += "users:\n"
	for name, user := range config.AuthInfos {
		content += fmt.Sprintf(`- name: %s
  user:
    token: %s
`, name, user.Token)
	}

	return os.WriteFile(path, []byte(content), 0644)
}
