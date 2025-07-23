package slo

import "time"

// SLO represents a Service Level Objective (local copy to avoid cycle)
type SLO struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	SLI          string        `json:"sli"`
	Target       float64       `json:"target"`
	Window       time.Duration `json:"window"`
	BudgetPolicy []BudgetRule  `json:"budget_policy"`
}

// BudgetRule defines actions based on error budget consumption
type BudgetRule struct {
	Threshold float64 `json:"threshold"` // Percentage of budget consumed
	Action    string  `json:"action"`    // notify, alert, page
}

// SLOStatus represents the current status of an SLO
type SLOStatus struct {
	SLO           SLO     `json:"slo"`
	CurrentValue  float64 `json:"current_value"`
	ErrorBudget   float64 `json:"error_budget"`
	BurnRate      float64 `json:"burn_rate"`
	IsViolated    bool    `json:"is_violated"`
	TimeToExhaust string  `json:"time_to_exhaust,omitempty"`
}

// Metric represents a metric for SLO calculation (local copy to avoid cycle)
type Metric struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
