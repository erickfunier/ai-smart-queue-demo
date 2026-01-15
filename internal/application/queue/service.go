package queue

import (
	"context"
	"encoding/json"

	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
)

// Service orchestrates queue-related use cases
type Service struct {
	jobRepo      queue.JobRepository
	queueService queue.QueueService
	metrics      queue.MetricsService
}

// NewService creates a new queue application service
func NewService(
	jobRepo queue.JobRepository,
	queueService queue.QueueService,
	metrics queue.MetricsService,
) *Service {
	return &Service{
		jobRepo:      jobRepo,
		queueService: queueService,
		metrics:      metrics,
	}
}

// CreateJobCommand represents the data needed to create a job
type CreateJobCommand struct {
	Queue   string
	Type    string
	Payload any
}

// CreateJob creates a new job and enqueues it
func (s *Service) CreateJob(ctx context.Context, cmd CreateJobCommand) (*queue.Job, error) {
	// Convert payload to JSON
	payloadBytes, err := json.Marshal(cmd.Payload)
	if err != nil {
		return nil, err
	}

	// Create domain entity with business rules
	job, err := queue.NewJob(cmd.Queue, cmd.Type, payloadBytes)
	if err != nil {
		return nil, err
	}

	// Persist the job
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	// Enqueue for processing
	if err := s.queueService.Enqueue(ctx, job); err != nil {
		return nil, err
	}

	// Record metrics
	s.metrics.RecordJobCreated(job.Queue, job.Type)

	return job, nil
}

// GetJob retrieves a job by ID
func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (*queue.Job, error) {
	return s.jobRepo.GetByID(ctx, id)
}

// GetJobsByStatus retrieves jobs by status
func (s *Service) GetJobsByStatus(ctx context.Context, status queue.Status, limit int) ([]*queue.Job, error) {
	return s.jobRepo.FindByStatus(ctx, status, limit)
}

// UpdateJobStatus updates the status of a job
func (s *Service) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status queue.Status) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	// Apply business rules based on status
	switch status {
	case queue.StatusProcessing:
		job.MarkAsProcessing()
	case queue.StatusCompleted:
		job.MarkAsCompleted()
		s.metrics.RecordJobCompleted(job.Queue, job.Type, 0) // Duration can be calculated
	case queue.StatusFailed:
		job.MarkAsFailed(nil)
		s.metrics.RecordJobFailed(job.Queue, job.Type)
	}

	return s.jobRepo.Update(ctx, job)
}

// RetryJob retries a failed job
func (s *Service) RetryJob(ctx context.Context, jobID uuid.UUID, maxAttempts int) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	if !job.CanRetry(maxAttempts) {
		return queue.ErrMaxAttemptsReached
	}

	job.MarkAsRetrying()
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	// Re-enqueue the job
	if err := s.queueService.Enqueue(ctx, job); err != nil {
		return err
	}

	s.metrics.RecordJobRetried(job.Queue, job.Type)
	return nil
}

// GetDLQJobs retrieves dead letter queue jobs
func (s *Service) GetDLQJobs(ctx context.Context, limit, offset int) ([]*queue.Job, int64, error) {
	jobs, err := s.jobRepo.GetDLQJobs(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.jobRepo.CountDLQJobs(ctx)
	if err != nil {
		return nil, 0, err
	}

	return jobs, count, nil
}

// DeleteJob deletes a job
func (s *Service) DeleteJob(ctx context.Context, id uuid.UUID) error {
	return s.jobRepo.Delete(ctx, id)
}

// GetMetrics retrieves queue metrics
func (s *Service) GetMetrics(ctx context.Context) (map[string]any, error) {
	metrics := make(map[string]any)

	// Count jobs by status
	for _, status := range []queue.Status{
		queue.StatusPending,
		queue.StatusProcessing,
		queue.StatusCompleted,
		queue.StatusFailed,
	} {
		count, err := s.jobRepo.CountByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		metrics[string(status)] = count
	}

	dlqCount, err := s.jobRepo.CountDLQJobs(ctx)
	if err != nil {
		return nil, err
	}
	metrics["dlq"] = dlqCount

	return metrics, nil
}
