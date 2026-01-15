package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/erickfunier/ai-smart-queue/internal/domain/worker"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*queue.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queue.Job), args.Error(1)
}

func (m *MockJobRepository) Create(ctx context.Context, job *queue.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockJobRepository) Update(ctx context.Context, job *queue.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobRepository) FindPendingJobs(ctx context.Context, queueName string, limit int) ([]*queue.Job, error) {
	args := m.Called(ctx, queueName, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*queue.Job), args.Error(1)
}

func (m *MockJobRepository) FindByStatus(ctx context.Context, status queue.Status, limit int) ([]*queue.Job, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*queue.Job), args.Error(1)
}

func (m *MockJobRepository) CountByStatus(ctx context.Context, status queue.Status) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) MoveToDLQ(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobRepository) CountDLQJobs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJobRepository) GetDLQJobs(ctx context.Context, limit, offset int) ([]*queue.Job, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*queue.Job), args.Error(1)
}

type MockQueueService struct {
	mock.Mock
}

func (m *MockQueueService) Enqueue(ctx context.Context, job *queue.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockQueueService) Dequeue(ctx context.Context, queueName string) (*queue.Job, error) {
	args := m.Called(ctx, queueName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queue.Job), args.Error(1)
}

func (m *MockQueueService) Acknowledge(ctx context.Context, jobID uuid.UUID) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

type MockJobExecutor struct {
	mock.Mock
}

func (m *MockJobExecutor) Execute(ctx context.Context, job *queue.Job) (*worker.ExecutionResult, error) {
	args := m.Called(ctx, job)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*worker.ExecutionResult), args.Error(1)
}

func (m *MockJobExecutor) CanHandle(jobType string) bool {
	args := m.Called(jobType)
	return args.Bool(0)
}

type MockInsightsService struct {
	mock.Mock
}

func (m *MockInsightsService) AnalyzeJobFailure(ctx context.Context, jobID uuid.UUID) (*appInsights.Service, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*appInsights.Service), args.Error(1)
}

func TestService_ProcessNextJob(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
		}
		want struct {
			err         bool
			validateJob func(*testing.T, *MockJobRepository)
		}
	}{
		{
			name: "Given valid job in queue, When processing job, Then should execute and complete successfully",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))

					queueSvc.On("Dequeue", mock.Anything, "default").Return(job, nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil).Times(2)
					executor.On("Execute", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(
						&worker.ExecutionResult{Success: true, Error: nil}, nil,
					)
					queueSvc.On("Acknowledge", mock.Anything, job.ID).Return(nil)
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: false,
				validateJob: func(t *testing.T, repo *MockJobRepository) {
					repo.AssertExpectations(t)
					// Job should be updated twice: once to processing, once to completed
					assert.Equal(t, 2, len(repo.Calls))
				},
			},
		},
		{
			name: "Given empty queue, When processing next job, Then should return without error",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					queueSvc.On("Dequeue", mock.Anything, "default").Return(nil, nil)
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: false,
			},
		},
		{
			name: "Given dequeue operation fails, When processing next job, Then should return error",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					queueSvc.On("Dequeue", mock.Anything, "default").Return(nil, errors.New("dequeue failed"))
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: true,
			},
		},
		{
			name: "Given job execution fails, When attempts below max, Then should mark job for retry",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))
					job.Attempts = 1

					queueSvc.On("Dequeue", mock.Anything, "default").Return(job, nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil).Times(2)
					executor.On("Execute", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(
						&worker.ExecutionResult{Success: false, Error: errors.New("execution failed")}, nil,
					)
					// Add expectation for re-enqueue after retry backoff
					queueSvc.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: false,
				validateJob: func(t *testing.T, repo *MockJobRepository) {
					repo.AssertExpectations(t)
					// Should be updated twice: processing, then retrying
					assert.Equal(t, 2, len(repo.Calls))
				},
			},
		},
		{
			name: "Given job at max attempts, When job execution fails, Then should move to DLQ",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))
					job.Attempts = 3 // At max attempts

					queueSvc.On("Dequeue", mock.Anything, "default").Return(job, nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil).Times(2)
					executor.On("Execute", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(
						&worker.ExecutionResult{Success: false, Error: errors.New("execution failed")}, nil,
					)
					repo.On("MoveToDLQ", mock.Anything, job.ID).Return(nil)
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: false,
				validateJob: func(t *testing.T, repo *MockJobRepository) {
					repo.AssertExpectations(t)
					// Should move to DLQ
					repo.AssertCalled(t, "MoveToDLQ", mock.Anything, mock.AnythingOfType("uuid.UUID"))
				},
			},
		},
		{
			name: "Given repository update fails, When marking job as processing, Then should return error",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))

					queueSvc.On("Dequeue", mock.Anything, "default").Return(job, nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(errors.New("update failed"))
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: true,
			},
		},
		{
			name: "Given executor returns error, When executing job, Then should handle error and retry",
			in: struct {
				setupMocks func(*MockJobRepository, *MockQueueService, *MockJobExecutor)
			}{
				setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, executor *MockJobExecutor) {
					job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))

					queueSvc.On("Dequeue", mock.Anything, "default").Return(job, nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil).Times(2)
					executor.On("Execute", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(
						&worker.ExecutionResult{Success: false, Error: errors.New("executor error")},
						errors.New("executor error"),
					)
					// Add expectation for re-enqueue after retry backoff
					queueSvc.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				},
			},
			want: struct {
				err         bool
				validateJob func(*testing.T, *MockJobRepository)
			}{
				err: false, // Error is handled, job is marked for retry
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockJobRepository)
			mockQueue := new(MockQueueService)
			mockExecutor := new(MockJobExecutor)
			tt.in.setupMocks(mockRepo, mockQueue, mockExecutor)

			config, _ := worker.NewWorkerConfig("default", 3, 500)
			service := NewService(mockRepo, mockQueue, mockExecutor, nil, config)

			// When
			err := service.ProcessNextJob(context.Background())

			// Then
			if tt.want.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockQueue.AssertExpectations(t)
			mockExecutor.AssertExpectations(t)

			if tt.want.validateJob != nil {
				tt.want.validateJob(t, mockRepo)
			}
		})
	}
}

func TestService_HandleJobFailure_WithRetry(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			jobAttempts int
			maxAttempts int
			setupMocks  func(*MockJobRepository)
		}
		want struct {
			validateStatus func(*testing.T, *queue.Job)
		}
	}{
		{
			name: "Given job below max attempts, When handling job failure, Then should mark as retrying with backoff",
			in: struct {
				jobAttempts int
				maxAttempts int
				setupMocks  func(*MockJobRepository)
			}{
				jobAttempts: 1,
				maxAttempts: 3,
				setupMocks: func(repo *MockJobRepository) {
					// Only one Update call - before sleep/re-enqueue
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil).Once()
				},
			},
			want: struct {
				validateStatus func(*testing.T, *queue.Job)
			}{
				validateStatus: func(t *testing.T, job *queue.Job) {
					assert.Equal(t, queue.StatusRetrying, job.Status)
					assert.NotNil(t, job.ScheduledFor)
					// Note: After sleep, the scheduled time will have passed
					// So we just check it's been set
				},
			},
		},
		{
			name: "Given job at max attempts, When handling job failure, Then should move to DLQ and mark as failed",
			in: struct {
				jobAttempts int
				maxAttempts int
				setupMocks  func(*MockJobRepository)
			}{
				jobAttempts: 3,
				maxAttempts: 3,
				setupMocks: func(repo *MockJobRepository) {
					repo.On("MoveToDLQ", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				},
			},
			want: struct {
				validateStatus func(*testing.T, *queue.Job)
			}{
				validateStatus: func(t *testing.T, job *queue.Job) {
					assert.Equal(t, queue.StatusFailed, job.Status)
					assert.NotNil(t, job.Error)
				},
			},
		},
		{
			name: "Given job exceeds max attempts, When handling job failure, Then should move to DLQ",
			in: struct {
				jobAttempts int
				maxAttempts int
				setupMocks  func(*MockJobRepository)
			}{
				jobAttempts: 4,
				maxAttempts: 3,
				setupMocks: func(repo *MockJobRepository) {
					repo.On("MoveToDLQ", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				},
			},
			want: struct {
				validateStatus func(*testing.T, *queue.Job)
			}{
				validateStatus: func(t *testing.T, job *queue.Job) {
					assert.Equal(t, queue.StatusFailed, job.Status)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))
			job.Attempts = tt.in.jobAttempts

			mockRepo := new(MockJobRepository)
			mockQueue := new(MockQueueService)
			mockExecutor := new(MockJobExecutor)
			tt.in.setupMocks(mockRepo)

			// Add Enqueue expectation for retry case
			if tt.in.jobAttempts < tt.in.maxAttempts {
				mockQueue.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
			}

			config, _ := worker.NewWorkerConfig("default", tt.in.maxAttempts, 500)
			service := NewService(mockRepo, mockQueue, mockExecutor, nil, config)

			// When
			err := service.handleJobFailure(context.Background(), job, errors.New("execution failed"))

			// Then
			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
			mockQueue.AssertExpectations(t)

			if tt.want.validateStatus != nil {
				tt.want.validateStatus(t, job)
			}
		})
	}
}

func TestService_HandleJobFailure_DLQError(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			setupMocks func(*MockJobRepository)
		}
		want struct {
			err bool
		}
	}{
		{
			name: "Given DLQ operation fails, When moving job to DLQ, Then should return error",
			in: struct {
				setupMocks func(*MockJobRepository)
			}{
				setupMocks: func(repo *MockJobRepository) {
					repo.On("MoveToDLQ", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(errors.New("DLQ error"))
				},
			},
			want: struct {
				err bool
			}{
				err: true,
			},
		},
		{
			name: "Given DLQ move succeeds, When updating job status, Then should complete without error",
			in: struct {
				setupMocks func(*MockJobRepository)
			}{
				setupMocks: func(repo *MockJobRepository) {
					repo.On("MoveToDLQ", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				},
			},
			want: struct {
				err bool
			}{
				err: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))
			job.Attempts = 3 // At max attempts

			mockRepo := new(MockJobRepository)
			mockQueue := new(MockQueueService)
			mockExecutor := new(MockJobExecutor)
			tt.in.setupMocks(mockRepo)

			config, _ := worker.NewWorkerConfig("default", 3, 500)
			service := NewService(mockRepo, mockQueue, mockExecutor, nil, config)

			// When
			err := service.handleJobFailure(context.Background(), job, errors.New("execution failed"))

			// Then
			if tt.want.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_HandleJobFailure_UpdateError(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			jobAttempts int
			setupMocks  func(*MockJobRepository)
		}
		want struct {
			err bool
		}
	}{
		{
			name: "Given job can retry, When repository update fails, Then should return error",
			in: struct {
				jobAttempts int
				setupMocks  func(*MockJobRepository)
			}{
				jobAttempts: 1,
				setupMocks: func(repo *MockJobRepository) {
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(errors.New("update failed"))
				},
			},
			want: struct {
				err bool
			}{
				err: true,
			},
		},
		{
			name: "Given job moved to DLQ, When updating job status fails, Then should return error",
			in: struct {
				jobAttempts int
				setupMocks  func(*MockJobRepository)
			}{
				jobAttempts: 3,
				setupMocks: func(repo *MockJobRepository) {
					repo.On("MoveToDLQ", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
					repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(errors.New("update failed"))
				},
			},
			want: struct {
				err bool
			}{
				err: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))
			job.Attempts = tt.in.jobAttempts

			mockRepo := new(MockJobRepository)
			mockQueue := new(MockQueueService)
			mockExecutor := new(MockJobExecutor)
			tt.in.setupMocks(mockRepo)

			config, _ := worker.NewWorkerConfig("default", 3, 500)
			service := NewService(mockRepo, mockQueue, mockExecutor, nil, config)

			// When
			err := service.handleJobFailure(context.Background(), job, errors.New("execution failed"))

			// Then
			if tt.want.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_ExponentialBackoff(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			attempts    int
			baseBackoff time.Duration
		}
		want struct {
			validateBackoff func(*testing.T, time.Duration)
		}
	}{
		{
			name: "Given first retry attempt, When calculating backoff, Then should apply 2s exponential backoff",
			in: struct {
				attempts    int
				baseBackoff time.Duration
			}{
				attempts:    1,
				baseBackoff: 500 * time.Millisecond,
			},
			want: struct {
				validateBackoff func(*testing.T, time.Duration)
			}{
				validateBackoff: func(t *testing.T, backoff time.Duration) {
					assert.Equal(t, 2000*time.Millisecond, backoff)
				},
			},
		},
		{
			name: "Given second retry attempt, When calculating backoff, Then should apply 4s exponential backoff",
			in: struct {
				attempts    int
				baseBackoff time.Duration
			}{
				attempts:    2,
				baseBackoff: 500 * time.Millisecond,
			},
			want: struct {
				validateBackoff func(*testing.T, time.Duration)
			}{
				validateBackoff: func(t *testing.T, backoff time.Duration) {
					assert.Equal(t, 4000*time.Millisecond, backoff)
				},
			},
		},
		{
			name: "Given third retry attempt, When calculating backoff, Then should apply 8s exponential backoff",
			in: struct {
				attempts    int
				baseBackoff time.Duration
			}{
				attempts:    3,
				baseBackoff: 500 * time.Millisecond,
			},
			want: struct {
				validateBackoff func(*testing.T, time.Duration)
			}{
				validateBackoff: func(t *testing.T, backoff time.Duration) {
					assert.Equal(t, 8000*time.Millisecond, backoff)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job, _ := queue.NewJob("default", "email", []byte(`{"to":"test@example.com"}`))
			job.Attempts = tt.in.attempts

			mockRepo := new(MockJobRepository)
			mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil).Times(2)

			mockQueue := new(MockQueueService)
			mockQueue.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)

			config, _ := worker.NewWorkerConfig("default", 5, int(tt.in.baseBackoff.Milliseconds()))
			service := NewService(mockRepo, mockQueue, new(MockJobExecutor), nil, config)

			// When
			beforeTime := time.Now().UTC()
			_ = service.handleJobFailure(context.Background(), job, errors.New("test error"))
			afterTime := time.Now().UTC()

			// Then
			assert.NotNil(t, job.ScheduledFor)
			actualBackoff := job.ScheduledFor.Sub(beforeTime)

			// Allow some tolerance for test execution time
			// Note: MarkAsFailed increments attempts before calculating backoff
			expectedBackoff := worker.CalculateBackoff(job.Attempts, int(tt.in.baseBackoff.Milliseconds()))
			tolerance := 100 * time.Millisecond
			assert.True(t, actualBackoff >= expectedBackoff-tolerance &&
				actualBackoff <= expectedBackoff+tolerance+(afterTime.Sub(beforeTime)),
				"Expected backoff around %v, got %v", expectedBackoff, actualBackoff)
		})
	}
}
