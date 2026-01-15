package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appQueue "github.com/erickfunier/ai-smart-queue/internal/application/queue"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestQueueHandlers_CreateJob(t *testing.T) {
	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		requestBody    any
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "Successfully create job",
			given: "valid job creation request",
			when:  "POST to /api/jobs",
			then:  "should return 201 with job details",
			requestBody: CreateJobRequest{
				Queue:   "default",
				Type:    "email",
				Payload: map[string]any{"to": "test@example.com"},
			},
			expectedStatus: http.StatusCreated,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp JobResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, "default", resp.Queue)
				assert.Equal(t, "email", resp.Type)
				assert.Equal(t, "pending", resp.Status)
			},
		},
		{
			name:           "Invalid JSON request",
			given:          "malformed JSON in request body",
			when:           "POST to /api/jobs",
			then:           "should return 400 bad request",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given - create real service with in-memory implementations for integration test
			mockRepo := &InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)}
			mockQueue := &InMemoryQueueSvc{jobs: []*queue.Job{}}
			mockMetrics := &InMemoryMetrics{}

			service := appQueue.NewService(mockRepo, mockQueue, mockMetrics)
			handlers := NewQueueHandlers(service, nil)

			var reqBody []byte
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBuffer(reqBody))
			rec := httptest.NewRecorder()

			// When
			handlers.CreateJob(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

// In-memory implementations for testing
type InMemoryJobRepo struct {
	jobs map[uuid.UUID]*queue.Job
}

func (r *InMemoryJobRepo) Create(ctx context.Context, job *queue.Job) error {
	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*queue.Job, error) {
	if job, ok := r.jobs[id]; ok {
		return job, nil
	}
	return nil, queue.ErrJobNotFound
}

func (r *InMemoryJobRepo) Update(ctx context.Context, job *queue.Job) error {
	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryJobRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.jobs, id)
	return nil
}

func (r *InMemoryJobRepo) FindPendingJobs(ctx context.Context, queueName string, limit int) ([]*queue.Job, error) {
	return nil, nil
}

func (r *InMemoryJobRepo) FindByStatus(ctx context.Context, status queue.Status, limit int) ([]*queue.Job, error) {
	var result []*queue.Job
	for _, job := range r.jobs {
		if job.Status == status && len(result) < limit {
			result = append(result, job)
		}
	}
	return result, nil
}

func (r *InMemoryJobRepo) CountByStatus(ctx context.Context, status queue.Status) (int64, error) {
	return 0, nil
}

func (r *InMemoryJobRepo) GetDLQJobs(ctx context.Context, limit, offset int) ([]*queue.Job, error) {
	return nil, nil
}

func (r *InMemoryJobRepo) MoveToDLQ(ctx context.Context, jobID uuid.UUID) error {
	return nil
}

func (r *InMemoryJobRepo) CountDLQJobs(ctx context.Context) (int64, error) {
	return 0, nil
}

type InMemoryQueueSvc struct {
	jobs []*queue.Job
}

func (q *InMemoryQueueSvc) Enqueue(ctx context.Context, job *queue.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}

func (q *InMemoryQueueSvc) Dequeue(ctx context.Context, queueName string) (*queue.Job, error) {
	return nil, nil
}

func (q *InMemoryQueueSvc) Acknowledge(ctx context.Context, jobID uuid.UUID) error {
	return nil
}

type InMemoryMetrics struct{}

func (m *InMemoryMetrics) RecordJobCreated(queueName, jobType string)                     {}
func (m *InMemoryMetrics) RecordJobCompleted(queueName, jobType string, duration float64) {}
func (m *InMemoryMetrics) RecordJobFailed(queueName, jobType string)                      {}
func (m *InMemoryMetrics) RecordJobRetried(queueName, jobType string)                     {}

func TestQueueHandlers_GetJob(t *testing.T) {
	// Create shared test IDs
	existingJobID := uuid.New()
	nonExistingJobID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		jobID          uuid.UUID
		setupRepo      func(*InMemoryJobRepo)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "Successfully get job",
			given: "existing job ID",
			when:  "GET to /api/jobs/{id}",
			then:  "should return 200 with job details",
			jobID: existingJobID,
			setupRepo: func(repo *InMemoryJobRepo) {
				job := &queue.Job{
					ID:        existingJobID,
					Queue:     "default",
					Type:      "email",
					Status:    queue.StatusPending,
					Payload:   []byte(`{"to":"test@example.com"}`),
					CreatedAt: now,
					UpdatedAt: now,
				}
				repo.jobs[existingJobID] = job
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp JobResponse
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, existingJobID.String(), resp.ID)
				assert.Equal(t, "default", resp.Queue)
			},
		},
		{
			name:           "Invalid job ID format",
			given:          "invalid UUID format",
			when:           "GET to /api/jobs/{id}",
			then:           "should return 400 bad request",
			jobID:          uuid.Nil,
			setupRepo:      func(repo *InMemoryJobRepo) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Job not found",
			given:          "non-existing job ID",
			when:           "GET to /api/jobs/{id}",
			then:           "should return 404 not found",
			jobID:          nonExistingJobID,
			setupRepo:      func(repo *InMemoryJobRepo) {},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)}
			mockQueue := &InMemoryQueueSvc{jobs: []*queue.Job{}}
			mockMetrics := &InMemoryMetrics{}
			tt.setupRepo(mockRepo)

			service := appQueue.NewService(mockRepo, mockQueue, mockMetrics)
			handlers := NewQueueHandlers(service, nil)

			// Build path
			var path string
			if tt.jobID == uuid.Nil {
				path = "/api/jobs/invalid-uuid"
			} else {
				path = "/api/jobs/" + tt.jobID.String()
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			// When
			handlers.GetJobByID(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}

func TestQueueHandlers_GetMetrics(t *testing.T) {
	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		setupRepo      func(*InMemoryJobRepo)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "Successfully get metrics",
			given: "jobs exist in different states",
			when:  "GET to /api/metrics",
			then:  "should return 200 with metrics",
			setupRepo: func(repo *InMemoryJobRepo) {
				repo.jobs[uuid.New()] = &queue.Job{Status: queue.StatusPending}
				repo.jobs[uuid.New()] = &queue.Job{Status: queue.StatusCompleted}
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var metrics map[string]any
				json.Unmarshal(rec.Body.Bytes(), &metrics)
				assert.NotNil(t, metrics)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)}
			mockQueue := &InMemoryQueueSvc{jobs: []*queue.Job{}}
			mockMetrics := &InMemoryMetrics{}
			tt.setupRepo(mockRepo)

			service := appQueue.NewService(mockRepo, mockQueue, mockMetrics)
			handlers := NewQueueHandlers(service, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
			rec := httptest.NewRecorder()

			// When
			handlers.GetMetrics(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}
func TestQueueHandlers_RetryJob(t *testing.T) {
	tests := []struct {
		name           string
		given          string
		when           string
		then           string
		jobID          string
		setupRepo      func(*InMemoryJobRepo, uuid.UUID)
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:  "Successfully retry job",
			given: "a failed job exists",
			when:  "POST to /api/jobs/retry?id={id}",
			then:  "should return 200 with success message",
			jobID: uuid.New().String(),
			setupRepo: func(repo *InMemoryJobRepo, id uuid.UUID) {
				job := &queue.Job{
					ID:       id,
					Queue:    "test-queue",
					Type:     "test",
					Status:   queue.StatusFailed,
					Attempts: 1,
				}
				repo.jobs[id] = job
			},
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]string
				json.Unmarshal(rec.Body.Bytes(), &resp)
				assert.Equal(t, "retrying", resp["status"])
			},
		},
		{
			name:           "Invalid job ID",
			given:          "an invalid job ID is provided",
			when:           "POST to /api/jobs/retry?id=invalid-id",
			then:           "should return 400 bad request",
			jobID:          "invalid-id",
			setupRepo:      func(repo *InMemoryJobRepo, id uuid.UUID) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "Job not found",
			given: "job does not exist",
			when:  "POST to /api/jobs/retry?id={id}",
			then:  "should return 500 internal server error",
			jobID: uuid.New().String(),
			setupRepo: func(repo *InMemoryJobRepo, id uuid.UUID) {
				// Don't add the job
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := &InMemoryJobRepo{jobs: make(map[uuid.UUID]*queue.Job)}
			mockQueue := &InMemoryQueueSvc{jobs: []*queue.Job{}}
			mockMetrics := &InMemoryMetrics{}

			var jobID uuid.UUID
			if id, err := uuid.Parse(tt.jobID); err == nil {
				jobID = id
			}
			tt.setupRepo(mockRepo, jobID)

			service := appQueue.NewService(mockRepo, mockQueue, mockMetrics)
			handlers := NewQueueHandlers(service, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/jobs/retry?id="+tt.jobID, nil)
			rec := httptest.NewRecorder()

			// When
			handlers.RetryJob(rec, req)

			// Then
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.validateResp != nil {
				tt.validateResp(t, rec)
			}
		})
	}
}
