package core

import (
	"errors"
	"strings"
	"testing"
)

func TestEngineError_Error(t *testing.T) {
	baseErr := errors.New("connection failed")
	
	tests := []struct {
		name     string
		err      *EngineError
		contains []string
	}{
		{
			name: "error with cause",
			err: &EngineError{
				Component: "api",
				Operation: "connect",
				Severity:  ErrorSeverityHigh,
				Message:   "failed to connect",
				Cause:     baseErr,
			},
			contains: []string{"high", "api.connect", "failed to connect", "connection failed"},
		},
		{
			name: "error without cause",
			err: &EngineError{
				Component: "health",
				Operation: "check",
				Severity:  ErrorSeverityMedium,
				Message:   "check timeout",
			},
			contains: []string{"medium", "health.check", "check timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected error string to contain '%s', got '%s'", expected, result)
				}
			}
		})
	}
}

func TestEngineError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := &EngineError{Cause: baseErr}
	
	if err.Unwrap() != baseErr {
		t.Errorf("expected unwrapped error to be base error")
	}
}

func TestNewEngineError(t *testing.T) {
	baseErr := errors.New("base error")
	
	err := NewEngineError(
		"test-component",
		"test-operation", 
		ErrorCategoryHealth,
		ErrorSeverityHigh,
		"test message",
		baseErr,
	)

	if err.Component != "test-component" {
		t.Errorf("expected component 'test-component', got %s", err.Component)
	}
	
	if err.Operation != "test-operation" {
		t.Errorf("expected operation 'test-operation', got %s", err.Operation)
	}
	
	if err.Category != ErrorCategoryHealth {
		t.Errorf("expected category %s, got %s", ErrorCategoryHealth, err.Category)
	}
	
	if err.Severity != ErrorSeverityHigh {
		t.Errorf("expected severity %s, got %s", ErrorSeverityHigh, err.Severity)
	}
	
	if err.Message != "test message" {
		t.Errorf("expected message 'test message', got %s", err.Message)
	}
	
	if err.Cause != baseErr {
		t.Errorf("expected cause to be base error")
	}
	
	if err.ID == "" {
		t.Error("expected non-empty error ID")
	}
	
	if err.Recoverable != true {
		t.Error("expected non-critical error to be recoverable")
	}
}

func TestEngineError_IsRecoverable(t *testing.T) {
	tests := []struct {
		name       string
		severity   ErrorSeverity
		recoverable bool
		expected   bool
	}{
		{"critical error", ErrorSeverityCritical, true, false},
		{"high error", ErrorSeverityHigh, true, true},
		{"medium error", ErrorSeverityMedium, true, true},
		{"low error", ErrorSeverityLow, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &EngineError{
				Severity:    tt.severity,
				Recoverable: tt.recoverable,
			}
			
			if err.IsRecoverable() != tt.expected {
				t.Errorf("expected IsRecoverable() to be %v, got %v", tt.expected, err.IsRecoverable())
			}
		})
	}
}

func TestEngineError_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected bool
	}{
		{"critical", ErrorSeverityCritical, true},
		{"high", ErrorSeverityHigh, false},
		{"medium", ErrorSeverityMedium, false},
		{"low", ErrorSeverityLow, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &EngineError{Severity: tt.severity}
			if err.IsCritical() != tt.expected {
				t.Errorf("expected IsCritical() to be %v, got %v", tt.expected, err.IsCritical())
			}
		})
	}
}

func TestErrorHandler(t *testing.T) {
	var callbackErr *EngineError
	handler := NewErrorHandler(5, func(err EngineError) {
		callbackErr = &err
	})

	// Test handling non-critical error
	err := NewEngineError("test", "op", ErrorCategoryHealth, ErrorSeverityMedium, "test", nil)
	result := handler.Handle(err)
	
	if result != nil {
		t.Errorf("expected nil for non-critical error, got %v", result)
	}
	
	if callbackErr == nil {
		t.Error("expected callback to be called")
	}

	// Test handling critical error
	criticalErr := NewEngineError("test", "op", ErrorCategoryHealth, ErrorSeverityCritical, "critical", nil)
	result = handler.Handle(criticalErr)
	
	if result == nil {
		t.Error("expected error for critical error")
	}
}

func TestNewHealthCheckError(t *testing.T) {
	baseErr := errors.New("check failed")
	err := NewHealthCheckError("pod-check", "validate", "Pod validation failed", baseErr)
	
	if err.Component != "pod-check" {
		t.Errorf("expected component 'pod-check', got %s", err.Component)
	}
	
	if err.Category != ErrorCategoryHealth {
		t.Errorf("expected category %s, got %s", ErrorCategoryHealth, err.Category)
	}
	
	if err.Severity != ErrorSeverityMedium {
		t.Errorf("expected severity %s, got %s", ErrorSeverityMedium, err.Severity)
	}
}