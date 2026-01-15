package insights

import (
	"context"
	"log"

	"github.com/erickfunier/ai-smart-queue/internal/domain/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
)

// Service orchestrates AI insights use cases
type Service struct {
	insightRepo insights.InsightRepository
	jobRepo     queue.JobRepository
	aiService   insights.AIService
}

// NewService creates a new insights application service
func NewService(
	insightRepo insights.InsightRepository,
	jobRepo queue.JobRepository,
	aiService insights.AIService,
) *Service {
	return &Service{
		insightRepo: insightRepo,
		jobRepo:     jobRepo,
		aiService:   aiService,
	}
}

// AnalyzeJobFailure analyzes a failed job and generates insights
func (s *Service) AnalyzeJobFailure(ctx context.Context, jobID uuid.UUID) (*insights.Insight, error) {
	log.Printf("[Insights] Starting AI analysis for failed job: id=%s", jobID)

	// Check if an insight already exists for this job (cache)
	existingInsight, err := s.insightRepo.GetByJobID(ctx, jobID)
	if err == nil && existingInsight != nil {
		log.Printf("[Insights] Using cached insight for job: id=%s, insight_id=%s", jobID, existingInsight.ID)
		return existingInsight, nil
	}

	log.Printf("[Insights] No cached insight found, proceeding with AI analysis: job_id=%s", jobID)
	// Get the failed job
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		log.Printf("[Insights] Failed to retrieve job: id=%s, error=%v", jobID, err)
		return nil, err
	}

	log.Printf("[Insights] Retrieved job: id=%s, type=%s, error=%s", job.ID, job.Type, job.Error)
	// Prepare analysis request
	request := &insights.AnalysisRequest{
		JobID:   job.ID.String(),
		Error:   job.Error,
		Payload: string(job.Payload),
	}

	// Call AI service for analysis
	log.Printf("[Insights] Calling AI service for analysis: job_id=%s", jobID)
	response, err := s.aiService.Analyze(ctx, request)
	if err != nil {
		log.Printf("[Insights] AI analysis failed: job_id=%s, error=%v", jobID, err)
		return nil, err
	}

	log.Printf("[Insights] AI analysis completed: job_id=%s, diagnosis=%s", jobID, response.Diagnosis)
	// Create insight from response
	insight, err := insights.NewInsight(jobID, response)
	if err != nil {
		log.Printf("[Insights] Failed to create insight: job_id=%s, error=%v", jobID, err)
		return nil, err
	}

	// Persist the insight
	log.Printf("[Insights] Persisting insight: id=%s, job_id=%s", insight.ID, jobID)
	if err := s.insightRepo.Create(ctx, insight); err != nil {
		log.Printf("[Insights] Failed to persist insight: error=%v", err)
		return nil, err
	}

	log.Printf("[Insights] Insight created successfully: id=%s, job_id=%s", insight.ID, jobID)
	return insight, nil
}

// GetInsight retrieves an insight by ID
func (s *Service) GetInsight(ctx context.Context, id uuid.UUID) (*insights.Insight, error) {
	return s.insightRepo.GetByID(ctx, id)
}

// GetInsightByJobID retrieves an insight for a specific job
func (s *Service) GetInsightByJobID(ctx context.Context, jobID uuid.UUID) (*insights.Insight, error) {
	return s.insightRepo.GetByJobID(ctx, jobID)
}

// ListInsights retrieves all insights with pagination
func (s *Service) ListInsights(ctx context.Context, limit, offset int) ([]*insights.Insight, error) {
	return s.insightRepo.List(ctx, limit, offset)
}

// ApplyInsightFix applies the suggested fix from an insight to a job
func (s *Service) ApplyInsightFix(ctx context.Context, insightID uuid.UUID) error {
	insight, err := s.insightRepo.GetByID(ctx, insightID)
	if err != nil {
		return err
	}

	job, err := s.jobRepo.GetByID(ctx, insight.JobID)
	if err != nil {
		return err
	}

	// Apply the suggested payload patch
	if len(insight.SuggestedFix.PayloadPatch) > 0 {
		patchedPayload, err := insight.ApplySuggestedFix(job.Payload)
		if err != nil {
			return err
		}
		job.Payload = patchedPayload
	}

	// Reset job for retry if recommended
	if insight.HasRetryRecommendation() {
		job.MarkAsRetrying()
	}

	return s.jobRepo.Update(ctx, job)
}
