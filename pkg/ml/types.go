package ml

import "time"

// Metric represents a metric collected during health check (local copy to avoid cycle)
type Metric struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Prediction represents a predicted future state
type Prediction struct {
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	Probability float64   `json:"probability"`
	Reason      string    `json:"reason"`
}
