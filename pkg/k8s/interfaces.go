package k8s

import "k8s.io/client-go/kubernetes"

// ContextManagerInterface defines the interface for context management
type ContextManagerInterface interface {
	ListContexts() ([]ContextInfo, error)
	GetCurrentContext() (ContextInfo, error)
	SwitchContext(contextName string) error
	GetClient(contextName string) (kubernetes.Interface, error)
	GetCurrentClient() (kubernetes.Interface, error)
	RefreshContexts() error
	GetNamespace(contextName string) string
}
