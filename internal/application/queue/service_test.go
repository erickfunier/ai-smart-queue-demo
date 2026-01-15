package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockJobRepository struct {
	mock.Mock
}

func (m *MockJobRepository) Create(ctx context.Context, job *queue.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*queue.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*queue.Job), args.Error(1)
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

func (m *MockJobRepository) GetDLQJobs(ctx context.Context, limit, offset int) ([]*queue.Job, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*queue.Job), args.Error(1)
}

func (m *MockJobRepository) MoveToDLQ(ctx context.Context, jobID uuid.UUID) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobRepository) CountDLQJobs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
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

type MockMetricsService struct {
	mock.Mock
}

func (m *MockMetricsService) RecordJobCreated(queueName, jobType string) {
	m.Called(queueName, jobType)
}

func (m *MockMetricsService) RecordJobCompleted(queueName, jobType string, duration float64) {
	m.Called(queueName, jobType, duration)
}

func (m *MockMetricsService) RecordJobFailed(queueName, jobType string) {
	m.Called(queueName, jobType)
}

func (m *MockMetricsService) RecordJobRetried(queueName, jobType string) {
	m.Called(queueName, jobType)
}

func TestService_CreateJob(t *testing.T) {
	tests := []struct {
		name        string
		given       string
		when        string
		then        string
		command     CreateJobCommand
		setupMocks  func(*MockJobRepository, *MockQueueService, *MockMetricsService)
		expectErr   bool
		validateJob func(*testing.T, *queue.Job)
	}{
		{
			name:  "Successful job creation",
			given: "valid job command with queue, type and payload",
			when:  "creating a new job",
			then:  "should create job, enqueue it and record metrics",
			command: CreateJobCommand{
				Queue:   "default",
				Type:    "email",
				Payload: map[string]any{"to": "test@example.com"},
			},
			setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, metrics *MockMetricsService) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				queueSvc.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				metrics.On("RecordJobCreated", "default", "email").Return()
			},
			expectErr: false,
			validateJob: func(t *testing.T, job *queue.Job) {
				assert.NotEqual(t, uuid.Nil, job.ID)
				assert.Equal(t, "default", job.Queue)
				assert.Equal(t, "email", job.Type)
				assert.Equal(t, queue.StatusPending, job.Status)
			},
		},
		{
			name:  "Empty queue name",
			given: "command with empty queue name",
			when:  "creating a new job",
			then:  "should return validation error",
			command: CreateJobCommand{
				Queue:   "",
				Type:    "email",
				Payload: map[string]any{},
			},
			setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, metrics *MockMetricsService) {
				// No mocks needed as validation fails before repo call
			},
			expectErr: true,
		},
		{
			name:  "Repository error",
			given: "valid command but repository fails",
			when:  "creating a new job",
			then:  "should return repository error",
			command: CreateJobCommand{
				Queue:   "default",
				Type:    "email",
				Payload: map[string]any{},
			},
			setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, metrics *MockMetricsService) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*queue.Job")).
					Return(errors.New("database error"))
			},
			expectErr: true,
		},
		{
			name:  "Queue service error",
			given: "valid command but queue service fails",
			when:  "creating a new job",
			then:  "should return queue service error",
			command: CreateJobCommand{
				Queue:   "default",
				Type:    "email",
				Payload: map[string]any{},
			},
			setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, metrics *MockMetricsService) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				queueSvc.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).
					Return(errors.New("redis error"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockJobRepository)
			mockQueueSvc := new(MockQueueService)
			mockMetrics := new(MockMetricsService)
			tt.setupMocks(mockRepo, mockQueueSvc, mockMetrics)

			service := NewService(mockRepo, mockQueueSvc, mockMetrics)
			ctx := context.Background()

			// When
			job, err := service.CreateJob(ctx, tt.command)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				if tt.validateJob != nil {
					tt.validateJob(t, job)
				}
			}

			mockRepo.AssertExpectations(t)
			mockQueueSvc.AssertExpectations(t)
			mockMetrics.AssertExpectations(t)
		})
	}
}

func TestService_GetJob(t *testing.T) {
	jobID := uuid.New()

	tests := []struct {
		name       string
		given      string
		when       string
		then       string
		id         uuid.UUID
		setupMocks func(*MockJobRepository)
		expectErr  bool
	}{
		{
			name:  "Job found",
			given: "existing job ID",
			when:  "getting job by ID",
			then:  "should return the job",
			id:    jobID,
			setupMocks: func(repo *MockJobRepository) {
				job := &queue.Job{
					ID:     jobID,
					Queue:  "default",
					Type:   "email",
					Status: queue.StatusPending,
				}
				repo.On("GetByID", mock.Anything, jobID).Return(job, nil)
			},
			expectErr: false,
		},
		{
			name:  "Job not found",
			given: "non-existing job ID",
			when:  "getting job by ID",
			then:  "should return error",
			id:    jobID,
			setupMocks: func(repo *MockJobRepository) {
				repo.On("GetByID", mock.Anything, jobID).Return(nil, errors.New("not found"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockJobRepository)
			mockQueueSvc := new(MockQueueService)
			mockMetrics := new(MockMetricsService)
			tt.setupMocks(mockRepo)

			service := NewService(mockRepo, mockQueueSvc, mockMetrics)
			ctx := context.Background()

			// When
			job, err := service.GetJob(ctx, tt.id)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, job)
				assert.Equal(t, tt.id, job.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_RetryJob(t *testing.T) {
	jobID := uuid.New()

	tests := []struct {
		name        string
		given       string
		when        string
		then        string
		maxAttempts int
		setupMocks  func(*MockJobRepository, *MockQueueService, *MockMetricsService)
		expectErr   bool
	}{
		{
			name:        "Retry eligible failed job",
			given:       "failed job with 2 attempts and max 3",
			when:        "retrying the job",
			then:        "should mark as retrying, update and re-enqueue",
			maxAttempts: 3,
			setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, metrics *MockMetricsService) {
				job := &queue.Job{
					ID:       jobID,
					Queue:    "default",
					Type:     "email",
					Status:   queue.StatusFailed,
					Attempts: 2,
				}
				repo.On("GetByID", mock.Anything, jobID).Return(job, nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				queueSvc.On("Enqueue", mock.Anything, mock.AnythingOfType("*queue.Job")).Return(nil)
				metrics.On("RecordJobRetried", "default", "email").Return()
			},
			expectErr: false,
		},
		{
			name:        "Max attempts reached",
			given:       "failed job with 3 attempts and max 3",
			when:        "retrying the job",
			then:        "should return ErrMaxAttemptsReached",
			maxAttempts: 3,
			setupMocks: func(repo *MockJobRepository, queueSvc *MockQueueService, metrics *MockMetricsService) {
				job := &queue.Job{
					ID:       jobID,
					Status:   queue.StatusFailed,
					Attempts: 3,
				}
				repo.On("GetByID", mock.Anything, jobID).Return(job, nil)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockJobRepository)
			mockQueueSvc := new(MockQueueService)
			mockMetrics := new(MockMetricsService)
			tt.setupMocks(mockRepo, mockQueueSvc, mockMetrics)

			service := NewService(mockRepo, mockQueueSvc, mockMetrics)
			ctx := context.Background()

			// When
			err := service.RetryJob(ctx, jobID, tt.maxAttempts)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockQueueSvc.AssertExpectations(t)
			mockMetrics.AssertExpectations(t)
		})
	}
}
