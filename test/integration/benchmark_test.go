package integration

import (
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/core"
	"github.com/kubepulse/kubepulse/pkg/health"
	"k8s.io/client-go/kubernetes/fake"
)

// BenchmarkEngineCreation benchmarks engine creation performance
func BenchmarkEngineCreation(b *testing.B) {
	fakeClient := fake.NewSimpleClientset()

	engineConfig := core.EngineConfig{
		KubeClient:  fakeClient,
		ContextName: "benchmark-context",
		Interval:    time.Second,
		EnableAI:    false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := core.NewEngine(engineConfig)
		if engine == nil {
			b.Fatal("Failed to create engine")
		}
	}
}

// BenchmarkHealthCheckExecution benchmarks health check execution
func BenchmarkHealthCheckExecution(b *testing.B) {
	fakeClient := fake.NewSimpleClientset()

	engineConfig := core.EngineConfig{
		KubeClient:  fakeClient,
		ContextName: "benchmark-context",
		Interval:    time.Second,
		EnableAI:    false,
	}

	engine := core.NewEngine(engineConfig)
	podCheck := health.NewPodHealthCheck()
	engine.AddCheck(podCheck)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := engine.GetResults()
		_ = results // Use the results to prevent optimization
	}
}

// BenchmarkMemoryAllocation benchmarks memory usage patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	fakeClient := fake.NewSimpleClientset()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engineConfig := core.EngineConfig{
			KubeClient:  fakeClient,
			ContextName: "benchmark-context",
			Interval:    time.Second,
			EnableAI:    false,
		}

		engine := core.NewEngine(engineConfig)
		podCheck := health.NewPodHealthCheck()
		engine.AddCheck(podCheck)

		// Simulate some work
		_ = engine.GetResults()
	}
}
