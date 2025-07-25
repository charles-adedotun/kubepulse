package k8s

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
)

// ContextInfo represents information about a Kubernetes context
type ContextInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	Namespace   string `json:"namespace"`
	Server      string `json:"server"`
	User        string `json:"user"`
	Current     bool   `json:"current"`
}

// ContextManager manages multiple Kubernetes contexts
type ContextManager struct {
	kubeconfigPath string
	config         *clientcmdapi.Config
	clients        map[string]kubernetes.Interface
	currentContext string
	mu             sync.RWMutex
}

// NewContextManager creates a new context manager
func NewContextManager(kubeconfigPath string) (*ContextManager, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	}

	// Load kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	cm := &ContextManager{
		kubeconfigPath: kubeconfigPath,
		config:         config,
		clients:        make(map[string]kubernetes.Interface),
		currentContext: config.CurrentContext,
	}

	// Initialize client for current context
	if cm.currentContext != "" {
		if _, err := cm.GetClient(cm.currentContext); err != nil {
			klog.Warningf("Failed to initialize client for current context %s: %v", cm.currentContext, err)
		}
	}

	return cm, nil
}

// ListContexts returns all available contexts
func (cm *ContextManager) ListContexts() ([]ContextInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var contexts []ContextInfo
	for name, context := range cm.config.Contexts {
		cluster := cm.config.Clusters[context.Cluster]
		// authInfo := cm.config.AuthInfos[context.AuthInfo] // Could be used for user info in the future

		namespace := context.Namespace
		if namespace == "" {
			namespace = "default"
		}

		info := ContextInfo{
			Name:        name,
			ClusterName: context.Cluster,
			Namespace:   namespace,
			User:        context.AuthInfo,
			Current:     name == cm.currentContext,
		}

		if cluster != nil {
			info.Server = cluster.Server
		}

		// authInfo could be used to add user info if needed in the future

		contexts = append(contexts, info)
	}

	return contexts, nil
}

// GetCurrentContext returns the current context info
func (cm *ContextManager) GetCurrentContext() (ContextInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.currentContext == "" {
		return ContextInfo{}, fmt.Errorf("no current context set")
	}

	contexts, err := cm.ListContexts()
	if err != nil {
		return ContextInfo{}, err
	}

	for _, ctx := range contexts {
		if ctx.Name == cm.currentContext {
			return ctx, nil
		}
	}

	return ContextInfo{}, fmt.Errorf("current context %s not found", cm.currentContext)
}

// SwitchContext switches to a different context
func (cm *ContextManager) SwitchContext(contextName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate context exists
	if _, exists := cm.config.Contexts[contextName]; !exists {
		return fmt.Errorf("context %s not found", contextName)
	}

	// Try to get or create client for the context
	// We allow switching even if connection fails
	if _, err := cm.getOrCreateClient(contextName); err != nil {
		klog.Warningf("Failed to create client for context %s: %v", contextName, err)
		// Still switch the context even if client creation fails
	}

	cm.currentContext = contextName
	klog.Infof("Switched to context: %s", contextName)
	return nil
}

// GetClient returns a Kubernetes client for the specified context
func (cm *ContextManager) GetClient(contextName string) (kubernetes.Interface, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.getOrCreateClient(contextName)
}

// GetCurrentClient returns the client for the current context
func (cm *ContextManager) GetCurrentClient() (kubernetes.Interface, error) {
	cm.mu.RLock()
	contextName := cm.currentContext
	cm.mu.RUnlock()

	if contextName == "" {
		return nil, fmt.Errorf("no current context set")
	}

	return cm.GetClient(contextName)
}

// getOrCreateClient gets or creates a client for the context (must be called with lock held)
func (cm *ContextManager) getOrCreateClient(contextName string) (kubernetes.Interface, error) {
	// Check if client already exists
	if client, exists := cm.clients[contextName]; exists {
		return client, nil
	}

	// Create new client
	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cm.kubeconfigPath},
		configOverrides,
	)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config: %w", err)
	}

	// Set timeout for the client
	restConfig.Timeout = 10 * time.Second

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Test the connection
	_, err = client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Cache the client
	cm.clients[contextName] = client
	return client, nil
}

// RefreshContexts reloads the kubeconfig and refreshes context information
func (cm *ContextManager) RefreshContexts() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	config, err := clientcmd.LoadFromFile(cm.kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to reload kubeconfig: %w", err)
	}

	cm.config = config

	// Clear cached clients to force recreation
	cm.clients = make(map[string]kubernetes.Interface)

	// If current context is no longer valid, clear it
	if _, exists := config.Contexts[cm.currentContext]; !exists {
		cm.currentContext = config.CurrentContext
	}

	return nil
}

// GetNamespace returns the namespace for the specified context
func (cm *ContextManager) GetNamespace(contextName string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if context, exists := cm.config.Contexts[contextName]; exists {
		if context.Namespace != "" {
			return context.Namespace
		}
	}
	return "default"
}
