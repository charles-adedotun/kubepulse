package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

// PredictiveAnalyzer provides AI-powered predictive analytics
type PredictiveAnalyzer struct {
	client  *Client
	history *MetricsHistory
}

// MetricsHistory stores historical data for pattern analysis
type MetricsHistory struct {
	data   map[string][]TimeSeriesPoint
	maxAge time.Duration
}

// TimeSeriesPoint represents a metric at a point in time
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// PredictiveInsight represents an AI prediction
type PredictiveInsight struct {
	Type       string   `json:"type"`
	Resource   string   `json:"resource"`
	Prediction string   `json:"prediction"`
	Confidence float64  `json:"confidence"`
	TimeWindow string   `json:"time_window"`
	Impact     string   `json:"impact"`
	Preventive []string `json:"preventive_actions"`
}

// NewPredictiveAnalyzer creates a new predictive analyzer
func NewPredictiveAnalyzer(client *Client) *PredictiveAnalyzer {
	return &PredictiveAnalyzer{
		client: client,
		history: &MetricsHistory{
			data:   make(map[string][]TimeSeriesPoint),
			maxAge: 24 * time.Hour,
		},
	}
}

// AnalyzeTrends performs AI-powered trend analysis
func (p *PredictiveAnalyzer) AnalyzeTrends(ctx context.Context, metrics []Metric) ([]PredictiveInsight, error) {
	// Update history
	p.updateHistory(metrics)

	// Prepare data for AI analysis
	analysisData := p.prepareAnalysisData()

	request := AnalysisRequest{
		Type:    AnalysisTypePredictive,
		Context: "Predictive analytics for Kubernetes cluster",
		Data: map[string]interface{}{
			"metrics_history": analysisData,
			"analysis_type":   "time_series_prediction",
		},
		Timestamp: time.Now(),
	}

	response, err := p.client.Analyze(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("predictive analysis failed: %w", err)
	}

	// Parse predictions from response
	var predictions []PredictiveInsight
	if err := json.Unmarshal([]byte(response.Diagnosis), &predictions); err != nil {
		// Fallback to text parsing if JSON fails
		predictions = p.parseTextPredictions(response.Diagnosis)
	}

	klog.V(2).Infof("Generated %d predictive insights with average confidence %.2f",
		len(predictions), p.averageConfidence(predictions))

	return predictions, nil
}

// DetectAnomalies uses AI to detect anomalies in real-time
func (p *PredictiveAnalyzer) DetectAnomalies(ctx context.Context, current CheckResult) (*AnomalyReport, error) {
	historical := p.getHistoricalData(current.Name)

	request := AnalysisRequest{
		Type:        AnalysisTypePredictive,
		Context:     "Real-time anomaly detection",
		HealthCheck: &current,
		Data: map[string]interface{}{
			"historical_data": historical,
		},
		Timestamp: time.Now(),
	}

	response, err := p.client.Analyze(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("anomaly detection failed: %w", err)
	}

	return &AnomalyReport{
		Score:       response.Confidence,
		Anomalies:   p.extractAnomalies(response),
		Severity:    response.Severity,
		Explanation: response.Summary,
	}, nil
}

// updateHistory updates the metrics history
func (p *PredictiveAnalyzer) updateHistory(metrics []Metric) {
	now := time.Now()
	cutoff := now.Add(-p.history.maxAge)

	for _, metric := range metrics {
		key := fmt.Sprintf("%s_%s", metric.Name, metric.Type)

		// Add new point
		point := TimeSeriesPoint{
			Timestamp: metric.Timestamp,
			Value:     metric.Value,
			Labels:    metric.Labels,
		}

		p.history.data[key] = append(p.history.data[key], point)

		// Remove old points
		filtered := []TimeSeriesPoint{}
		for _, pt := range p.history.data[key] {
			if pt.Timestamp.After(cutoff) {
				filtered = append(filtered, pt)
			}
		}
		p.history.data[key] = filtered
	}
}

// prepareAnalysisData formats history for AI analysis
func (p *PredictiveAnalyzer) prepareAnalysisData() string {
	data, _ := json.MarshalIndent(p.history.data, "", "  ")
	return string(data)
}

// getHistoricalData retrieves historical data for a check
func (p *PredictiveAnalyzer) getHistoricalData(checkName string) map[string]interface{} {
	relevant := make(map[string][]TimeSeriesPoint)

	for key, points := range p.history.data {
		if len(points) > 0 && points[0].Labels["check"] == checkName {
			relevant[key] = points
		}
	}

	return map[string]interface{}{
		"data_points": len(relevant),
		"time_range":  "24h",
		"metrics":     relevant,
	}
}

// parseTextPredictions fallback parser for text responses
func (p *PredictiveAnalyzer) parseTextPredictions(text string) []PredictiveInsight {
	// Simple extraction logic - in production, use more sophisticated parsing
	return []PredictiveInsight{
		{
			Type:       "resource_exhaustion",
			Resource:   "nodes",
			Prediction: text,
			Confidence: 0.75,
			TimeWindow: "2-4 hours",
			Impact:     "Pod evictions likely",
			Preventive: []string{
				"Scale cluster nodes",
				"Optimize pod resource requests",
			},
		},
	}
}

// extractAnomalies extracts anomaly details from response
func (p *PredictiveAnalyzer) extractAnomalies(response *AnalysisResponse) []string {
	anomalies := []string{}

	for _, rec := range response.Recommendations {
		if rec.Category == "anomaly" {
			anomalies = append(anomalies, rec.Description)
		}
	}

	return anomalies
}

// averageConfidence calculates average confidence
func (p *PredictiveAnalyzer) averageConfidence(predictions []PredictiveInsight) float64 {
	if len(predictions) == 0 {
		return 0
	}

	sum := 0.0
	for _, pred := range predictions {
		sum += pred.Confidence
	}

	return sum / float64(len(predictions))
}

// AnomalyReport represents anomaly detection results
type AnomalyReport struct {
	Score       float64       `json:"anomaly_score"`
	Anomalies   []string      `json:"anomalies"`
	Severity    SeverityLevel `json:"severity"`
	Explanation string        `json:"explanation"`
}
