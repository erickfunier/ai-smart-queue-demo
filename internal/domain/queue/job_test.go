package queue

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewJob(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			queue   string
			jobType string
			payload []byte
		}
		want struct {
			err error
		}
	}{
		{
			name: "Given valid queue name type and payload, When creating a new job, Then should create job with pending status",
			in: struct {
				queue   string
				jobType string
				payload []byte
			}{
				queue:   "default",
				jobType: "email",
				payload: []byte(`{"to":"test@example.com"}`),
			},
			want: struct {
				err error
			}{
				err: nil,
			},
		},
		{
			name: "Given empty queue name, When creating a new job, Then should return ErrInvalidQueue",
			in: struct {
				queue   string
				jobType string
				payload []byte
			}{
				queue:   "",
				jobType: "email",
				payload: []byte(`{}`),
			},
			want: struct {
				err error
			}{
				err: ErrInvalidQueue,
			},
		},
		{
			name: "Given empty job type, When creating a new job, Then should return ErrInvalidType",
			in: struct {
				queue   string
				jobType string
				payload []byte
			}{
				queue:   "default",
				jobType: "",
				payload: []byte(`{}`),
			},
			want: struct {
				err error
			}{
				err: ErrInvalidType,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := NewJob(tt.in.queue, tt.in.jobType, tt.in.payload)

			if tt.want.err != nil {
				assert.ErrorIs(t, err, tt.want.err)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				assert.NotEqual(t, uuid.Nil, job.ID)
				assert.Equal(t, tt.in.queue, job.Queue)
				assert.Equal(t, tt.in.jobType, job.Type)
				assert.Equal(t, StatusPending, job.Status)
				assert.Equal(t, 0, job.Attempts)
				assert.Equal(t, tt.in.payload, job.Payload)
				assert.False(t, job.CreatedAt.IsZero())
				assert.False(t, job.UpdatedAt.IsZero())
			}
		})
	}
}

func TestJob_CanRetry(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			status      Status
			attempts    int
			maxAttempts int
		}
		want struct {
			canRetry bool
		}
	}{
		{
			name: "Given failed job with 2 attempts and max 3, When checking retry, Then should return true",
			in: struct {
				status      Status
				attempts    int
				maxAttempts int
			}{
				status:      StatusFailed,
				attempts:    2,
				maxAttempts: 3,
			},
			want: struct {
				canRetry bool
			}{
				canRetry: true,
			},
		},
		{
			name: "Given failed job with 3 attempts and max 3, When checking retry, Then should return false",
			in: struct {
				status      Status
				attempts    int
				maxAttempts int
			}{
				status:      StatusFailed,
				attempts:    3,
				maxAttempts: 3,
			},
			want: struct {
				canRetry bool
			}{
				canRetry: false,
			},
		},
		{
			name: "Given failed job with 4 attempts and max 3, When checking retry, Then should return false",
			in: struct {
				status      Status
				attempts    int
				maxAttempts int
			}{
				status:      StatusFailed,
				attempts:    4,
				maxAttempts: 3,
			},
			want: struct {
				canRetry bool
			}{
				canRetry: false,
			},
		},
		{
			name: "Given completed job, When checking retry, Then should return false",
			in: struct {
				status      Status
				attempts    int
				maxAttempts int
			}{
				status:      StatusCompleted,
				attempts:    1,
				maxAttempts: 3,
			},
			want: struct {
				canRetry bool
			}{
				canRetry: false,
			},
		},
		{
			name: "Given processing job, When checking retry, Then should return false",
			in: struct {
				status      Status
				attempts    int
				maxAttempts int
			}{
				status:      StatusProcessing,
				attempts:    1,
				maxAttempts: 3,
			},
			want: struct {
				canRetry bool
			}{
				canRetry: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Status:   tt.in.status,
				Attempts: tt.in.attempts,
			}

			result := job.CanRetry(tt.in.maxAttempts)

			assert.Equal(t, tt.want.canRetry, result)
		})
	}
}

func TestJob_MarkAsProcessing(t *testing.T) {
	// Given
	job := &Job{
		Status:    StatusPending,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	oldUpdateTime := job.UpdatedAt

	// When
	job.MarkAsProcessing()

	// Then
	assert.Equal(t, StatusProcessing, job.Status)
	assert.True(t, job.UpdatedAt.After(oldUpdateTime))
}

func TestJob_MarkAsCompleted(t *testing.T) {
	// Given
	job := &Job{
		Status:    StatusProcessing,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	oldUpdateTime := job.UpdatedAt

	// When
	job.MarkAsCompleted()

	// Then
	assert.Equal(t, StatusCompleted, job.Status)
	assert.True(t, job.UpdatedAt.After(oldUpdateTime))
}

func TestJob_MarkAsFailed(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			initialAttempts int
			err             error
		}
		want struct {
			attempts int
		}
	}{
		{
			name: "Given job with 0 attempts, When marking as failed, Then should increment to 1 and set error",
			in: struct {
				initialAttempts int
				err             error
			}{
				initialAttempts: 0,
				err:             errors.New("network timeout"),
			},
			want: struct {
				attempts int
			}{
				attempts: 1,
			},
		},
		{
			name: "Given job with 1 attempt, When marking as failed, Then should increment to 2 and set error",
			in: struct {
				initialAttempts int
				err             error
			}{
				initialAttempts: 1,
				err:             errors.New("database error"),
			},
			want: struct {
				attempts int
			}{
				attempts: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Status:    StatusProcessing,
				Attempts:  tt.in.initialAttempts,
				UpdatedAt: time.Now().Add(-1 * time.Hour),
			}
			oldUpdateTime := job.UpdatedAt

			job.MarkAsFailed(tt.in.err)

			assert.Equal(t, StatusFailed, job.Status)
			assert.Equal(t, tt.want.attempts, job.Attempts)
			assert.Equal(t, tt.in.err.Error(), job.Error)
			assert.True(t, job.UpdatedAt.After(oldUpdateTime))
		})
	}
}

func TestJob_MarkAsRetrying(t *testing.T) {
	// Given
	job := &Job{
		Status:    StatusFailed,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	oldUpdateTime := job.UpdatedAt

	// When
	job.MarkAsRetrying()

	// Then
	assert.Equal(t, StatusRetrying, job.Status)
	assert.True(t, job.UpdatedAt.After(oldUpdateTime))
}

func TestJob_Schedule(t *testing.T) {
	// Given
	job := &Job{
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	scheduledFor := time.Now().Add(5 * time.Minute)
	oldUpdateTime := job.UpdatedAt

	// When
	job.Schedule(scheduledFor)

	// Then
	assert.NotNil(t, job.ScheduledFor)
	assert.Equal(t, scheduledFor, *job.ScheduledFor)
	assert.True(t, job.UpdatedAt.After(oldUpdateTime))
}

func TestJob_IsReady(t *testing.T) {
	now := time.Now().UTC()
	pastTime := now.Add(-5 * time.Minute)
	futureTime := now.Add(5 * time.Minute)

	tests := []struct {
		name string
		in   struct {
			status       Status
			scheduledFor *time.Time
		}
		want struct {
			isReady bool
		}
	}{
		{
			name: "Given pending job with no schedule, When checking if ready, Then should return true",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusPending,
				scheduledFor: nil,
			},
			want: struct {
				isReady bool
			}{
				isReady: true,
			},
		},
		{
			name: "Given retrying job with no schedule, When checking if ready, Then should return true",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusRetrying,
				scheduledFor: nil,
			},
			want: struct {
				isReady bool
			}{
				isReady: true,
			},
		},
		{
			name: "Given pending job scheduled for past, When checking if ready, Then should return true",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusPending,
				scheduledFor: &pastTime,
			},
			want: struct {
				isReady bool
			}{
				isReady: true,
			},
		},
		{
			name: "Given pending job scheduled for future, When checking if ready, Then should return false",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusPending,
				scheduledFor: &futureTime,
			},
			want: struct {
				isReady bool
			}{
				isReady: false,
			},
		},
		{
			name: "Given processing job, When checking if ready, Then should return false",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusProcessing,
				scheduledFor: nil,
			},
			want: struct {
				isReady bool
			}{
				isReady: false,
			},
		},
		{
			name: "Given completed job, When checking if ready, Then should return false",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusCompleted,
				scheduledFor: nil,
			},
			want: struct {
				isReady bool
			}{
				isReady: false,
			},
		},
		{
			name: "Given failed job, When checking if ready, Then should return false",
			in: struct {
				status       Status
				scheduledFor *time.Time
			}{
				status:       StatusFailed,
				scheduledFor: nil,
			},
			want: struct {
				isReady bool
			}{
				isReady: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Status:       tt.in.status,
				ScheduledFor: tt.in.scheduledFor,
			}

			result := job.IsReady()

			assert.Equal(t, tt.want.isReady, result)
		})
	}
}
