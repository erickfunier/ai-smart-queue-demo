package insights

import (
	"context"

	"github.com/google/uuid"
)

// InsightRepository defines the interface for insight persistence
type InsightRepository interface {
	Create(ctx context.Context, insight *Insight) error
	GetByID(ctx context.Context, id uuid.UUID) (*Insight, error)
	GetByJobID(ctx context.Context, jobID uuid.UUID) (*Insight, error)
	List(ctx context.Context, limit, offset int) ([]*Insight, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// AIService defines the interface for AI analysis
// This is a port that will be implemented by an adapter (e.g., Ollama)
type AIService interface {
	Analyze(ctx context.Context, request *AnalysisRequest) (*AnalysisResponse, error)
}
