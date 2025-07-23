package health

import (
	"context"
	"fmt"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServiceHealthCheck checks the health of services
type ServiceHealthCheck struct {
	namespace string
	interval  time.Duration
}

// NewServiceHealthCheck creates a new service health check
func NewServiceHealthCheck() *ServiceHealthCheck {
	return &ServiceHealthCheck{
		namespace: "",
		interval:  30 * time.Second,
	}
}

// Name returns the name of the health check
func (s *ServiceHealthCheck) Name() string {
	return "service-health"
}

// Description returns a description of the health check
func (s *ServiceHealthCheck) Description() string {
	return "Monitors service endpoints, selectors, and connectivity"
}

// Check performs the service health check
func (s *ServiceHealthCheck) Check(ctx context.Context, client kubernetes.Interface) (core.CheckResult, error) {
	result := core.CheckResult{
		Name:      s.Name(),
		Timestamp: time.Now(),
		Status:    core.HealthStatusHealthy,
		Details:   make(map[string]interface{}),
		Metrics:   []core.Metric{},
	}

	// Get namespaces to check
	namespaces, err := s.getNamespacesToCheck(ctx, client)
	if err != nil {
		return result, fmt.Errorf("failed to get namespaces: %w", err)
	}

	var totalServices, healthyServices, unhealthyServices int
	var serviceIssues []string

	for _, ns := range namespaces {
		services, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return result, fmt.Errorf("failed to list services in namespace %s: %w", ns, err)
		}

		for _, service := range services.Items {
			totalServices++

			if s.isServiceHealthy(ctx, client, &service) {
				healthyServices++
			} else {
				unhealthyServices++
				serviceIssues = append(serviceIssues,
					fmt.Sprintf("%s/%s: No endpoints", service.Namespace, service.Name))
			}
		}
	}

	// Calculate health status
	if unhealthyServices > 0 {
		result.Status = core.HealthStatusDegraded
		result.Message = fmt.Sprintf("%d of %d services have issues", unhealthyServices, totalServices)
	} else {
		result.Message = fmt.Sprintf("All %d services are healthy", totalServices)
	}

	// Add details
	result.Details["total_services"] = totalServices
	result.Details["healthy_services"] = healthyServices
	result.Details["unhealthy_services"] = unhealthyServices
	if len(serviceIssues) > 0 {
		result.Details["issues"] = serviceIssues
	}

	// Add metrics
	result.Metrics = append(result.Metrics,
		core.Metric{
			Name:      "service_total",
			Value:     float64(totalServices),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
		core.Metric{
			Name:      "service_healthy",
			Value:     float64(healthyServices),
			Type:      core.MetricTypeGauge,
			Timestamp: time.Now(),
		},
	)

	result.Confidence = 1.0
	return result, nil
}

// Configure sets up the health check with configuration
func (s *ServiceHealthCheck) Configure(config map[string]interface{}) error {
	if v, ok := config["namespace"].(string); ok {
		s.namespace = v
	}
	return nil
}

// Interval returns how often this check should run
func (s *ServiceHealthCheck) Interval() time.Duration {
	return s.interval
}

// Criticality returns the importance level of this check
func (s *ServiceHealthCheck) Criticality() core.Criticality {
	return core.CriticalityMedium
}

// isServiceHealthy checks if a service is healthy
func (s *ServiceHealthCheck) isServiceHealthy(ctx context.Context, client kubernetes.Interface, service *corev1.Service) bool {
	// Check if service has endpoints
	endpoints, err := client.CoreV1().Endpoints(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
	if err != nil {
		return false
	}

	// Check if endpoints have ready addresses
	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 {
			return true
		}
	}

	return false
}

// getNamespacesToCheck returns the list of namespaces to check
func (s *ServiceHealthCheck) getNamespacesToCheck(ctx context.Context, client kubernetes.Interface) ([]string, error) {
	if s.namespace != "" {
		return []string{s.namespace}, nil
	}

	// Get all namespaces except system ones
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []string
	for _, ns := range namespaces.Items {
		// Skip system namespaces for services
		if ns.Name != "kube-system" && ns.Name != "kube-public" && ns.Name != "kube-node-lease" {
			result = append(result, ns.Name)
		}
	}

	return result, nil
}
