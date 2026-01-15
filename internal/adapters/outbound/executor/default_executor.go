package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/erickfunier/ai-smart-queue/internal/domain/worker"
	"github.com/erickfunier/ai-smart-queue/internal/infrastructure/config"
)

// DefaultJobExecutor is a simple executor that handles basic job types
type DefaultJobExecutor struct {
	config *config.Config
	rng    *rand.Rand
}

// NewDefaultJobExecutor creates a new default job executor
func NewDefaultJobExecutor(cfg *config.Config) *DefaultJobExecutor {
	return &DefaultJobExecutor{
		config: cfg,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *DefaultJobExecutor) Execute(ctx context.Context, job *queue.Job) (*worker.ExecutionResult, error) {
	slog.InfoContext(ctx, "Executing job",
		slog.String("jobId", job.ID.String()),
		slog.String("jobType", job.Type),
		slog.String("queue", job.Queue),
	)

	// Parse payload
	var payload map[string]any
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		slog.ErrorContext(ctx, "Failed to parse job payload",
			slog.String("jobId", job.ID.String()),
			slog.String("error", err.Error()),
		)
		return &worker.ExecutionResult{
			Success: false,
			Error:   err,
		}, nil
	}

	// Simulate job execution based on type
	switch job.Type {
	case "email":
		return e.executeEmailJob(ctx, job.ID.String(), payload)
	case "notification":
		return e.executeNotificationJob(ctx, job.ID.String(), payload)
	case "data_processing":
		return e.executeDataProcessingJob(ctx, job.ID.String(), payload)
	default:
		return &worker.ExecutionResult{
			Success: false,
			Error:   errors.New("unsupported job type: " + job.Type),
		}, nil
	}
}

func (e *DefaultJobExecutor) CanHandle(jobType string) bool {
	supportedTypes := map[string]bool{
		"email":           true,
		"notification":    true,
		"data_processing": true,
	}
	return supportedTypes[jobType]
}

func (e *DefaultJobExecutor) executeEmailJob(ctx context.Context, jobID string, payload map[string]any) (*worker.ExecutionResult, error) {
	// Simulate email sending
	slog.InfoContext(ctx, "Sending email",
		slog.String("jobId", jobID),
		slog.String("to", fmt.Sprintf("%v", payload["to"])),
		slog.String("subject", fmt.Sprintf("%v", payload["subject"])),
	)

	// Check if simulation is enabled and should fail
	if e.shouldSimulateFailure() {
		errorMsg := e.getRandomError("email")
		slog.WarnContext(ctx, "Simulating email sending failure",
			slog.String("jobId", jobID),
			slog.String("error", errorMsg),
			slog.Bool("simulated", true),
		)
		return &worker.ExecutionResult{
			Success: false,
			Error:   errors.New(errorMsg),
		}, nil
	}

	// Add your actual email sending logic here
	// For now, we'll just simulate success

	slog.InfoContext(ctx, "Email sent successfully",
		slog.String("jobId", jobID),
		slog.String("to", fmt.Sprintf("%v", payload["to"])),
	)

	return &worker.ExecutionResult{
		Success: true,
		Output:  "Email sent successfully",
	}, nil
}

func (e *DefaultJobExecutor) executeNotificationJob(ctx context.Context, jobID string, payload map[string]any) (*worker.ExecutionResult, error) {
	// Simulate notification sending
	slog.InfoContext(ctx, "Sending notification",
		slog.String("jobId", jobID),
		slog.String("message", fmt.Sprintf("%v", payload["message"])),
	)

	// Check if simulation is enabled and should fail
	if e.shouldSimulateFailure() {
		errorMsg := e.getRandomError("notification")
		slog.WarnContext(ctx, "Simulating notification failure",
			slog.String("jobId", jobID),
			slog.String("error", errorMsg),
			slog.Bool("simulated", true),
		)
		return &worker.ExecutionResult{
			Success: false,
			Error:   errors.New(errorMsg),
		}, nil
	}

	// Add your actual notification logic here

	slog.InfoContext(ctx, "Notification sent successfully",
		slog.String("jobId", jobID),
	)

	return &worker.ExecutionResult{
		Success: true,
		Output:  "Notification sent successfully",
	}, nil
}

func (e *DefaultJobExecutor) executeDataProcessingJob(ctx context.Context, jobID string, payload map[string]any) (*worker.ExecutionResult, error) {
	// Simulate data processing
	slog.InfoContext(ctx, "Processing data",
		slog.String("jobId", jobID),
		slog.Any("data", payload["data"]),
	)

	// Check if simulation is enabled and should fail
	if e.shouldSimulateFailure() {
		errorMsg := e.getRandomError("data_processing")
		slog.WarnContext(ctx, "Simulating data processing failure",
			slog.String("jobId", jobID),
			slog.String("error", errorMsg),
			slog.Bool("simulated", true),
		)
		return &worker.ExecutionResult{
			Success: false,
			Error:   errors.New(errorMsg),
		}, nil
	}

	// Add your actual data processing logic here

	slog.InfoContext(ctx, "Data processed successfully",
		slog.String("jobId", jobID),
	)

	return &worker.ExecutionResult{
		Success: true,
		Output:  "Data processed successfully",
	}, nil
}

// shouldSimulateFailure determines if this execution should fail based on configuration
func (e *DefaultJobExecutor) shouldSimulateFailure() bool {
	if !e.config.Simulation.Enabled {
		return false
	}
	return e.rng.Float64() < e.config.Simulation.FailureRate
}

// getRandomError returns a random error message for the given job type
func (e *DefaultJobExecutor) getRandomError(jobType string) string {
	errors := map[string][]string{
		"email": {
			"failed to connect to SMTP server: connection timeout",
			"SMTP authentication failed: invalid credentials",
			"email rejected by recipient server: mailbox full",
			"email size exceeds maximum allowed limit",
			"DNS lookup failed for mail server",
		},
		"notification": {
			"push notification service unavailable",
			"invalid device token",
			"notification payload too large",
			"rate limit exceeded for notifications",
			"failed to establish SSL connection",
		},
		"data_processing": {
			"out of memory during data processing",
			"invalid data format: JSON parsing error",
			"database connection lost during transaction",
			"data validation failed: missing required fields",
			"processing timeout exceeded",
		},
	}

	jobErrors, ok := errors[jobType]
	if !ok {
		return fmt.Sprintf("unknown error processing %s job", jobType)
	}

	return jobErrors[e.rng.Intn(len(jobErrors))]
}
