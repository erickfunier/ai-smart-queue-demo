package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	appInsights "github.com/erickfunier/ai-smart-queue/internal/application/insights"
	"github.com/google/uuid"
)

// InsightsHandlers handles HTTP requests for insights operations
type InsightsHandlers struct {
	insightsService *appInsights.Service
}

// NewInsightsHandlers creates a new insights HTTP handlers
func NewInsightsHandlers(insightsService *appInsights.Service) *InsightsHandlers {
	return &InsightsHandlers{
		insightsService: insightsService,
	}
}

type InsightResponse struct {
	ID             string         `json:"id"`
	JobID          string         `json:"job_id"`
	Diagnosis      string         `json:"diagnosis"`
	Recommendation string         `json:"recommendation"`
	SuggestedFix   map[string]any `json:"suggested_fix"`
	CreatedAt      string         `json:"created_at"`
}

func (h *InsightsHandlers) GetInsightByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/insights/{id}
	idStr := r.URL.Path[len("/api/insights/"):]
	if idStr == "" {
		log.Printf("[GetInsightByID] Missing insight ID in path")
		http.Error(w, "insight id is required", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[GetInsightByID] Invalid insight ID: %s", idStr)
		http.Error(w, "invalid insight id", http.StatusBadRequest)
		return
	}

	log.Printf("[GetInsightByID] Fetching insight: id=%s", id)
	insight, err := h.insightsService.GetInsight(r.Context(), id)
	if err != nil {
		log.Printf("[GetInsightByID] Insight not found: id=%s", id)
		http.Error(w, "insight not found", http.StatusNotFound)
		return
	}
	log.Printf("[GetInsightByID] Insight retrieved: id=%s, job_id=%s", insight.ID, insight.JobID)

	response := InsightResponse{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *InsightsHandlers) GetInsightByJobID(w http.ResponseWriter, r *http.Request) {
	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		log.Printf("[GetInsightByJobID] Missing job_id parameter")
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		log.Printf("[GetInsightByJobID] Invalid job_id: %s", jobIDStr)
		http.Error(w, "invalid job_id", http.StatusBadRequest)
		return
	}

	log.Printf("[GetInsightByJobID] Fetching insight for job: job_id=%s", jobID)
	insight, err := h.insightsService.GetInsightByJobID(r.Context(), jobID)
	if err != nil {
		log.Printf("[GetInsightByJobID] Insight not found for job: job_id=%s", jobID)
		http.Error(w, "insight not found", http.StatusNotFound)
		return
	}
	log.Printf("[GetInsightByJobID] Insight retrieved: id=%s, job_id=%s", insight.ID, insight.JobID)

	response := InsightResponse{
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *InsightsHandlers) ListInsights(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("[ListInsights] Fetching insights: limit=%d, offset=%d", limit, offset)
	insights, err := h.insightsService.ListInsights(r.Context(), limit, offset)
	if err != nil {
		log.Printf("[ListInsights] Failed to fetch insights: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[ListInsights] Found %d insights", len(insights))

	var responses []InsightResponse
	for _, insight := range insights {
		responses = append(responses, InsightResponse{
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
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

func (h *InsightsHandlers) AnalyzeJob(w http.ResponseWriter, r *http.Request) {
	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		http.Error(w, "invalid job_id", http.StatusBadRequest)
		return
	}

	// Create a context with longer timeout for AI analysis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	insight, err := h.insightsService.AnalyzeJobFailure(ctx, jobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := InsightResponse{
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
