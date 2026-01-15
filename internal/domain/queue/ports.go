package queue

import (
	"context"

	"github.com/google/uuid"
)

// JobRepository defines the interface for job persistence
// This is a port (output port) - secondary adapter will implement this
type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*Job, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query methods
	FindPendingJobs(ctx context.Context, queue string, limit int) ([]*Job, error)
	FindByStatus(ctx context.Context, status Status, limit int) ([]*Job, error)
	CountByStatus(ctx context.Context, status Status) (int64, error)

	// Dead letter queue
	GetDLQJobs(ctx context.Context, limit, offset int) ([]*Job, error)
	MoveToDLQ(ctx context.Context, jobID uuid.UUID) error
	CountDLQJobs(ctx context.Context) (int64, error)
}

// QueueService defines the interface for queue operations
// This will be used by workers to dequeue jobs
type QueueService interface {
	Enqueue(ctx context.Context, job *Job) error
	Dequeue(ctx context.Context, queueName string) (*Job, error)
	Acknowledge(ctx context.Context, jobID uuid.UUID) error
}

// MetricsService defines the interface for metrics collection
type MetricsService interface {
	RecordJobCreated(queue, jobType string)
	RecordJobCompleted(queue, jobType string, duration float64)
	RecordJobFailed(queue, jobType string)
	RecordJobRetried(queue, jobType string)
}
