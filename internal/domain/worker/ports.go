package worker

import (
	"context"

	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
)

// JobExecutor defines the interface for executing jobs
// Different job types can have different executors
type JobExecutor interface {
	Execute(ctx context.Context, job *queue.Job) (*ExecutionResult, error)
	CanHandle(jobType string) bool
}
