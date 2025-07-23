package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures     int           // Number of failures before opening
	Timeout         time.Duration // How long to wait before trying again
	ResetTimeout    time.Duration // How long to stay in half-open state
	OnStateChange   func(from, to CircuitState)
}

// CircuitBreaker implements the circuit breaker pattern for AI calls
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitState
	failures     int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}

	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker is open, request rejected")
	}

	// Execute the function
	err := fn(ctx)
	
	// Handle the result
	cb.handleResult(err)
	
	return err
}

// canExecute determines if the request can be executed
func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailTime) > cb.config.Timeout {
			cb.setState(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// handleResult processes the result of the function execution
func (cb *CircuitBreaker) handleResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()
		
		if cb.state == CircuitHalfOpen {
			// Failed in half-open state, go back to open
			cb.setState(CircuitOpen)
		} else if cb.failures >= cb.config.MaxFailures {
			// Too many failures, open the circuit
			cb.setState(CircuitOpen)
		}
	} else {
		// Success - reset failures
		if cb.state == CircuitHalfOpen {
			// Success in half-open state, close the circuit
			cb.setState(CircuitClosed)
		}
		cb.failures = 0
	}
}

// setState changes the circuit breaker state and notifies if callback is set
func (cb *CircuitBreaker) setState(newState CircuitState) {
	oldState := cb.state
	cb.state = newState
	
	klog.V(2).Infof("Circuit breaker state changed from %s to %s (failures: %d)", 
		oldState, newState, cb.failures)
	
	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(oldState, newState)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failures
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	oldState := cb.state
	cb.state = CircuitClosed
	cb.failures = 0
	
	klog.V(2).Infof("Circuit breaker manually reset from %s to closed", oldState)
	
	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(oldState, CircuitClosed)
	}
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return map[string]interface{}{
		"state":         cb.state.String(),
		"failures":      cb.failures,
		"max_failures":  cb.config.MaxFailures,
		"last_failure":  cb.lastFailTime,
		"timeout":       cb.config.Timeout.String(),
	}
}