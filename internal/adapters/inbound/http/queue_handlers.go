package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	appQueue "github.com/erickfunier/ai-smart-queue/internal/application/queue"
	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
)

// QueueHandlers handles HTTP requests for queue operations
type QueueHandlers struct {
	queueService    *appQueue.Service
	insightsService *appInsights.Service
}

// NewQueueHandlers creates a new queue HTTP handlers
func NewQueueHandlers(queueService *appQueue.Service, insightsService *appInsights.Service) *QueueHandlers {
	return &QueueHandlers{
		queueService:    queueService,
		insightsService: insightsService,
	}
}

type CreateJobRequest struct {
	Queue   string      `json:"queue"`
	Type    string      `json:"type"`
	Payload any `json:"payload"`
}

type JobResponse struct {
	ID        string           `json:"id"`
	Queue     string           `json:"queue"`
	Type      string           `json:"type"`
	Status    string           `json:"status"`
	Attempts  int              `json:"attempts"`
	Payload   any      `json:"payload"`
	Error     string           `json:"error,omitempty"`
	Insight   *InsightResponse `json:"insight,omitempty"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

func (h *QueueHandlers) CreateJob(w http.ResponseWriter, r *http.Request) {
	log.Printf("[CreateJob] Received request from %s", r.RemoteAddr)
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[CreateJob] Failed to decode request: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	log.Printf("[CreateJob] Creating job: queue=%s, type=%s", req.Queue, req.Type)

	cmd := appQueue.CreateJobCommand{
		Queue:   req.Queue,
		Type:    req.Type,
		Payload: req.Payload,
	}

	job, err := h.queueService.CreateJob(r.Context(), cmd)
	if err != nil {
		log.Printf("[CreateJob] Failed to create job: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[CreateJob] Job created successfully: id=%s, queue=%s", job.ID, job.Queue)

	var payload any
	json.Unmarshal(job.Payload, &payload)

	response := JobResponse{
		ID:        job.ID.String(),
		Queue:     job.Queue,
		Type:      job.Type,
		Status:    string(job.Status),
		Attempts:  job.Attempts,
		Payload:   payload,
		CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: job.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[CreateJob] Failed to encode response: %v", err)
	}
}

func (h *QueueHandlers) GetJobByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/jobs/{id}
	idStr := r.URL.Path[len("/api/jobs/"):]
	if idStr == "" {
		log.Printf("[GetJobByID] Missing job ID in path")
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[GetJobByID] Invalid job ID: %s", idStr)
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}

	log.Printf("[GetJobByID] Fetching job: id=%s", id)
	job, err := h.queueService.GetJob(r.Context(), id)
	if err != nil {
		log.Printf("[GetJobByID] Job not found: id=%s", id)
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	log.Printf("[GetJobByID] Job retrieved: id=%s, status=%s", job.ID, job.Status)

	var payload any
	json.Unmarshal(job.Payload, &payload)

	response := JobResponse{
		ID:        job.ID.String(),
		Queue:     job.Queue,
		Type:      job.Type,
		Status:    string(job.Status),
		Attempts:  job.Attempts,
		Payload:   payload,
		Error:     job.Error,
		CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: job.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Try to fetch insights for this job if it has failed
	if h.insightsService != nil && job.Status == queue.StatusFailed {
		insight, err := h.insightsService.GetInsightByJobID(r.Context(), id)
		if err == nil && insight != nil {
			log.Printf("[GetJob] Including insight in response: insight_id=%s", insight.ID)
			response.Insight = &InsightResponse{
				ID:             insight.ID.String(),
				JobID:          insight.JobID.String(),
				Diagnosis:      insight.Diagnosis,
				Recommendation: insight.Recommendation,
				SuggestedFix: map[string]any{
					"timeout_seconds": insight.SuggestedFix.TimeoutSeconds,
					"max_retries":     insight.SuggestedFix.MaxRetries,
					"payload_patch":   insight.SuggestedFix.PayloadPatch,
				},
				CreatedAt: insight.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *QueueHandlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	// Optional filters
	statusStr := r.URL.Query().Get("status")
	queueName := r.URL.Query().Get("queue")

	// Pagination
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	log.Printf("[ListJobs] Fetching jobs: status=%s, queue=%s, limit=%d, offset=%d", statusStr, queueName, limit, offset)

	var jobs []*queue.Job
	var err error

	// If status filter is provided, use GetJobsByStatus
	if statusStr != "" {
		jobs, err = h.queueService.GetJobsByStatus(r.Context(), queue.Status(statusStr), limit)
		if err != nil {
			log.Printf("[ListJobs] Failed to fetch jobs: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// TODO: Implement GetAllJobs in service for listing without status filter
		// For now, return empty array if no filter provided
		log.Printf("[ListJobs] No status filter - returning empty list (implement GetAllJobs for full listing)")
		jobs = []*queue.Job{}
	}

	log.Printf("[ListJobs] Found %d jobs", len(jobs))

	var responses []JobResponse
	for _, job := range jobs {
		var payload any
		json.Unmarshal(job.Payload, &payload)

		responses = append(responses, JobResponse{
			ID:        job.ID.String(),
			Queue:     job.Queue,
			Type:      job.Type,
			Status:    string(job.Status),
			Attempts:  job.Attempts,
			Payload:   payload,
			Error:     job.Error,
			CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: job.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

func (h *QueueHandlers) GetDLQJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	log.Printf("[GetDLQJobs] Fetching DLQ jobs: limit=%d, offset=%d", limit, offset)
	jobs, total, err := h.queueService.GetDLQJobs(r.Context(), limit, offset)
	if err != nil {
		log.Printf("[GetDLQJobs] Failed to fetch DLQ jobs: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[GetDLQJobs] Found %d DLQ jobs (total=%d)", len(jobs), total)

	var responses []JobResponse
	for _, job := range jobs {
		var payload any
		json.Unmarshal(job.Payload, &payload)

		responses = append(responses, JobResponse{
			ID:        job.ID.String(),
			Queue:     job.Queue,
			Type:      job.Type,
			Status:    string(job.Status),
			Attempts:  job.Attempts,
			Payload:   payload,
			Error:     job.Error,
			CreatedAt: job.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: job.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	result := map[string]any{
		"jobs":   responses,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *QueueHandlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	log.Printf("[GetMetrics] Fetching queue metrics")
	metrics, err := h.queueService.GetMetrics(r.Context())
	if err != nil {
		log.Printf("[GetMetrics] Failed to fetch metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[GetMetrics] Metrics retrieved successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (h *QueueHandlers) RetryJob(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		log.Printf("[RetryJob] Missing job ID parameter")
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[RetryJob] Invalid job ID: %s", idStr)
		http.Error(w, "invalid job id", http.StatusBadRequest)
		return
	}

	log.Printf("[RetryJob] Retrying job: id=%s", id)
	maxAttempts := 3
	if err := h.queueService.RetryJob(r.Context(), id, maxAttempts); err != nil {
		log.Printf("[RetryJob] Failed to retry job: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[RetryJob] Job retry initiated: id=%s", id)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "retrying"})
}
