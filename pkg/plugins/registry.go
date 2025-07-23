package plugins

import (
	"fmt"
	"sync"

	"github.com/kubepulse/kubepulse/pkg/core"
)

// Registry manages all registered health check plugins
type Registry struct {
	checks map[string]core.HealthCheck
	mu     sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		checks: make(map[string]core.HealthCheck),
	}
}

// Register adds a new health check to the registry
func (r *Registry) Register(check core.HealthCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := check.Name()
	if _, exists := r.checks[name]; exists {
		return fmt.Errorf("health check %s already registered", name)
	}

	r.checks[name] = check
	return nil
}

// Unregister removes a health check from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.checks[name]; !exists {
		return fmt.Errorf("health check %s not found", name)
	}

	delete(r.checks, name)
	return nil
}

// Get retrieves a health check by name
func (r *Registry) Get(name string) (core.HealthCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	check, exists := r.checks[name]
	if !exists {
		return nil, fmt.Errorf("health check %s not found", name)
	}

	return check, nil
}

// List returns all registered health checks
func (r *Registry) List() []core.HealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	checks := make([]core.HealthCheck, 0, len(r.checks))
	for _, check := range r.checks {
		checks = append(checks, check)
	}

	return checks
}

// ListByName returns the names of all registered health checks
func (r *Registry) ListByName() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.checks))
	for name := range r.checks {
		names = append(names, name)
	}

	return names
}

// Count returns the number of registered health checks
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.checks)
}

// Clear removes all health checks from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checks = make(map[string]core.HealthCheck)
}
