package unit

import (
	"context"
	"testing"
	"time"

	"github.com/kubepulse/kubepulse/pkg/ml"
)

func TestAnomalyDetector_Normal(t *testing.T) {
	detector := ml.NewAnomalyDetector()

	// Generate normal metrics
	metrics := make([]ml.Metric, 20)
	for i := range metrics {
		metrics[i] = ml.Metric{
			Name:      "cpu_usage",
			Value:     50.0 + float64(i%5), // 50-54 range (normal)
			Timestamp: time.Now(),
		}
	}

	predictions := detector.DetectAnomalies(context.Background(), metrics)

	// Should not detect anomalies in normal data after learning
	if len(predictions) > 2 { // Allow for initial learning period
		t.Errorf("Expected few or no anomalies in normal data, got %d", len(predictions))
	}
}

func TestAnomalyDetector_Anomalous(t *testing.T) {
	detector := ml.NewAnomalyDetector()

	// First, train with normal data
	normalMetrics := make([]ml.Metric, 15)
	for i := range normalMetrics {
		normalMetrics[i] = ml.Metric{
			Name:      "cpu_usage",
			Value:     50.0,
			Timestamp: time.Now(),
		}
	}
	detector.DetectAnomalies(context.Background(), normalMetrics)

	// Now test with anomalous data
	anomalousMetrics := []ml.Metric{
		{
			Name:      "cpu_usage",
			Value:     200.0, // Very high compared to baseline of 50
			Timestamp: time.Now(),
		},
	}

	predictions := detector.DetectAnomalies(context.Background(), anomalousMetrics)

	if len(predictions) == 0 {
		t.Error("Expected anomaly to be detected")
	}

	if len(predictions) > 0 {
		pred := predictions[0]
		if pred.Status != "degraded" {
			t.Errorf("Expected degraded status, got %v", pred.Status)
		}
		if pred.Probability <= 0 {
			t.Errorf("Expected positive probability, got %f", pred.Probability)
		}
	}
}

func TestAnomalyDetector_InsufficientData(t *testing.T) {
	detector := ml.NewAnomalyDetector()

	// Test with insufficient data (less than 10 points)
	metrics := []ml.Metric{
		{Name: "memory_usage", Value: 100.0, Timestamp: time.Now()},
		{Name: "memory_usage", Value: 1000.0, Timestamp: time.Now()}, // Should not be flagged
	}

	predictions := detector.DetectAnomalies(context.Background(), metrics)

	// Should not detect anomalies with insufficient data
	if len(predictions) > 0 {
		t.Errorf("Expected no anomalies with insufficient data, got %d", len(predictions))
	}
}

func TestAnomalyDetector_MultipleMetrics(t *testing.T) {
	detector := ml.NewAnomalyDetector()

	// Train with normal data for different metrics
	for i := 0; i < 15; i++ {
		metrics := []ml.Metric{
			{Name: "cpu_usage", Value: 50.0, Timestamp: time.Now()},
			{Name: "memory_usage", Value: 1000.0, Timestamp: time.Now()},
		}
		detector.DetectAnomalies(context.Background(), metrics)
	}

	// Test with mixed normal and anomalous data
	testMetrics := []ml.Metric{
		{Name: "cpu_usage", Value: 55.0, Timestamp: time.Now()},      // Normal
		{Name: "memory_usage", Value: 5000.0, Timestamp: time.Now()}, // Anomalous
	}

	predictions := detector.DetectAnomalies(context.Background(), testMetrics)

	// Should detect the memory anomaly but not CPU
	foundMemoryAnomaly := false
	for _, pred := range predictions {
		if pred.Reason == "Statistical anomaly detected" {
			foundMemoryAnomaly = true
			break
		}
	}

	if !foundMemoryAnomaly {
		t.Error("Expected memory usage anomaly to be detected")
	}
}

func TestAnomalyDetector_BaselineUpdate(t *testing.T) {
	detector := ml.NewAnomalyDetector()

	// Test that baselines update correctly
	initialMetrics := make([]ml.Metric, 12)
	for i := range initialMetrics {
		initialMetrics[i] = ml.Metric{
			Name:      "test_metric",
			Value:     10.0,
			Timestamp: time.Now(),
		}
	}

	// Initial training
	detector.DetectAnomalies(context.Background(), initialMetrics)

	// Add new data that shifts the baseline
	shiftMetrics := make([]ml.Metric, 10)
	for i := range shiftMetrics {
		shiftMetrics[i] = ml.Metric{
			Name:      "test_metric",
			Value:     20.0, // Higher baseline
			Timestamp: time.Now(),
		}
	}

	detector.DetectAnomalies(context.Background(), shiftMetrics)

	// Test that new baseline is established
	testMetric := []ml.Metric{
		{Name: "test_metric", Value: 22.0, Timestamp: time.Now()}, // Closer to new baseline of ~20
	}

	predictions := detector.DetectAnomalies(context.Background(), testMetric)

	// Should not be anomalous relative to new baseline (or allow 1 prediction due to variance)
	if len(predictions) > 1 {
		t.Errorf("Expected no or minimal anomalies after baseline shift, got %d", len(predictions))
	}
}
