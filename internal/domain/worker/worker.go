package worker

import (
	"errors"
	"time"
)

// WorkerConfig contains worker configuration
type WorkerConfig struct {
	QueueName     string
	MaxAttempts   int
	BaseBackoffMs int
	PollInterval  time.Duration
}

// ExecutionResult represents the result of job execution
type ExecutionResult struct {
	Success bool
	Error   error
	Output  any
}

var (
	ErrExecutionFailed    = errors.New("job execution failed")
	ErrInvalidConfig      = errors.New("invalid worker configuration")
	ErrQueueNameRequired  = errors.New("queue name is required")
	ErrMaxAttemptsInvalid = errors.New("max attempts must be greater than 0")
)

// NewWorkerConfig creates and validates worker configuration
func NewWorkerConfig(queueName string, maxAttempts, baseBackoffMs int) (*WorkerConfig, error) {
	if queueName == "" {
		return nil, ErrQueueNameRequired
	}
	if maxAttempts <= 0 {
		return nil, ErrMaxAttemptsInvalid
	}

	return &WorkerConfig{
		QueueName:     queueName,
		MaxAttempts:   maxAttempts,
		BaseBackoffMs: baseBackoffMs,
		PollInterval:  5 * time.Second, // Default poll interval
	}, nil
}

// CalculateBackoff calculates exponential backoff duration
func CalculateBackoff(attempt int, baseMs int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	return time.Duration(baseMs*(1<<attempt)) * time.Millisecond
}
