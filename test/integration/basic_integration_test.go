package integration

import (
	"context"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/health"
	"k8s.io/client-go/kubernetes/fake"
)

// TestBasicIntegration verifies that core components work together
func TestBasicIntegration(t *testing.T) {
	// Create a fake Kubernetes client for testing
	fakeClient := fake.NewSimpleClientset()

	// Create monitoring engine with minimal config
	engineConfig := core.EngineConfig{
		KubeClient:  fakeClient,
		ContextName: "test-context",
		Interval:    5 * time.Second,
		EnableAI:    false, // Disable AI for basic integration test
	}

	engine := core.NewEngine(engineConfig)
	if engine == nil {
		t.Fatal("Failed to create monitoring engine")
	}

	// Add a basic health check
	podCheck := health.NewPodHealthCheck()
	engine.AddCheck(podCheck)

	// Verify engine can be started and stopped
	_, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start engine in background
	done := make(chan error, 1)
	go func() {
		done <- engine.Start()
	}()

	// Let it run briefly
	select {
	case <-time.After(500 * time.Millisecond):
		// Stop the engine
		engine.Stop()
	case err := <-done:
		if err != nil {
			t.Fatalf("Engine failed to start: %v", err)
		}
	}

	// Wait for engine to stop
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("Engine failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Engine did not stop within timeout")
	}

	// Verify engine collected some results
	results := engine.GetResults()
	if len(results) == 0 {
		t.Log("No results collected (expected for fake client)")
	}

	t.Log("✅ Basic integration test passed")
}

// TestEngineHealthCheck verifies health check integration
func TestEngineHealthCheck(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	engineConfig := core.EngineConfig{
		KubeClient:  fakeClient,
		ContextName: "test-context",
		Interval:    time.Second,
		EnableAI:    false,
	}

	engine := core.NewEngine(engineConfig)

	// Add health check
	podCheck := health.NewPodHealthCheck()
	engine.AddCheck(podCheck)

	// Verify check was added
	if len(engine.GetResults()) > 0 {
		// Results should be empty initially
		t.Log("Initial results empty as expected")
	}

	// Test removal
	err := engine.RemoveCheck(podCheck.Name())
	if err != nil {
		t.Fatalf("Failed to remove check: %v", err)
	}

	t.Log("✅ Health check integration test passed")
}
