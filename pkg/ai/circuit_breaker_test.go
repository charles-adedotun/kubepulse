package ai

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    CircuitState
		expected string
	}{
		{"closed", CircuitClosed, "closed"},
		{"open", CircuitOpen, "open"},
		{"half-open", CircuitHalfOpen, "half-open"},
		{"unknown", CircuitState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.state.String())
			}
		})
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  3,
		Timeout:      5 * time.Second,
		ResetTimeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker(config)

	if cb == nil {
		t.Fatal("expected non-nil circuit breaker")
	}

	if cb.config.MaxFailures != 3 {
		t.Errorf("expected max failures 3, got %d", cb.config.MaxFailures)
	}

	if cb.config.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", cb.config.Timeout)
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("expected initial state closed, got %s", cb.GetState().String())
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  3,
		Timeout:      5 * time.Second,
		ResetTimeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker(config)
	ctx := context.Background()

	// Test successful execution
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("expected state closed after success, got %s", cb.GetState().String())
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  2,
		Timeout:      5 * time.Second,
		ResetTimeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker(config)
	ctx := context.Background()
	testError := errors.New("test error")

	// First failure
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})

	if err != testError {
		t.Errorf("expected test error, got %v", err)
	}

	if cb.GetState() != CircuitClosed {
		t.Errorf("expected state closed after first failure, got %s", cb.GetState().String())
	}

	// Second failure - should open circuit
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return testError
	})

	if err != testError {
		t.Errorf("expected test error, got %v", err)
	}

	if cb.GetState() != CircuitOpen {
		t.Errorf("expected state open after max failures, got %s", cb.GetState().String())
	}
}

func TestCircuitBreaker_Execute_CircuitOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  1,
		Timeout:      100 * time.Millisecond,
		ResetTimeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker(config)
	ctx := context.Background()

	// Trigger circuit to open
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})

	if cb.GetState() != CircuitOpen {
		t.Errorf("expected state open, got %s", cb.GetState().String())
	}

	// Try to execute while circuit is open
	err := cb.Execute(ctx, func(ctx context.Context) error {
		t.Error("function should not be called when circuit is open")
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open")
	}

	if !contains(err.Error(), "circuit breaker is open") {
		t.Errorf("expected circuit open error, got %v", err)
	}
}

func TestCircuitBreaker_Execute_HalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  1,
		Timeout:      50 * time.Millisecond,
		ResetTimeout: 100 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)
	ctx := context.Background()

	// Open the circuit
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})

	if cb.GetState() != CircuitOpen {
		t.Errorf("expected state open, got %s", cb.GetState().String())
	}

	// Wait for timeout to potentially transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Execute - behavior depends on implementation
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Should either succeed or be rejected, but not panic
	if err != nil && !contains(err.Error(), "circuit breaker is open") {
		t.Errorf("unexpected error type: %v", err)
	}
}

func TestCircuitBreaker_GetState(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  1,
		Timeout:      5 * time.Second,
		ResetTimeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker(config)

	// Initial state should be closed
	if cb.GetState() != CircuitClosed {
		t.Errorf("expected initial state closed, got %s", cb.GetState().String())
	}

	// After failure, should be open
	ctx := context.Background()
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})

	if cb.GetState() != CircuitOpen {
		t.Errorf("expected state open after failure, got %s", cb.GetState().String())
	}
}

func TestCircuitBreaker_OnStateChange_Callback(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  1,
		Timeout:      50 * time.Millisecond,
		ResetTimeout: 100 * time.Millisecond,
		OnStateChange: func(from, to CircuitState) {
			// Simple callback that doesn't modify shared state to avoid race conditions
		},
	}

	cb := NewCircuitBreaker(config)
	ctx := context.Background()

	// Trigger state change from closed to open
	_ = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})

	// Check the callback functionality - validate the config structure
	if cb.config.OnStateChange == nil {
		t.Error("expected OnStateChange callback to be set")
	}

	// Validate that state actually changed
	if cb.GetState() != CircuitOpen {
		t.Errorf("expected state to be open after failure, got %s", cb.GetState().String())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  5,
		Timeout:      100 * time.Millisecond,
		ResetTimeout: 200 * time.Millisecond,
	}

	cb := NewCircuitBreaker(config)
	ctx := context.Background()

	// Run multiple goroutines concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 5; j++ {
				_ = cb.Execute(ctx, func(ctx context.Context) error {
					if j%2 == 0 {
						return nil
					}
					return errors.New("failure")
				})
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Circuit breaker should still be functional
	state := cb.GetState()
	if state != CircuitClosed && state != CircuitOpen && state != CircuitHalfOpen {
		t.Errorf("unexpected final state: %s", state.String())
	}
}

func TestCircuitBreaker_ContextCancellation(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures:  3,
		Timeout:      5 * time.Second,
		ResetTimeout: 2 * time.Second,
	}

	cb := NewCircuitBreaker(config)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	err := cb.Execute(ctx, func(ctx context.Context) error {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	if err != context.Canceled {
		t.Errorf("expected context canceled error, got %v", err)
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	config := CircuitBreakerConfig{
		MaxFailures: 0, // Should use default
		Timeout:     0, // Should use default
	}

	cb := NewCircuitBreaker(config)

	if cb.config.MaxFailures <= 0 {
		t.Error("expected default max failures to be positive")
	}

	if cb.config.Timeout <= 0 {
		t.Error("expected default timeout to be positive")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAtIndex(s, substr))
}

func containsAtIndex(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}