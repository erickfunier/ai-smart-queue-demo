package worker

import (
	"context"
	"log/slog"
	"time"

	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/erickfunier/ai-smart-queue/internal/domain/worker"
)

// Service orchestrates worker-related use cases
type Service struct {
	jobRepo         queue.JobRepository
	queueService    queue.QueueService
	executor        worker.JobExecutor
	insightsService *appInsights.Service
	config          *worker.WorkerConfig
}

// NewService creates a new worker application service
func NewService(
	jobRepo queue.JobRepository,
	queueService queue.QueueService,
	executor worker.JobExecutor,
	insightsService *appInsights.Service,
	config *worker.WorkerConfig,
) *Service {
	return &Service{
		jobRepo:         jobRepo,
		queueService:    queueService,
		executor:        executor,
		insightsService: insightsService,
		config:          config,
	}
}

// ProcessNextJob processes the next available job from the queue
func (s *Service) ProcessNextJob(ctx context.Context) error {
	// Dequeue a job
	slog.InfoContext(ctx, "Polling queue for jobs",
		slog.String("queue", s.config.QueueName),
	)
	job, err := s.queueService.Dequeue(ctx, s.config.QueueName)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to dequeue job",
			slog.String("error", err.Error()),
			slog.String("queue", s.config.QueueName),
		)
		return err
	}

	if job == nil {
		// No jobs available
		slog.DebugContext(ctx, "No jobs available in queue",
			slog.String("queue", s.config.QueueName),
		)
		return nil
	}

	slog.InfoContext(ctx, "Dequeued job",
		slog.String("jobId", job.ID.String()),
		slog.String("jobType", job.Type),
		slog.String("queue", job.Queue),
		slog.Int("attempt", job.Attempts),
	)

	// Mark job as processing
	slog.InfoContext(ctx, "Marking job as processing",
		slog.String("jobId", job.ID.String()),
	)
	job.MarkAsProcessing()
	if err := s.jobRepo.Update(ctx, job); err != nil {
		slog.ErrorContext(ctx, "Failed to update job status to processing",
			slog.String("jobId", job.ID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Execute the job
	slog.InfoContext(ctx, "Executing job",
		slog.String("jobId", job.ID.String()),
		slog.String("jobType", job.Type),
	)
	result, err := s.executor.Execute(ctx, job)
	if err != nil || !result.Success {
		slog.WarnContext(ctx, "Job execution failed",
			slog.String("jobId", job.ID.String()),
			slog.String("error", result.Error.Error()),
		)
		return s.handleJobFailure(ctx, job, result.Error)
	}

	// Mark as completed
	slog.InfoContext(ctx, "Job executed successfully",
		slog.String("jobId", job.ID.String()),
	)
	job.MarkAsCompleted()
	if err := s.jobRepo.Update(ctx, job); err != nil {
		slog.ErrorContext(ctx, "Failed to update job status to completed",
			slog.String("jobId", job.ID.String()),
			slog.String("error", err.Error()),
		)
		return err
	}

	slog.InfoContext(ctx, "Job completed successfully",
		slog.String("jobId", job.ID.String()),
		slog.String("jobType", job.Type),
		slog.String("queue", job.Queue),
	)
	// Acknowledge from queue
	return s.queueService.Acknowledge(ctx, job.ID)
}

// handleJobFailure handles job failure with retry logic and AI insights
func (s *Service) handleJobFailure(ctx context.Context, job *queue.Job, execError error) error {
	job.MarkAsFailed(execError)

	// Generate AI insights for any job failure (before retry or permanent failure)
	if s.insightsService != nil && job.Attempts == 1 {
		jobIDStr := job.ID.String()
		slog.InfoContext(ctx, "Generating AI insights for failed job",
			slog.String("jobId", jobIDStr),
			slog.Int("attempt", job.Attempts),
		)
		go func() {
			// Run async to not block worker
			_, err := s.insightsService.AnalyzeJobFailure(context.Background(), job.ID)
			if err != nil {
				slog.ErrorContext(context.Background(), "Failed to generate AI insights",
					slog.String("jobId", jobIDStr),
					slog.String("error", err.Error()),
				)
			} else {
				slog.InfoContext(context.Background(), "AI insights generated successfully",
					slog.String("jobId", jobIDStr),
				)
			}
		}()
	}

	if job.CanRetry(s.config.MaxAttempts) {
		// Schedule retry with exponential backoff
		backoff := worker.CalculateBackoff(job.Attempts, s.config.BaseBackoffMs)
		retryTime := time.Now().UTC().Add(backoff)
		job.Schedule(retryTime)
		job.MarkAsRetrying()

		slog.InfoContext(ctx, "Job will retry with backoff",
			slog.String("jobId", job.ID.String()),
			slog.Duration("backoff", backoff),
			slog.Int("attempt", job.Attempts),
			slog.Int("maxAttempts", s.config.MaxAttempts),
		)

		// Update job in database first
		if err := s.jobRepo.Update(ctx, job); err != nil {
			slog.ErrorContext(ctx, "Failed to update job for retry",
				slog.String("jobId", job.ID.String()),
				slog.String("error", err.Error()),
			)
			return err
		}

		// Wait for the backoff period, then re-enqueue
		time.Sleep(backoff)
		slog.InfoContext(ctx, "Re-enqueueing job for retry",
			slog.String("jobId", job.ID.String()),
		)
		return s.queueService.Enqueue(ctx, job)
	} else {
		// Max attempts reached - move to DLQ (AI insights already generated on first failure)
		slog.WarnContext(ctx, "Job failed permanently, moving to DLQ",
			slog.String("jobId", job.ID.String()),
			slog.Int("attempts", job.Attempts),
			slog.String("reason", "max_attempts_exceeded"),
		)

		if err := s.jobRepo.MoveToDLQ(ctx, job.ID); err != nil {
			slog.ErrorContext(ctx, "Failed to move job to DLQ",
				slog.String("jobId", job.ID.String()),
				slog.String("error", err.Error()),
			)
			return err
		}

		slog.InfoContext(ctx, "Job moved to DLQ",
			slog.String("jobId", job.ID.String()),
		)
	}

	return s.jobRepo.Update(ctx, job)
}

// Start starts the worker processing loop
func (s *Service) Start(ctx context.Context) {
	slog.InfoContext(ctx, "Worker started",
		slog.String("queue", s.config.QueueName),
		slog.Duration("pollInterval", s.config.PollInterval),
		slog.Int("maxAttempts", s.config.MaxAttempts),
	)

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Worker shutting down",
				slog.String("queue", s.config.QueueName),
			)
			return
		case <-ticker.C:
			if err := s.ProcessNextJob(ctx); err != nil {
				slog.ErrorContext(ctx, "Error processing job",
					slog.String("error", err.Error()),
				)
			}
		}
	}
}
