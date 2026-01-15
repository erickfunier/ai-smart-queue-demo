package insights

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Insight represents an AI-generated analysis of a job failure
type Insight struct {
	ID             uuid.UUID
	JobID          uuid.UUID
	Diagnosis      string
	Recommendation string
	SuggestedFix   SuggestedFix
	CreatedAt      time.Time
}

// SuggestedFix contains AI-recommended fixes for job failures
type SuggestedFix struct {
	TimeoutSeconds int            `json:"timeout_seconds"`
	MaxRetries     int            `json:"max_retries"`
	PayloadPatch   map[string]any `json:"payload_patch"`
}

// AnalysisRequest represents the data needed for AI analysis
type AnalysisRequest struct {
	JobID   string
	Error   string
	Payload string
}

// AnalysisResponse represents the AI analysis result
type AnalysisResponse struct {
	Diagnosis      string       `json:"diagnosis"`
	Recommendation string       `json:"recommendation"`
	SuggestedFix   SuggestedFix `json:"suggested_fix"`
}

var (
	ErrInvalidJobID        = errors.New("invalid job ID")
	ErrAnalysisFailed      = errors.New("AI analysis failed")
	ErrInsightNotFound     = errors.New("insight not found")
	ErrInvalidAnalysisData = errors.New("invalid analysis data")
)

// NewInsight creates a new insight from an analysis response
func NewInsight(jobID uuid.UUID, response *AnalysisResponse) (*Insight, error) {
	if jobID == uuid.Nil {
		return nil, ErrInvalidJobID
	}
	if response == nil || response.Diagnosis == "" {
		return nil, ErrInvalidAnalysisData
	}

	return &Insight{
		ID:             uuid.New(),
		JobID:          jobID,
		Diagnosis:      response.Diagnosis,
		Recommendation: response.Recommendation,
		SuggestedFix:   response.SuggestedFix,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// ApplySuggestedFix applies the suggested fix to a job payload
func (i *Insight) ApplySuggestedFix(originalPayload []byte) ([]byte, error) {
	if len(i.SuggestedFix.PayloadPatch) == 0 {
		return originalPayload, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(originalPayload, &payload); err != nil {
		return nil, err
	}

	// Apply patches
	for key, value := range i.SuggestedFix.PayloadPatch {
		payload[key] = value
	}

	return json.Marshal(payload)
}

// HasTimeoutRecommendation checks if the insight recommends a timeout adjustment
func (i *Insight) HasTimeoutRecommendation() bool {
	return i.SuggestedFix.TimeoutSeconds > 0
}

// HasRetryRecommendation checks if the insight recommends retry adjustments
func (i *Insight) HasRetryRecommendation() bool {
	return i.SuggestedFix.MaxRetries > 0
}
