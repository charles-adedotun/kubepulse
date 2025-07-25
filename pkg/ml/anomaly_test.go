package ml

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestNewAnomalyDetector(t *testing.T) {
	detector := NewAnomalyDetector()

	if detector == nil {
		t.Fatal("expected non-nil AnomalyDetector")
	}

	if detector.baselines == nil {
		t.Fatal("expected initialized baselines map")
	}

	if detector.window != 24*time.Hour {
		t.Errorf("expected window 24h, got %v", detector.window)
	}

	if detector.threshold != 2.0 {
		t.Errorf("expected threshold 2.0, got %f", detector.threshold)
	}
}

func TestAnomalyDetector_DetectAnomalies_InsufficientData(t *testing.T) {
	detector := NewAnomalyDetector()
	ctx := context.Background()

	metrics := []Metric{
		{Name: "cpu", Value: 50.0, Timestamp: time.Now()},
		{Name: "cpu", Value: 55.0, Timestamp: time.Now()},
	}

	predictions := detector.DetectAnomalies(ctx, metrics)

	// Should have no predictions due to insufficient data
	if len(predictions) != 0 {
		t.Errorf("expected 0 predictions with insufficient data, got %d", len(predictions))
	}
}

func TestAnomalyDetector_DetectAnomalies_Normal(t *testing.T) {
	detector := NewAnomalyDetector()
	ctx := context.Background()

	// Build up baseline with normal values
	normalMetrics := make([]Metric, 15)
	for i := 0; i < 15; i++ {
		normalMetrics[i] = Metric{
			Name:      "cpu",
			Value:     50.0 + float64(i%5), // Values between 50-54
			Timestamp: time.Now(),
		}
	}

	predictions := detector.DetectAnomalies(ctx, normalMetrics)

	// Should have no predictions for normal values
	if len(predictions) != 0 {
		t.Errorf("expected 0 predictions for normal values, got %d", len(predictions))
	}
}

func TestAnomalyDetector_DetectAnomalies_Anomalous(t *testing.T) {
	detector := NewAnomalyDetector()
	ctx := context.Background()

	// Build up baseline with normal values
	normalMetrics := make([]Metric, 12)
	for i := 0; i < 12; i++ {
		normalMetrics[i] = Metric{
			Name:      "cpu",
			Value:     50.0,
			Timestamp: time.Now(),
		}
	}

	// Process normal metrics first to establish baseline
	detector.DetectAnomalies(ctx, normalMetrics)

	// Now test with anomalous value
	anomalousMetrics := []Metric{
		{Name: "cpu", Value: 500.0, Timestamp: time.Now()}, // Very high value
	}

	predictions := detector.DetectAnomalies(ctx, anomalousMetrics)

	if len(predictions) != 1 {
		t.Errorf("expected 1 prediction for anomalous value, got %d", len(predictions))
	}

	if len(predictions) > 0 {
		prediction := predictions[0]
		if prediction.Status != "degraded" {
			t.Errorf("expected status 'degraded', got %s", prediction.Status)
		}
		if prediction.Probability <= 0 {
			t.Errorf("expected positive probability, got %f", prediction.Probability)
		}
		if prediction.Reason != "Statistical anomaly detected" {
			t.Errorf("expected reason 'Statistical anomaly detected', got %s", prediction.Reason)
		}
	}
}

func TestAnomalyDetector_GetOrCreateBaseline(t *testing.T) {
	detector := NewAnomalyDetector()

	// Test creating new baseline
	baseline1 := detector.getOrCreateBaseline("test_metric")
	if baseline1 == nil {
		t.Fatal("expected non-nil baseline")
	}

	if baseline1.Mean != 0 {
		t.Errorf("expected initial mean 0, got %f", baseline1.Mean)
	}

	if baseline1.StdDev != 1 {
		t.Errorf("expected initial stddev 1, got %f", baseline1.StdDev)
	}

	if baseline1.Count != 0 {
		t.Errorf("expected initial count 0, got %d", baseline1.Count)
	}

	if cap(baseline1.Window) != 100 {
		t.Errorf("expected window capacity 100, got %d", cap(baseline1.Window))
	}

	// Test retrieving existing baseline
	baseline2 := detector.getOrCreateBaseline("test_metric")
	if baseline1 != baseline2 {
		t.Error("expected same baseline instance")
	}
}

func TestAnomalyDetector_UpdateBaseline(t *testing.T) {
	detector := NewAnomalyDetector()
	baseline := detector.getOrCreateBaseline("test_metric")

	// Add some values
	values := []float64{10.0, 20.0, 30.0}
	for _, value := range values {
		detector.updateBaseline(baseline, value)
	}

	if baseline.Count != 3 {
		t.Errorf("expected count 3, got %d", baseline.Count)
	}

	if len(baseline.Window) != 3 {
		t.Errorf("expected window length 3, got %d", len(baseline.Window))
	}

	expectedMean := 20.0
	if baseline.Mean != expectedMean {
		t.Errorf("expected mean %f, got %f", expectedMean, baseline.Mean)
	}

	// Check standard deviation calculation (approximately)
	// Variance = ((10-20)² + (20-20)² + (30-20)²) / 3 = (100 + 0 + 100) / 3
	expectedVariance := (100.0 + 0.0 + 100.0) / 3.0
	expectedStdDev := math.Sqrt(expectedVariance)
	if math.Abs(baseline.StdDev-expectedStdDev) > 0.1 {
		t.Errorf("expected stddev approximately %f, got %f", expectedStdDev, baseline.StdDev)
	}
}

func TestAnomalyDetector_UpdateBaseline_WindowLimit(t *testing.T) {
	detector := NewAnomalyDetector()
	baseline := detector.getOrCreateBaseline("test_metric")

	// Add more than 100 values to test window limit
	for i := 0; i < 150; i++ {
		detector.updateBaseline(baseline, float64(i))
	}

	if len(baseline.Window) != 100 {
		t.Errorf("expected window length 100, got %d", len(baseline.Window))
	}

	// Check that oldest values were removed
	if baseline.Window[0] != 50.0 {
		t.Errorf("expected first value 50.0 (oldest after sliding), got %f", baseline.Window[0])
	}

	if baseline.Window[99] != 149.0 {
		t.Errorf("expected last value 149.0, got %f", baseline.Window[99])
	}
}

func TestAnomalyDetector_UpdateBaseline_ZeroStdDev(t *testing.T) {
	detector := NewAnomalyDetector()
	baseline := detector.getOrCreateBaseline("test_metric")

	// Add identical values to test zero standard deviation handling
	for i := 0; i < 5; i++ {
		detector.updateBaseline(baseline, 42.0)
	}

	if baseline.StdDev != 1.0 {
		t.Errorf("expected stddev 1.0 (default for zero variance), got %f", baseline.StdDev)
	}
}

func TestAnomalyDetector_CalculateProbability(t *testing.T) {
	detector := NewAnomalyDetector()

	// Set up a baseline manually
	baseline := &Baseline{
		Mean:   50.0,
		StdDev: 10.0,
		Count:  10,
		Window: []float64{40, 45, 50, 55, 60},
	}
	detector.baselines["test_metric"] = baseline

	metric := Metric{
		Name:  "test_metric",
		Value: 80.0, // 3 standard deviations away
	}

	probability := detector.calculateProbability(metric)

	expectedZscore := 3.0 // |80 - 50| / 10
	expectedProbability := math.Min(expectedZscore/10.0, 1.0)

	if math.Abs(probability-expectedProbability) > 0.001 {
		t.Errorf("expected probability %f, got %f", expectedProbability, probability)
	}
}

func TestAnomalyDetector_IsAnomalous(t *testing.T) {
	detector := NewAnomalyDetector()

	// Build baseline with normal values
	for i := 0; i < 12; i++ {
		metric := Metric{Name: "test", Value: 50.0}
		detector.isAnomalous(metric)
	}

	// Test normal value
	normalMetric := Metric{Name: "test", Value: 52.0}
	if detector.isAnomalous(normalMetric) {
		t.Error("expected normal value to not be anomalous")
	}

	// Test anomalous value (way outside threshold)
	anomalousMetric := Metric{Name: "test", Value: 500.0}
	if !detector.isAnomalous(anomalousMetric) {
		t.Error("expected anomalous value to be detected")
	}
}
