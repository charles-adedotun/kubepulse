package core

import (
	"context"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// MockHealthCheck implements HealthCheck interface for testing
type MockHealthCheck struct {
	name        string
	description string
	result      CheckResult
	err         error
	interval    time.Duration
	configured  bool
}

func (m *MockHealthCheck) Name() string        { return m.name }
func (m *MockHealthCheck) Description() string { return m.description }
func (m *MockHealthCheck) Check(ctx context.Context, client kubernetes.Interface) (CheckResult, error) {
	return m.result, m.err
}
func (m *MockHealthCheck) Configure(config map[string]interface{}) error {
	m.configured = true
	return nil
}
func (m *MockHealthCheck) Interval() time.Duration  { return m.interval }
func (m *MockHealthCheck) Criticality() Criticality { return CriticalityMedium }

func TestNewEngine(t *testing.T) {
	client := fake.NewSimpleClientset()
	config := EngineConfig{
		KubeClient:  client,
		ContextName: "test-context",
		Interval:    10 * time.Second,
	}

	engine := NewEngine(config)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}

	if engine.client != client {
		t.Error("Client not set correctly")
	}

	if engine.currentContext != "test-context" {
		t.Error("Context not set correctly")
	}

	if engine.interval != 10*time.Second {
		t.Error("Interval not set correctly")
	}

	if engine.results == nil {
		t.Error("Results map not initialized")
	}

	if engine.resultsTTL != 24*time.Hour {
		t.Error("Results TTL not set to default")
	}
}

func TestAddAndRemoveCheck(t *testing.T) {
	engine := &Engine{
		checks: make([]HealthCheck, 0),
	}

	check1 := &MockHealthCheck{name: "check1"}
	check2 := &MockHealthCheck{name: "check2"}

	// Test adding checks
	engine.AddCheck(check1)
	engine.AddCheck(check2)

	if len(engine.checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(engine.checks))
	}

	// Test removing check
	err := engine.RemoveCheck("check1")
	if err != nil {
		t.Errorf("Failed to remove check: %v", err)
	}

	if len(engine.checks) != 1 {
		t.Errorf("Expected 1 check after removal, got %d", len(engine.checks))
	}

	// Test removing non-existent check
	err = engine.RemoveCheck("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent check")
	}
}

func TestRunChecks(t *testing.T) {
	client := fake.NewSimpleClientset()

	// Add some test resources
	_, _ = client.CoreV1().Nodes().Create(context.TODO(), &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{Type: v1.NodeReady, Status: v1.ConditionTrue},
			},
		},
	}, metav1.CreateOptions{})

	// Import alerts package
	alertManager := alerts.NewManager()

	engine := &Engine{
		client:       client,
		checks:       make([]HealthCheck, 0),
		results:      make(map[string]CheckResult),
		alertChan:    make(chan Alert, 10),
		metricsChan:  make(chan Metric, 10),
		ctx:          context.Background(),
		resultsTTL:   24 * time.Hour,
		alertManager: alertManager,
	}

	// Add mock checks
	successCheck := &MockHealthCheck{
		name: "success-check",
		result: CheckResult{
			Name:      "success-check",
			Status:    HealthStatusHealthy,
			Message:   "All good",
			Timestamp: time.Now(),
		},
	}

	failureCheck := &MockHealthCheck{
		name: "failure-check",
		result: CheckResult{
			Name:      "failure-check",
			Status:    HealthStatusUnhealthy,
			Message:   "Something wrong",
			Timestamp: time.Now(),
		},
	}

	engine.AddCheck(successCheck)
	engine.AddCheck(failureCheck)

	// Run checks
	engine.runChecks()

	// Verify results
	time.Sleep(100 * time.Millisecond) // Give time for goroutines

	results := engine.GetResults()
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if result, ok := results["success-check"]; !ok || result.Status != HealthStatusHealthy {
		t.Error("Success check result not stored correctly")
	}

	if result, ok := results["failure-check"]; !ok || result.Status != HealthStatusUnhealthy {
		t.Error("Failure check result not stored correctly")
	}
}

func TestResultsTTLCleanup(t *testing.T) {
	engine := &Engine{
		results:    make(map[string]CheckResult),
		resultsTTL: 100 * time.Millisecond, // Short TTL for testing
		ctx:        context.Background(),
	}

	// Add old and new results
	oldResult := CheckResult{
		Name:      "old-check",
		Status:    HealthStatusHealthy,
		Timestamp: time.Now().Add(-200 * time.Millisecond),
	}

	newResult := CheckResult{
		Name:      "new-check",
		Status:    HealthStatusHealthy,
		Timestamp: time.Now(),
	}

	engine.storeResult(oldResult)
	engine.storeResult(newResult)

	// Verify both results exist
	if len(engine.GetResults()) != 2 {
		t.Error("Expected 2 results before cleanup")
	}

	// Trigger cleanup
	ticker := make(chan time.Time, 1)
	ticker <- time.Now()

	go engine.cleanupExpiredResults(ticker)
	time.Sleep(50 * time.Millisecond)

	// Verify old result was cleaned up
	results := engine.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 result after cleanup, got %d", len(results))
	}

	if _, ok := results["old-check"]; ok {
		t.Error("Old result should have been cleaned up")
	}

	if _, ok := results["new-check"]; !ok {
		t.Error("New result should still exist")
	}
}

func TestUpdateClient(t *testing.T) {
	engine := &Engine{
		client:         fake.NewSimpleClientset(),
		currentContext: "old-context",
		alertChan:      make(chan Alert, 1),
		ctx:            context.Background(),
	}

	newClient := fake.NewSimpleClientset()
	engine.UpdateClient(newClient, "new-context")

	if engine.client != newClient {
		t.Error("Client not updated")
	}

	if engine.currentContext != "new-context" {
		t.Error("Context not updated")
	}

	// Check that context switch alert was sent
	select {
	case alert := <-engine.alertChan:
		if alert.Severity != AlertSeverityInfo {
			t.Error("Expected info alert for context switch")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected alert for context switch")
	}
}

func TestCalculateClusterHealth(t *testing.T) {
	engine := &Engine{
		results: map[string]CheckResult{
			"check1": {
				Name:   "check1",
				Status: HealthStatusHealthy,
			},
			"check2": {
				Name:   "check2",
				Status: HealthStatusDegraded,
			},
			"check3": {
				Name:   "check3",
				Status: HealthStatusUnhealthy,
			},
		},
	}

	health := engine.GetClusterHealth("test-cluster")

	if health.Status != HealthStatusDegraded {
		t.Errorf("Expected degraded status, got %s", health.Status)
	}

	if health.Score.Weighted == 0 {
		t.Error("Expected non-zero weighted score")
	}

	if len(health.Checks) != 3 {
		t.Errorf("Expected 3 checks, got %d", len(health.Checks))
	}

	// Count status types
	statusCounts := map[HealthStatus]int{}
	for _, check := range health.Checks {
		statusCounts[check.Status]++
	}

	if statusCounts[HealthStatusHealthy] != 1 || statusCounts[HealthStatusDegraded] != 1 || statusCounts[HealthStatusUnhealthy] != 1 {
		t.Error("Incorrect status counts in health checks")
	}
}

func TestConcurrentResultAccess(t *testing.T) {
	engine := &Engine{
		results: make(map[string]CheckResult),
	}

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			result := CheckResult{
				Name:      checkName(id),
				Status:    HealthStatusHealthy,
				Timestamp: time.Now(),
			}
			engine.storeResult(result)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			engine.GetResults()
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all results
	results := engine.GetResults()
	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
}

func checkName(id int) string {
	return "check" + string(rune('0'+id))
}

func TestErrorPropagation(t *testing.T) {
	client := fake.NewSimpleClientset()

	// Make client return errors
	client.PrependReactor("list", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	engine := &Engine{
		client:      client,
		checks:      make([]HealthCheck, 0),
		results:     make(map[string]CheckResult),
		alertChan:   make(chan Alert, 10),
		metricsChan: make(chan Metric, 10),
		ctx:         context.Background(),
		resultsTTL:  24 * time.Hour,
	}

	errorCheck := &MockHealthCheck{
		name: "error-check",
		err:  context.DeadlineExceeded,
	}

	engine.AddCheck(errorCheck)
	engine.runChecks()

	time.Sleep(100 * time.Millisecond)

	result, ok := engine.GetResult("error-check")
	if !ok {
		t.Fatal("Error check result not found")
	}

	if result.Status != HealthStatusUnknown {
		t.Errorf("Expected unknown status for error, got %s", result.Status)
	}

	if result.Error == nil {
		t.Error("Expected error to be captured in result")
	}
}
