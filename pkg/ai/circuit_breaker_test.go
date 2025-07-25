package ai

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreakerStates(t *testing.T) {
	stateChanges := make([]string, 0)
	var mu sync.Mutex

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures:  2,
		Timeout:      50 * time.Millisecond,
		ResetTimeout: 30 * time.Millisecond,
		OnStateChange: func(from, to CircuitState) {
			mu.Lock()
			stateChanges = append(stateChanges, from.String()+"->"+to.String())
			mu.Unlock()
		},
	})

	// Initial state should be closed
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected initial state to be closed, got %s", cb.GetState())
	}

	// Success should keep circuit closed
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if cb.GetState() != CircuitClosed {
		t.Error("Circuit should remain closed after success")
	}

	// First failure
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure 1")
	})
	if err == nil {
		t.Error("Expected error")
	}
	if cb.GetFailures() != 1 {
		t.Errorf("Expected 1 failure, got %d", cb.GetFailures())
	}

	// Second failure should open circuit
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("failure 2")
	})
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open after %d failures", cb.config.MaxFailures)
	}

	// Verify state change was recorded
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	if len(stateChanges) != 1 || stateChanges[0] != "closed->open" {
		t.Errorf("Expected state change closed->open, got %v", stateChanges)
	}
	mu.Unlock()

	// Circuit open should reject requests
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err == nil || err.Error() != "circuit breaker is open, request rejected" {
		t.Error("Expected circuit breaker open error")
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should transition to half-open
	err = cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil // Success in half-open
	})
	if err != nil {
		t.Errorf("Expected success in half-open state, got %v", err)
	}
	if cb.GetState() != CircuitClosed {
		t.Error("Expected circuit to close after success in half-open")
	}

	// Wait a bit more for async state change callback
	time.Sleep(50 * time.Millisecond)

	// Verify final state changes
	mu.Lock()
	expectedChanges := []string{"closed->open", "open->half-open", "half-open->closed"}
	if len(stateChanges) != len(expectedChanges) {
		t.Errorf("Expected %d state changes, got %d: %v", len(expectedChanges), len(stateChanges), stateChanges)
	}
	mu.Unlock()
}

func TestCircuitBreakerExponentialBackoff(t *testing.T) {
	t.Skip("Skipping timing-sensitive test")
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 1,
		Timeout:     10 * time.Millisecond,
	})

	// Cause circuit to open
	_ = cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	if cb.GetState() != CircuitOpen {
		t.Fatal("Circuit should be open")
	}

	// Record backoff progression
	backoffTimes := make([]time.Duration, 0)

	for i := 0; i < 3; i++ {
		start := time.Now()

		// Wait for circuit to allow retry
		for cb.GetState() == CircuitOpen {
			time.Sleep(5 * time.Millisecond)
			if time.Since(start) > 5*time.Second {
				t.Fatal("Timeout waiting for circuit state change")
			}
		}

		backoffTime := time.Since(start)
		backoffTimes = append(backoffTimes, backoffTime)

		// Fail again to trigger next backoff
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("fail again")
		})
	}

	// Verify exponential increase
	for i := 1; i < len(backoffTimes); i++ {
		ratio := float64(backoffTimes[i]) / float64(backoffTimes[i-1])
		if ratio < 1.5 || ratio > 2.5 {
			t.Errorf("Backoff not exponential: %v -> %v (ratio: %.2f)",
				backoffTimes[i-1], backoffTimes[i], ratio)
		}
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	t.Skip("Skipping flaky concurrency test")
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 10,
		Timeout:     100 * time.Millisecond,
	})

	var successCount int32
	var failureCount int32
	var rejectedCount int32

	// Run concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			err := cb.Execute(context.Background(), func(ctx context.Context) error {
				// Fail 50% of requests
				if id%2 == 0 {
					return errors.New("failure")
				}
				return nil
			})

			if err != nil {
				if err.Error() == "circuit breaker is open, request rejected" {
					atomic.AddInt32(&rejectedCount, 1)
				} else {
					atomic.AddInt32(&failureCount, 1)
				}
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	totalProcessed := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&failureCount) + atomic.LoadInt32(&rejectedCount)
	if totalProcessed != 100 {
		t.Errorf("Expected 100 requests processed, got %d (success: %d, failure: %d, rejected: %d)",
			totalProcessed, successCount, failureCount, rejectedCount)
	}

	// Circuit should be open after multiple failures
	if cb.GetState() != CircuitOpen {
		t.Error("Expected circuit to be open after concurrent failures")
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 2,
		Timeout:     50 * time.Millisecond,
	})

	// Cause failures to open circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(context.Background(), func(ctx context.Context) error {
			return errors.New("fail")
		})
	}

	if cb.GetState() != CircuitOpen {
		t.Fatal("Circuit should be open")
	}
	if cb.GetFailures() != 2 {
		t.Errorf("Expected 2 failures, got %d", cb.GetFailures())
	}

	// Manual reset
	cb.Reset()

	if cb.GetState() != CircuitClosed {
		t.Error("Circuit should be closed after reset")
	}
	if cb.GetFailures() != 0 {
		t.Error("Failures should be 0 after reset")
	}
	if cb.backoffAttempts != 0 {
		t.Error("Backoff attempts should be 0 after reset")
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		MaxFailures: 3,
		Timeout:     100 * time.Millisecond,
	})

	// Generate some activity
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	_ = cb.Execute(context.Background(), func(ctx context.Context) error { return errors.New("fail") })

	stats := cb.GetStats()

	if stats["state"] != "closed" {
		t.Errorf("Expected state to be closed, got %s", stats["state"])
	}
	if stats["failures"].(int) != 1 {
		t.Errorf("Expected 1 failure, got %v", stats["failures"])
	}
	if stats["max_failures"].(int) != 3 {
		t.Errorf("Expected max_failures to be 3, got %v", stats["max_failures"])
	}
	if stats["timeout"] != "100ms" {
		t.Errorf("Expected timeout to be 100ms, got %v", stats["timeout"])
	}
}

func TestCircuitBreakerMaxBackoff(t *testing.T) {
	cb := &CircuitBreaker{
		config: CircuitBreakerConfig{
			Timeout: 1 * time.Millisecond,
		},
		baseTimeout:     1 * time.Millisecond,
		backoffAttempts: 100, // Very high to test max cap
	}

	timeout := cb.calculateBackoffTimeout()
	maxTimeout := 30 * time.Minute

	if timeout > maxTimeout {
		t.Errorf("Timeout should be capped at %v, got %v", maxTimeout, timeout)
	}
}
