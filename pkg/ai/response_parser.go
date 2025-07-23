package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// StructuredResponse represents a more reliable AI response format
type StructuredResponse struct {
	Summary         string                 `json:"summary"`
	Diagnosis       string                 `json:"diagnosis"`
	Confidence      float64                `json:"confidence"`
	Severity        SeverityLevel          `json:"severity"`
	Recommendations []Recommendation       `json:"recommendations"`
	Actions         []SuggestedAction      `json:"actions"`
	Context         map[string]interface{} `json:"context"`
}

// ResponseParser handles parsing AI responses with multiple fallback strategies
type ResponseParser struct {
	patterns map[string]*regexp.Regexp
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{
		patterns: map[string]*regexp.Regexp{
			"summary":    regexp.MustCompile(`(?i)(?:summary|overview):\s*(.+?)(?:\n\n|\n[A-Z]|$)`),
			"diagnosis":  regexp.MustCompile(`(?i)(?:diagnosis|analysis):\s*(.+?)(?:\n\n|\n[A-Z]|$)`),
			"confidence": regexp.MustCompile(`(?i)confidence[:\s]+(\d+(?:\.\d+)?)[%\s]?`),
			"severity":   regexp.MustCompile(`(?i)severity[:\s]+(critical|high|medium|low)`),
			"json_block": regexp.MustCompile("```json\\s*([\\s\\S]*?)\\s*```"),
			"action_cmd": regexp.MustCompile(`kubectl\s+[^\n]+`),
			"numbered":   regexp.MustCompile(`^\d+\.\s+(.+)$`),
			"bulleted":   regexp.MustCompile(`^[-*]\s+(.+)$`),
		},
	}
}

// ParseResponse attempts to parse AI response using multiple strategies
func (p *ResponseParser) ParseResponse(text string, request AnalysisRequest) (*AnalysisResponse, error) {
	// Strategy 1: Try to parse as JSON if it contains JSON blocks
	if structured := p.tryParseJSON(text); structured != nil {
		return p.convertStructuredResponse(structured), nil
	}

	// Strategy 2: Use pattern-based extraction with improved logic
	response := &AnalysisResponse{
		Summary:         p.extractSummary(text),
		Diagnosis:       p.extractDiagnosis(text),
		Confidence:      p.extractConfidence(text),
		Severity:        p.extractSeverity(text),
		Recommendations: p.extractRecommendations(text),
		Actions:         p.extractActions(text),
		Context:         make(map[string]interface{}),
	}

	// Set reasonable defaults
	if response.Confidence == 0 {
		response.Confidence = 0.7 // Moderate confidence as default
	}
	if response.Severity == "" {
		response.Severity = SeverityMedium
	}
	if response.Summary == "" {
		response.Summary = p.generateFallbackSummary(text, request)
	}

	return response, nil
}

// tryParseJSON attempts to extract and parse JSON from the response
func (p *ResponseParser) tryParseJSON(text string) *StructuredResponse {
	// Look for JSON blocks first
	if matches := p.patterns["json_block"].FindStringSubmatch(text); len(matches) > 1 {
		var structured StructuredResponse
		if err := json.Unmarshal([]byte(matches[1]), &structured); err == nil {
			return &structured
		}
	}

	// Try to parse the entire response as JSON
	var structured StructuredResponse
	if err := json.Unmarshal([]byte(text), &structured); err == nil {
		return &structured
	}

	return nil
}

// convertStructuredResponse converts structured response to analysis response
func (p *ResponseParser) convertStructuredResponse(sr *StructuredResponse) *AnalysisResponse {
	return &AnalysisResponse{
		Summary:         sr.Summary,
		Diagnosis:       sr.Diagnosis,
		Confidence:      sr.Confidence,
		Severity:        sr.Severity,
		Recommendations: sr.Recommendations,
		Actions:         sr.Actions,
		Context:         sr.Context,
	}
}

// extractSummary extracts summary with improved logic
func (p *ResponseParser) extractSummary(text string) string {
	// Try pattern matching first
	if matches := p.patterns["summary"].FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: look for first substantial paragraph
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 50 && !strings.HasPrefix(line, "#") && !strings.Contains(line, ":") {
			// Truncate if too long
			if len(line) > 200 {
				line = line[:197] + "..."
			}
			return line
		}
	}

	// Last resort: first meaningful sentence
	sentences := strings.Split(text, ".")
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 {
			return sentence + "."
		}
	}

	return "AI analysis completed"
}

// extractDiagnosis extracts diagnosis with better context
func (p *ResponseParser) extractDiagnosis(text string) string {
	// Try pattern matching
	if matches := p.patterns["diagnosis"].FindStringSubmatch(text); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Look for detailed sections
	keywords := []string{"diagnosis", "analysis", "root cause", "issue", "problem"}
	for _, keyword := range keywords {
		if idx := strings.Index(strings.ToLower(text), keyword+":"); idx != -1 {
			remaining := text[idx:]
			if endIdx := strings.Index(remaining, "\n\n"); endIdx != -1 {
				return strings.TrimSpace(remaining[len(keyword)+1 : endIdx])
			}
		}
	}

	// Fallback to extracting technical paragraphs
	paragraphs := strings.Split(text, "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) > 100 && (strings.Contains(para, "kubectl") ||
			strings.Contains(para, "pod") || strings.Contains(para, "node")) {
			return para
		}
	}

	return text // Return full text as last resort
}

// extractConfidence uses multiple indicators
func (p *ResponseParser) extractConfidence(text string) float64 {
	// Try direct pattern matching
	if matches := p.patterns["confidence"].FindStringSubmatch(text); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			if val > 1 { // Assume percentage
				return val / 100
			}
			return val
		}
	}

	// Use keywords to estimate confidence
	text = strings.ToLower(text)
	if strings.Contains(text, "definitely") || strings.Contains(text, "certain") {
		return 0.95
	}
	if strings.Contains(text, "likely") || strings.Contains(text, "probably") {
		return 0.8
	}
	if strings.Contains(text, "might") || strings.Contains(text, "possibly") {
		return 0.6
	}
	if strings.Contains(text, "uncertain") || strings.Contains(text, "unclear") {
		return 0.4
	}

	return 0.7 // Default moderate confidence
}

// extractSeverity with pattern matching and keywords
func (p *ResponseParser) extractSeverity(text string) SeverityLevel {
	// Try pattern matching
	if matches := p.patterns["severity"].FindStringSubmatch(text); len(matches) > 1 {
		switch strings.ToLower(matches[1]) {
		case "critical":
			return SeverityCritical
		case "high":
			return SeverityHigh
		case "medium":
			return SeverityMedium
		case "low":
			return SeverityLow
		}
	}

	// Keyword-based detection
	text = strings.ToLower(text)
	criticalKeywords := []string{"critical", "urgent", "emergency", "down", "failed"}
	highKeywords := []string{"important", "significant", "major"}
	lowKeywords := []string{"minor", "trivial", "cosmetic"}

	for _, keyword := range criticalKeywords {
		if strings.Contains(text, keyword) {
			return SeverityCritical
		}
	}
	for _, keyword := range highKeywords {
		if strings.Contains(text, keyword) {
			return SeverityHigh
		}
	}
	for _, keyword := range lowKeywords {
		if strings.Contains(text, keyword) {
			return SeverityLow
		}
	}

	return SeverityMedium
}

// extractRecommendations finds numbered or bulleted lists
func (p *ResponseParser) extractRecommendations(text string) []Recommendation {
	var recommendations []Recommendation
	lines := strings.Split(text, "\n")
	priority := 1

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for numbered or bulleted items
		if matches := p.patterns["numbered"].FindStringSubmatch(line); len(matches) > 1 {
			rec := Recommendation{
				Title:       p.generateTitle(matches[1]),
				Description: matches[1],
				Priority:    priority,
				Category:    "operational",
				Impact:      "medium",
				Effort:      "medium",
			}
			recommendations = append(recommendations, rec)
			priority++
		} else if matches := p.patterns["bulleted"].FindStringSubmatch(line); len(matches) > 1 {
			rec := Recommendation{
				Title:       p.generateTitle(matches[1]),
				Description: matches[1],
				Priority:    priority,
				Category:    "general",
				Impact:      "medium",
				Effort:      "low",
			}
			recommendations = append(recommendations, rec)
			priority++
		}
	}

	// If no structured recommendations found, create generic ones
	if len(recommendations) == 0 && len(text) > 100 {
		recommendations = append(recommendations, Recommendation{
			Title:       "Review AI Analysis",
			Description: "Review the detailed analysis provided by the AI system",
			Priority:    1,
			Category:    "general",
			Impact:      "informational",
			Effort:      "low",
		})
	}

	return recommendations
}

// extractActions finds kubectl commands and action items
func (p *ResponseParser) extractActions(text string) []SuggestedAction {
	var actions []SuggestedAction
	actionID := 1

	// Find kubectl commands
	cmdMatches := p.patterns["action_cmd"].FindAllString(text, -1)
	for _, cmd := range cmdMatches {
		action := SuggestedAction{
			ID:               fmt.Sprintf("action-%d", actionID),
			Type:             ActionTypeKubectl,
			Title:            "Execute kubectl command",
			Description:      cmd,
			Command:          cmd,
			IsAutomatic:      false,
			RequiresApproval: true,
		}
		actions = append(actions, action)
		actionID++
	}

	return actions
}

// generateTitle creates a concise title from a description
func (p *ResponseParser) generateTitle(description string) string {
	words := strings.Fields(description)
	if len(words) <= 6 {
		return description
	}
	return strings.Join(words[:6], " ") + "..."
}

// generateFallbackSummary creates a summary when none is found
func (p *ResponseParser) generateFallbackSummary(text string, request AnalysisRequest) string {
	switch request.Type {
	case AnalysisTypeDiagnostic:
		return "Diagnostic analysis completed with findings"
	case AnalysisTypeHealing:
		return "Healing recommendations generated"
	case AnalysisTypePredictive:
		return "Predictive analysis completed"
	case AnalysisTypeRootCause:
		return "Root cause analysis completed"
	default:
		return "AI analysis completed successfully"
	}
}
