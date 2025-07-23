package ml

import (
	"context"
	"math"
	"time"
)

// AnomalyDetector implements basic anomaly detection using statistical methods
type AnomalyDetector struct {
	baselines map[string]*Baseline
	window    time.Duration
	threshold float64
}

// Baseline represents learned normal behavior
type Baseline struct {
	Mean   float64
	StdDev float64
	Count  int
	Window []float64
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		baselines: make(map[string]*Baseline),
		window:    24 * time.Hour,
		threshold: 2.0, // 2 standard deviations
	}
}

// DetectAnomalies analyzes metrics for anomalies
func (a *AnomalyDetector) DetectAnomalies(ctx context.Context, metrics []Metric) []Prediction {
	predictions := make([]Prediction, 0)

	for _, metric := range metrics {
		if a.isAnomalous(metric) {
			prediction := Prediction{
				Timestamp:   time.Now().Add(time.Hour),
				Status:      "degraded",
				Probability: a.calculateProbability(metric),
				Reason:      "Statistical anomaly detected",
			}
			predictions = append(predictions, prediction)
		}
	}

	return predictions
}

// isAnomalous checks if a metric value is anomalous
func (a *AnomalyDetector) isAnomalous(metric Metric) bool {
	baseline := a.getOrCreateBaseline(metric.Name)

	if baseline.Count < 10 {
		// Not enough data for anomaly detection
		a.updateBaseline(baseline, metric.Value)
		return false
	}

	// Calculate z-score
	zscore := math.Abs(metric.Value-baseline.Mean) / baseline.StdDev
	isAnomalous := zscore > a.threshold

	// Update baseline with new value
	a.updateBaseline(baseline, metric.Value)

	return isAnomalous
}

// getOrCreateBaseline gets or creates a baseline for a metric
func (a *AnomalyDetector) getOrCreateBaseline(metricName string) *Baseline {
	if baseline, exists := a.baselines[metricName]; exists {
		return baseline
	}

	baseline := &Baseline{
		Mean:   0,
		StdDev: 1,
		Count:  0,
		Window: make([]float64, 0, 100),
	}
	a.baselines[metricName] = baseline
	return baseline
}

// updateBaseline updates baseline statistics with new value
func (a *AnomalyDetector) updateBaseline(baseline *Baseline, value float64) {
	baseline.Window = append(baseline.Window, value)
	if len(baseline.Window) > 100 {
		baseline.Window = baseline.Window[1:]
	}

	baseline.Count++

	// Calculate mean
	sum := 0.0
	for _, v := range baseline.Window {
		sum += v
	}
	baseline.Mean = sum / float64(len(baseline.Window))

	// Calculate standard deviation
	variance := 0.0
	for _, v := range baseline.Window {
		variance += (v - baseline.Mean) * (v - baseline.Mean)
	}
	baseline.StdDev = math.Sqrt(variance / float64(len(baseline.Window)))

	if baseline.StdDev == 0 {
		baseline.StdDev = 1 // Avoid division by zero
	}
}

// calculateProbability calculates the probability of the anomaly
func (a *AnomalyDetector) calculateProbability(metric Metric) float64 {
	baseline := a.baselines[metric.Name]
	zscore := math.Abs(metric.Value-baseline.Mean) / baseline.StdDev

	// Convert z-score to probability (0-1)
	probability := math.Min(zscore/10.0, 1.0)
	return probability
}
