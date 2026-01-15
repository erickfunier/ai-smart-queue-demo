package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/insights"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInsightsHandlers_GetInsightByID(t *testing.T) {
	// Create a test insight ID that will be shared
	testInsightID := uuid.New()

	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		insightID      uuid.UUID
		setupService   func(uuid.UUID) *appInsights.Service
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "Successfully get insight by ID",
			given:     "valid insight ID in path",
			when:      "GET to /api/insights/{id}",
			then:      "should return 200 with insight details",
			insightID: testInsightID,
			setupService: func(id uuid.UUID) *appInsights.Service {
				testInsight := &insights.Insight{
					ID:             id,
					JobID:          uuid.New(),
					Diagnosis:      "Connection timeout",
					Recommendation: "Increase timeout value",
					SuggestedFix: insights.SuggestedFix{
						TimeoutSeconds: 30,
						MaxRetries:     5,
					},
					CreatedAt: time.Now().UTC(),
				}

				insightRepo := &InMemoryInsightRepo{
					insights: map[uuid.UUID]*insights.Insight{
						id: testInsight,
					},
				}
				jobRepo := &InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)}
				aiService := &MockAIService{}

				return appInsights.NewService(insightRepo, jobRepo, aiService)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp InsightResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, "Connection timeout", resp.Diagnosis)
				assert.Equal(t, "Increase timeout value", resp.Recommendation)
			},
		},
		{
			name:      "Invalid insight ID",
			given:     "invalid UUID in path",
			when:      "GET to /api/insights/{id}",
			then:      "should return 400 bad request",
			insightID: uuid.Nil,
			setupService: func(id uuid.UUID) *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{insights: map[uuid.UUID]*insights.Insight{}},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Insight not found",
			given:     "valid UUID but insight doesn't exist",
			when:      "GET to /api/insights/{id}",
			then:      "should return 404 not found",
			insightID: uuid.New(),
			setupService: func(id uuid.UUID) *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{insights: map[uuid.UUID]*insights.Insight{}},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			service := tt.setupService(tt.insightID)
			handlers := NewInsightsHandlers(service)

			// Build path
			var path string
			if tt.insightID == uuid.Nil {
				path = "/api/insights/invalid-uuid"
			} else {
				path = "/api/insights/" + tt.insightID.String()
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			// When
			handlers.GetInsightByID(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

func TestInsightsHandlers_GetInsightByJobID(t *testing.T) {
	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		jobID          string
		setupService   func(uuid.UUID) *appInsights.Service
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "Successfully get insight by job ID",
			given: "valid job ID with existing insight",
			when:  "GET to /api/insights?job_id={id}",
			then:  "should return 200 with insight details",
			jobID: uuid.New().String(),
			setupService: func(jobID uuid.UUID) *appInsights.Service {
				insightRepo := &InMemoryInsightRepo{
					insights:      map[uuid.UUID]*insights.Insight{},
					insightsByJob: map[uuid.UUID]*insights.Insight{},
				}

				testInsight := &insights.Insight{
					ID:             uuid.New(),
					JobID:          jobID,
					Diagnosis:      "Memory leak detected",
					Recommendation: "Optimize memory usage",
					SuggestedFix: insights.SuggestedFix{
						TimeoutSeconds: 60,
						MaxRetries:     3,
					},
					CreatedAt: time.Now().UTC(),
				}
				insightRepo.insights[testInsight.ID] = testInsight
				insightRepo.insightsByJob[jobID] = testInsight

				return appInsights.NewService(
					insightRepo,
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp InsightResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, "Memory leak detected", resp.Diagnosis)
			},
		},
		{
			name:  "Missing job_id parameter",
			given: "no job_id in query string",
			when:  "GET to /api/insights",
			then:  "should return 400 bad request",
			jobID: "",
			setupService: func(jobID uuid.UUID) *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{insights: map[uuid.UUID]*insights.Insight{}},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "Invalid job_id format",
			given: "invalid UUID format",
			when:  "GET to /api/insights?job_id=invalid",
			then:  "should return 400 bad request",
			jobID: "invalid-uuid",
			setupService: func(jobID uuid.UUID) *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{insights: map[uuid.UUID]*insights.Insight{}},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var jobID uuid.UUID
			if tt.jobID != "" && tt.jobID != "invalid-uuid" {
				jobID = uuid.MustParse(tt.jobID)
			}
			service := tt.setupService(jobID)
			handlers := NewInsightsHandlers(service)

			url := "/api/insights"
			if tt.jobID != "" {
				url += "?job_id=" + tt.jobID
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			// When
			handlers.GetInsightByJobID(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

func TestInsightsHandlers_ListInsights(t *testing.T) {
	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		queryParams    string
		setupService   func() *appInsights.Service
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:        "Successfully list insights with default pagination",
			given:       "multiple insights in repository",
			when:        "GET to /api/insights",
			then:        "should return 200 with array of insights",
			queryParams: "",
			setupService: func() *appInsights.Service {
				insightRepo := &InMemoryInsightRepo{
					insights: map[uuid.UUID]*insights.Insight{},
					list:     []*insights.Insight{},
				}

				for i := 0; i < 3; i++ {
					insight := &insights.Insight{
						ID:             uuid.New(),
						JobID:          uuid.New(),
						Diagnosis:      "Test diagnosis",
						Recommendation: "Test recommendation",
						CreatedAt:      time.Now().UTC(),
					}
					insightRepo.insights[insight.ID] = insight
					insightRepo.list = append(insightRepo.list, insight)
				}

				return appInsights.NewService(
					insightRepo,
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp []InsightResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, 3, len(resp))
			},
		},
		{
			name:        "List insights with custom pagination",
			given:       "limit and offset parameters",
			when:        "GET to /api/insights?limit=2&offset=1",
			then:        "should return paginated results",
			queryParams: "?limit=2&offset=1",
			setupService: func() *appInsights.Service {
				insightRepo := &InMemoryInsightRepo{
					insights: map[uuid.UUID]*insights.Insight{},
					list:     []*insights.Insight{},
				}

				for i := 0; i < 5; i++ {
					insight := &insights.Insight{
						ID:        uuid.New(),
						JobID:     uuid.New(),
						CreatedAt: time.Now().UTC(),
					}
					insightRepo.list = append(insightRepo.list, insight)
				}

				return appInsights.NewService(
					insightRepo,
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp []InsightResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.LessOrEqual(t, len(resp), 2)
			},
		},
		{
			name:        "Empty list when no insights exist",
			given:       "empty repository",
			when:        "GET to /api/insights",
			then:        "should return 200 with empty array",
			queryParams: "",
			setupService: func() *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{
						insights: map[uuid.UUID]*insights.Insight{},
						list:     []*insights.Insight{},
					},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp []InsightResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, 0, len(resp))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			service := tt.setupService()
			handlers := NewInsightsHandlers(service)

			req := httptest.NewRequest(http.MethodGet, "/api/insights"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			// When
			handlers.ListInsights(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

func TestInsightsHandlers_AnalyzeJob(t *testing.T) {
	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		jobID          string
		setupService   func(uuid.UUID) *appInsights.Service
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "Successfully analyze job",
			given: "valid job ID with failed job",
			when:  "POST to /api/insights/analyze?job_id={id}",
			then:  "should return 201 with created insight",
			jobID: uuid.New().String(),
			setupService: func(jobID uuid.UUID) *appInsights.Service {
				jobRepo := &InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)}
				failedJob := &queue.Job{
					ID:        jobID,
					Queue:     "default",
					Type:      "email",
					Status:    queue.StatusFailed,
					Error:     "Connection timeout",
					Payload:   []byte(`{"to":"test@example.com"}`),
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				jobRepo.jobs[jobID] = failedJob

				insightRepo := &InMemoryInsightRepo{
					insights:      map[uuid.UUID]*insights.Insight{},
					insightsByJob: map[uuid.UUID]*insights.Insight{},
				}

				aiService := &MockAIService{
					response: &insights.AnalysisResponse{
						Diagnosis:      "Network timeout issue",
						Recommendation: "Increase timeout to 30s",
						SuggestedFix: insights.SuggestedFix{
							TimeoutSeconds: 30,
							MaxRetries:     5,
						},
					},
				}

				return appInsights.NewService(insightRepo, jobRepo, aiService)
			},
			expectedStatus: http.StatusCreated,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp InsightResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.NotEmpty(t, resp.ID)
				assert.Equal(t, "Network timeout issue", resp.Diagnosis)
				assert.Equal(t, "Increase timeout to 30s", resp.Recommendation)
			},
		},
		{
			name:  "Missing job_id parameter",
			given: "no job_id in query string",
			when:  "POST to /api/insights/analyze",
			then:  "should return 400 bad request",
			jobID: "",
			setupService: func(jobID uuid.UUID) *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{insights: map[uuid.UUID]*insights.Insight{}},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "Invalid job_id format",
			given: "invalid UUID format",
			when:  "POST to /api/insights/analyze?job_id=invalid",
			then:  "should return 400 bad request",
			jobID: "invalid-uuid",
			setupService: func(jobID uuid.UUID) *appInsights.Service {
				return appInsights.NewService(
					&InMemoryInsightRepo{insights: map[uuid.UUID]*insights.Insight{}},
					&InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)},
					&MockAIService{},
				)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var jobID uuid.UUID
			if tt.jobID != "" && tt.jobID != "invalid-uuid" {
				jobID = uuid.MustParse(tt.jobID)
			}
			service := tt.setupService(jobID)
			handlers := NewInsightsHandlers(service)

			url := "/api/insights/analyze"
			if tt.jobID != "" {
				url += "?job_id=" + tt.jobID
			}

			req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte{}))
			rec := httptest.NewRecorder()

			// When
			handlers.AnalyzeJob(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

// In-memory implementations for testing
type InMemoryInsightRepo struct {
	insights      map[uuid.UUID]*insights.Insight
	insightsByJob map[uuid.UUID]*insights.Insight
	list          []*insights.Insight
}

func (r *InMemoryInsightRepo) Create(ctx context.Context, insight *insights.Insight) error {
	r.insights[insight.ID] = insight
	r.insightsByJob[insight.JobID] = insight
	r.list = append(r.list, insight)
	return nil
}

func (r *InMemoryInsightRepo) GetByID(ctx context.Context, id uuid.UUID) (*insights.Insight, error) {
	if insight, ok := r.insights[id]; ok {
		return insight, nil
	}
	return nil, insights.ErrInsightNotFound
}

func (r *InMemoryInsightRepo) GetByJobID(ctx context.Context, jobID uuid.UUID) (*insights.Insight, error) {
	if insight, ok := r.insightsByJob[jobID]; ok {
		return insight, nil
	}
	return nil, insights.ErrInsightNotFound
}

func (r *InMemoryInsightRepo) List(ctx context.Context, limit, offset int) ([]*insights.Insight, error) {
	if offset >= len(r.list) {
		return []*insights.Insight{}, nil
	}
	end := offset + limit
	if end > len(r.list) {
		end = len(r.list)
	}
	return r.list[offset:end], nil
}

func (r *InMemoryInsightRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.insights, id)
	return nil
}

type MockAIService struct {
	response *insights.AnalysisResponse
	err      error
}

func (m *MockAIService) Analyze(ctx context.Context, request *insights.AnalysisRequest) (*insights.AnalysisResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}
	return &insights.AnalysisResponse{
		Diagnosis:      "Default diagnosis",
		Recommendation: "Default recommendation",
		SuggestedFix: insights.SuggestedFix{
			TimeoutSeconds: 30,
			MaxRetries:     3,
		},
	}, nil
}
