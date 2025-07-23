package core

import (
	"fmt"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryHealth       ErrorCategory = "health_check"
	ErrorCategoryAI           ErrorCategory = "ai_analysis"
	ErrorCategoryAPI          ErrorCategory = "api"
	ErrorCategoryKubernetes   ErrorCategory = "kubernetes"
	ErrorCategoryConfiguration ErrorCategory = "configuration"
	ErrorCategorySystem       ErrorCategory = "system"
)

// EngineError represents a structured error with context
type EngineError struct {
	ID          string                 `json:"id"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Category    ErrorCategory          `json:"category"`
	Severity    ErrorSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	Cause       error                  `json:"-"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Recoverable bool                   `json:"recoverable"`
	RetryCount  int                    `json:"retry_count,omitempty"`
}

// Error implements the error interface
func (e *EngineError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s.%s: %s (caused by: %v)", 
			e.Severity, e.Component, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s.%s: %s", 
		e.Severity, e.Component, e.Operation, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *EngineError) Unwrap() error {
	return e.Cause
}

// NewEngineError creates a new structured error
func NewEngineError(component, operation string, category ErrorCategory, severity ErrorSeverity, message string, cause error) *EngineError {
	return &EngineError{
		ID:          generateErrorID(),
		Component:   component,
		Operation:   operation,
		Category:    category,
		Severity:    severity,
		Message:     message,
		Cause:       cause,
		Context:     make(map[string]interface{}),
		Timestamp:   time.Now(),
		Recoverable: severity != ErrorSeverityCritical,
	}
}

// WithContext adds context to the error
func (e *EngineError) WithContext(key string, value interface{}) *EngineError {
	e.Context[key] = value
	return e
}

// WithRetryCount sets the retry count
func (e *EngineError) WithRetryCount(count int) *EngineError {
	e.RetryCount = count
	return e
}

// IsRecoverable returns whether the error is recoverable
func (e *EngineError) IsRecoverable() bool {
	return e.Recoverable && e.Severity != ErrorSeverityCritical
}

// IsCritical returns whether the error is critical
func (e *EngineError) IsCritical() bool {
	return e.Severity == ErrorSeverityCritical
}

// ErrorHandler manages error handling and recovery
type ErrorHandler struct {
	errors        []EngineError
	maxErrors     int
	errorCallback func(EngineError)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(maxErrors int, callback func(EngineError)) *ErrorHandler {
	return &ErrorHandler{
		errors:        make([]EngineError, 0),
		maxErrors:     maxErrors,
		errorCallback: callback,
	}
}

// Handle processes an error and determines recovery strategy
func (h *ErrorHandler) Handle(err *EngineError) error {
	// Add to error history
	h.addError(*err)
	
	// Execute callback if provided
	if h.errorCallback != nil {
		h.errorCallback(*err)
	}
	
	// Determine if we should continue or fail
	if err.IsCritical() {
		return fmt.Errorf("critical error encountered: %w", err)
	}
	
	// For recoverable errors, log and continue
	return nil
}

// GetRecentErrors returns recent errors
func (h *ErrorHandler) GetRecentErrors(limit int) []EngineError {
	if limit > len(h.errors) {
		limit = len(h.errors)
	}
	
	start := len(h.errors) - limit
	if start < 0 {
		start = 0
	}
	
	return h.errors[start:]
}

// GetErrorsByCategory returns errors filtered by category
func (h *ErrorHandler) GetErrorsByCategory(category ErrorCategory) []EngineError {
	var filtered []EngineError
	for _, err := range h.errors {
		if err.Category == category {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// addError adds an error to the history with size management
func (h *ErrorHandler) addError(err EngineError) {
	h.errors = append(h.errors, err)
	
	// Maintain maximum size
	if len(h.errors) > h.maxErrors {
		h.errors = h.errors[1:]
	}
}

// generateErrorID creates a unique error identifier
func generateErrorID() string {
	return fmt.Sprintf("err-%d", time.Now().UnixNano())
}

// Helper functions for creating common errors

// NewHealthCheckError creates a health check error
func NewHealthCheckError(checkName, operation, message string, cause error) *EngineError {
	return NewEngineError(
		checkName,
		operation,
		ErrorCategoryHealth,
		ErrorSeverityMedium,
		message,
		cause,
	)
}

// NewAIError creates an AI analysis error
func NewAIError(operation, message string, cause error) *EngineError {
	return NewEngineError(
		"ai",
		operation,
		ErrorCategoryAI,
		ErrorSeverityMedium,
		message,
		cause,
	)
}

// NewKubernetesError creates a Kubernetes API error
func NewKubernetesError(operation, message string, cause error) *EngineError {
	return NewEngineError(
		"kubernetes",
		operation,
		ErrorCategoryKubernetes,
		ErrorSeverityHigh,
		message,
		cause,
	)
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(component, message string, cause error) *EngineError {
	return NewEngineError(
		component,
		"configure",
		ErrorCategoryConfiguration,
		ErrorSeverityCritical,
		message,
		cause,
	)
}