package queue

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Job represents the core job entity in the domain
type Job struct {
	ID           uuid.UUID
	Queue        string
	Type         string
	Status       Status
	Attempts     int
	Payload      []byte
	Error        string
	ScheduledFor *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Status represents job processing status
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusRetrying   Status = "retrying"
)

// Business rules and validation

var (
	ErrInvalidQueue       = errors.New("queue name is required")
	ErrInvalidType        = errors.New("job type is required")
	ErrMaxAttemptsReached = errors.New("maximum retry attempts reached")
	ErrJobNotFound        = errors.New("job not found")
)

// NewJob creates a new job with validation
func NewJob(queue, jobType string, payload []byte) (*Job, error) {
	if queue == "" {
		return nil, ErrInvalidQueue
	}
	if jobType == "" {
		return nil, ErrInvalidType
	}

	now := time.Now().UTC()
	return &Job{
		ID:        uuid.New(),
		Queue:     queue,
		Type:      jobType,
		Status:    StatusPending,
		Attempts:  0,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// CanRetry checks if the job can be retried based on business rules
func (j *Job) CanRetry(maxAttempts int) bool {
	return j.Attempts < maxAttempts && j.Status == StatusFailed
}

// MarkAsProcessing marks the job as being processed
func (j *Job) MarkAsProcessing() {
	j.Status = StatusProcessing
	j.UpdatedAt = time.Now().UTC()
}

// MarkAsCompleted marks the job as successfully completed
func (j *Job) MarkAsCompleted() {
	j.Status = StatusCompleted
	j.UpdatedAt = time.Now().UTC()
}

// MarkAsFailed marks the job as failed with an error message
func (j *Job) MarkAsFailed(err error) {
	j.Status = StatusFailed
	j.Error = err.Error()
	j.Attempts++
	j.UpdatedAt = time.Now().UTC()
}

// MarkAsRetrying marks the job for retry
func (j *Job) MarkAsRetrying() {
	j.Status = StatusRetrying
	j.UpdatedAt = time.Now().UTC()
}

// Schedule schedules the job for future execution
func (j *Job) Schedule(scheduledFor time.Time) {
	j.ScheduledFor = &scheduledFor
	j.UpdatedAt = time.Now().UTC()
}

// IsReady checks if the job is ready to be processed
func (j *Job) IsReady() bool {
	if j.Status != StatusPending && j.Status != StatusRetrying {
		return false
	}
	if j.ScheduledFor != nil && j.ScheduledFor.After(time.Now().UTC()) {
		return false
	}
	return true
}
