package insights

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erickfunier/ai-smart-queue/internal/domain/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockInsightRepository struct {
	mock.Mock
}

func (m *MockInsightRepository) Create(ctx context.Context, insight *insights.Insight) error {
	args := m.Called(ctx, insight)
	return args.Error(0)
}

func (m *MockInsightRepository) GetByID(ctx context.Context, id uuid.UUID) (*insights.Insight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*insights.Insight), args.Error(1)
}

func (m *MockInsightRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) (*insights.Insight, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*insights.Insight), args.Error(1)
}

func (m *MockInsightRepository) List(ctx context.Context, limit, offset int) ([]*insights.Insight, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*insights.Insight), args.Error(1)
}

func (m *MockInsightRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

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

type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) Analyze(ctx context.Context, request *insights.AnalysisRequest) (*insights.AnalysisResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*insights.AnalysisResponse), args.Error(1)
}

func TestService_AnalyzeJobFailure(t *testing.T) {
	tests := []struct {
		name            string
		given           string
		when            string
		then            string
		jobID           uuid.UUID
		setupMocks      func(*MockInsightRepository, *MockJobRepository, *MockAIService, uuid.UUID)
		expectErr       bool
		validateInsight func(*testing.T, *insights.Insight)
	}{
		{
			name:  "Return cached insight without calling AI",
			given: "existing insight for job ID",
			when:  "analyzing job failure",
			then:  "should return cached insight without calling AI service",
			jobID: uuid.New(),
			setupMocks: func(insightRepo *MockInsightRepository, jobRepo *MockJobRepository, aiSvc *MockAIService, jobID uuid.UUID) {
				cachedInsight := &insights.Insight{
					ID:             uuid.New(),
					JobID:          jobID,
					Diagnosis:      "Cached network connectivity issue",
					Recommendation: "Cached recommendation to increase timeout",
					SuggestedFix: insights.SuggestedFix{
						TimeoutSeconds: 30,
						MaxRetries:     5,
						PayloadPatch:   map[string]any{"timeout": 30},
					},
					CreatedAt: time.Now().UTC(),
				}
				insightRepo.On("GetByJobID", mock.Anything, jobID).Return(cachedInsight, nil)
				// AI service and job repo should NOT be called
			},
			expectErr: false,
			validateInsight: func(t *testing.T, insight *insights.Insight) {
				assert.NotEqual(t, uuid.Nil, insight.ID)
				assert.Equal(t, "Cached network connectivity issue", insight.Diagnosis)
				assert.Equal(t, "Cached recommendation to increase timeout", insight.Recommendation)
				assert.Equal(t, 30, insight.SuggestedFix.TimeoutSeconds)
				assert.Equal(t, 5, insight.SuggestedFix.MaxRetries)
			},
		},
		{
			name:  "Successfully analyze failed job when no cache exists",
			given: "valid failed job with error information and no cached insight",
			when:  "analyzing job failure",
			then:  "should create insight with AI analysis",
			jobID: uuid.New(),
			setupMocks: func(insightRepo *MockInsightRepository, jobRepo *MockJobRepository, aiSvc *MockAIService, jobID uuid.UUID) {
				// No cached insight
				insightRepo.On("GetByJobID", mock.Anything, jobID).Return(nil, errors.New("not found"))

				failedJob := &queue.Job{
					ID:        jobID,
					Queue:     "default",
					Type:      "email",
					Status:    queue.StatusFailed,
					Error:     "Connection timeout after 10s",
					Payload:   []byte(`{"to":"test@example.com","subject":"Hello"}`),
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				jobRepo.On("GetByID", mock.Anything, jobID).Return(failedJob, nil)

				aiResponse := &insights.AnalysisResponse{
					Diagnosis:      "Network connectivity issue causing timeout",
					Recommendation: "Increase connection timeout to 30 seconds",
					SuggestedFix: insights.SuggestedFix{
						TimeoutSeconds: 30,
						MaxRetries:     5,
						PayloadPatch:   map[string]any{"timeout": 30},
					},
				}
				aiSvc.On("Analyze", mock.Anything, mock.AnythingOfType("*insights.AnalysisRequest")).
					Return(aiResponse, nil)

				insightRepo.On("Create", mock.Anything, mock.AnythingOfType("*insights.Insight")).
					Return(nil)
			},
			expectErr: false,
			validateInsight: func(t *testing.T, insight *insights.Insight) {
				assert.NotEqual(t, uuid.Nil, insight.ID)
				assert.Equal(t, "Network connectivity issue causing timeout", insight.Diagnosis)
				assert.Equal(t, "Increase connection timeout to 30 seconds", insight.Recommendation)
				assert.Equal(t, 30, insight.SuggestedFix.TimeoutSeconds)
				assert.Equal(t, 5, insight.SuggestedFix.MaxRetries)
			},
		},
		{
			name:  "Job not found",
			given: "non-existent job ID",
			when:  "analyzing job failure",
			then:  "should return job not found error",
			jobID: uuid.New(),
			setupMocks: func(insightRepo *MockInsightRepository, jobRepo *MockJobRepository, aiSvc *MockAIService, jobID uuid.UUID) {
				insightRepo.On("GetByJobID", mock.Anything, jobID).Return(nil, errors.New("not found"))
				jobRepo.On("GetByID", mock.Anything, jobID).
					Return(nil, errors.New("job not found"))
			},
			expectErr: true,
		},
		{
			name:  "AI service error",
			given: "valid job but AI service fails",
			when:  "analyzing job failure",
			then:  "should return AI service error",
			jobID: uuid.New(),
			setupMocks: func(insightRepo *MockInsightRepository, jobRepo *MockJobRepository, aiSvc *MockAIService, jobID uuid.UUID) {
				insightRepo.On("GetByJobID", mock.Anything, jobID).Return(nil, errors.New("not found"))

				failedJob := &queue.Job{
					ID:        jobID,
					Error:     "Test error",
					Payload:   []byte(`{}`),
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				jobRepo.On("GetByID", mock.Anything, jobID).Return(failedJob, nil)

				aiSvc.On("Analyze", mock.Anything, mock.AnythingOfType("*insights.AnalysisRequest")).
					Return(nil, insights.ErrAnalysisFailed)
			},
			expectErr: true,
		},
		{
			name:  "Repository create error",
			given: "valid analysis but repository fails to persist",
			when:  "analyzing job failure",
			then:  "should return repository error",
			jobID: uuid.New(),
			setupMocks: func(insightRepo *MockInsightRepository, jobRepo *MockJobRepository, aiSvc *MockAIService, jobID uuid.UUID) {
				insightRepo.On("GetByJobID", mock.Anything, jobID).Return(nil, errors.New("not found"))

				failedJob := &queue.Job{
					ID:        jobID,
					Error:     "Test error",
					Payload:   []byte(`{}`),
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				jobRepo.On("GetByID", mock.Anything, jobID).Return(failedJob, nil)

				aiResponse := &insights.AnalysisResponse{
					Diagnosis:      "Test diagnosis",
					Recommendation: "Test recommendation",
					SuggestedFix:   insights.SuggestedFix{},
				}
				aiSvc.On("Analyze", mock.Anything, mock.AnythingOfType("*insights.AnalysisRequest")).
					Return(aiResponse, nil)

				insightRepo.On("Create", mock.Anything, mock.AnythingOfType("*insights.Insight")).
					Return(errors.New("database error"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			insightRepo := new(MockInsightRepository)
			jobRepo := new(MockJobRepository)
			aiService := new(MockAIService)

			tt.setupMocks(insightRepo, jobRepo, aiService, tt.jobID)

			service := NewService(insightRepo, jobRepo, aiService)
			ctx := context.Background()

			// When
			insight, err := service.AnalyzeJobFailure(ctx, tt.jobID)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, insight)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, insight)
				if tt.validateInsight != nil {
					tt.validateInsight(t, insight)
				}
			}

			insightRepo.AssertExpectations(t)
			jobRepo.AssertExpectations(t)
			aiService.AssertExpectations(t)
		})
	}
}

func TestService_GetInsight(t *testing.T) {
	tests := []struct {
		name            string
		given           string
		when            string
		then            string
		insightID       uuid.UUID
		setupMocks      func(*MockInsightRepository, uuid.UUID)
		expectErr       bool
		validateInsight func(*testing.T, *insights.Insight)
	}{
		{
			name:      "Successfully get insight by ID",
			given:     "valid insight ID",
			when:      "retrieving insight",
			then:      "should return insight",
			insightID: uuid.New(),
			setupMocks: func(repo *MockInsightRepository, id uuid.UUID) {
				insight := &insights.Insight{
					ID:             id,
					JobID:          uuid.New(),
					Diagnosis:      "Test diagnosis",
					Recommendation: "Test recommendation",
					CreatedAt:      time.Now().UTC(),
				}
				repo.On("GetByID", mock.Anything, id).Return(insight, nil)
			},
			expectErr: false,
			validateInsight: func(t *testing.T, insight *insights.Insight) {
				assert.Equal(t, "Test diagnosis", insight.Diagnosis)
			},
		},
		{
			name:      "Insight not found",
			given:     "non-existent insight ID",
			when:      "retrieving insight",
			then:      "should return not found error",
			insightID: uuid.New(),
			setupMocks: func(repo *MockInsightRepository, id uuid.UUID) {
				repo.On("GetByID", mock.Anything, id).
					Return(nil, insights.ErrInsightNotFound)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			insightRepo := new(MockInsightRepository)
			tt.setupMocks(insightRepo, tt.insightID)

			service := NewService(insightRepo, new(MockJobRepository), new(MockAIService))
			ctx := context.Background()

			// When
			insight, err := service.GetInsight(ctx, tt.insightID)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, insight)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, insight)
				if tt.validateInsight != nil {
					tt.validateInsight(t, insight)
				}
			}

			insightRepo.AssertExpectations(t)
		})
	}
}

func TestService_GetInsightByJobID(t *testing.T) {
	tests := []struct {
		name            string
		given           string
		when            string
		then            string
		jobID           uuid.UUID
		setupMocks      func(*MockInsightRepository, uuid.UUID)
		expectErr       bool
		validateInsight func(*testing.T, *insights.Insight)
	}{
		{
			name:  "Successfully get insight by job ID",
			given: "valid job ID with existing insight",
			when:  "retrieving insight by job ID",
			then:  "should return insight",
			jobID: uuid.New(),
			setupMocks: func(repo *MockInsightRepository, jobID uuid.UUID) {
				insight := &insights.Insight{
					ID:             uuid.New(),
					JobID:          jobID,
					Diagnosis:      "Job-specific diagnosis",
					Recommendation: "Job-specific recommendation",
					CreatedAt:      time.Now().UTC(),
				}
				repo.On("GetByJobID", mock.Anything, jobID).Return(insight, nil)
			},
			expectErr: false,
			validateInsight: func(t *testing.T, insight *insights.Insight) {
				assert.Equal(t, "Job-specific diagnosis", insight.Diagnosis)
			},
		},
		{
			name:  "Insight not found for job",
			given: "job ID without insight",
			when:  "retrieving insight by job ID",
			then:  "should return not found error",
			jobID: uuid.New(),
			setupMocks: func(repo *MockInsightRepository, jobID uuid.UUID) {
				repo.On("GetByJobID", mock.Anything, jobID).
					Return(nil, insights.ErrInsightNotFound)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			insightRepo := new(MockInsightRepository)
			tt.setupMocks(insightRepo, tt.jobID)

			service := NewService(insightRepo, new(MockJobRepository), new(MockAIService))
			ctx := context.Background()

			// When
			insight, err := service.GetInsightByJobID(ctx, tt.jobID)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, insight)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, insight)
				if tt.validateInsight != nil {
					tt.validateInsight(t, insight)
				}
			}

			insightRepo.AssertExpectations(t)
		})
	}
}

func TestService_ListInsights(t *testing.T) {
	tests := []struct {
		name         string
		given        string
		when         string
		then         string
		limit        int
		offset       int
		setupMocks   func(*MockInsightRepository, int, int)
		expectErr    bool
		validateList func(*testing.T, []*insights.Insight)
	}{
		{
			name:   "Successfully list insights with default pagination",
			given:  "multiple insights in repository",
			when:   "listing insights with limit 50 and offset 0",
			then:   "should return list of insights",
			limit:  50,
			offset: 0,
			setupMocks: func(repo *MockInsightRepository, limit, offset int) {
				insightsList := []*insights.Insight{
					{ID: uuid.New(), JobID: uuid.New(), Diagnosis: "Diagnosis 1", CreatedAt: time.Now().UTC()},
					{ID: uuid.New(), JobID: uuid.New(), Diagnosis: "Diagnosis 2", CreatedAt: time.Now().UTC()},
					{ID: uuid.New(), JobID: uuid.New(), Diagnosis: "Diagnosis 3", CreatedAt: time.Now().UTC()},
				}
				repo.On("List", mock.Anything, limit, offset).Return(insightsList, nil)
			},
			expectErr: false,
			validateList: func(t *testing.T, list []*insights.Insight) {
				assert.Equal(t, 3, len(list))
			},
		},
		{
			name:   "List with custom pagination",
			given:  "custom limit and offset",
			when:   "listing insights with limit 10 and offset 5",
			then:   "should return paginated results",
			limit:  10,
			offset: 5,
			setupMocks: func(repo *MockInsightRepository, limit, offset int) {
				insightsList := []*insights.Insight{
					{ID: uuid.New(), JobID: uuid.New(), Diagnosis: "Paginated insight", CreatedAt: time.Now().UTC()},
				}
				repo.On("List", mock.Anything, limit, offset).Return(insightsList, nil)
			},
			expectErr: false,
			validateList: func(t *testing.T, list []*insights.Insight) {
				assert.Equal(t, 1, len(list))
			},
		},
		{
			name:   "Empty list when no insights exist",
			given:  "empty repository",
			when:   "listing insights",
			then:   "should return empty list",
			limit:  50,
			offset: 0,
			setupMocks: func(repo *MockInsightRepository, limit, offset int) {
				repo.On("List", mock.Anything, limit, offset).Return([]*insights.Insight{}, nil)
			},
			expectErr: false,
			validateList: func(t *testing.T, list []*insights.Insight) {
				assert.Equal(t, 0, len(list))
			},
		},
		{
			name:   "Repository error",
			given:  "repository error occurs",
			when:   "listing insights",
			then:   "should return error",
			limit:  50,
			offset: 0,
			setupMocks: func(repo *MockInsightRepository, limit, offset int) {
				repo.On("List", mock.Anything, limit, offset).
					Return(nil, errors.New("database error"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			insightRepo := new(MockInsightRepository)
			tt.setupMocks(insightRepo, tt.limit, tt.offset)

			service := NewService(insightRepo, new(MockJobRepository), new(MockAIService))
			ctx := context.Background()

			// When
			list, err := service.ListInsights(ctx, tt.limit, tt.offset)

			// Then
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, list)
				if tt.validateList != nil {
					tt.validateList(t, list)
				}
			}

			insightRepo.AssertExpectations(t)
		})
	}
}
