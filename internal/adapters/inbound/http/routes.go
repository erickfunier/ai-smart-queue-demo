package http

import (
	"log"
	"net/http"
)

// RegisterQueueRoutes registers all queue-related routes
func RegisterQueueRoutes(mux *http.ServeMux, handlers *QueueHandlers) {
	// Register handler for /api/jobs (without trailing slash)
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		log.Printf("[Router] Path: %s, Method: %s", path, r.Method)

		if path == "/api/jobs" || path == "/api/jobs/" {
			// /api/jobs endpoint
			switch r.Method {
			case http.MethodPost:
				handlers.CreateJob(w, r)
			case http.MethodGet:
				// List jobs with optional filters
				handlers.ListJobs(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			// /api/jobs/{id} endpoint
			if r.Method == http.MethodGet {
				handlers.GetJobByID(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})

	// POST /api/jobs - Create job
	// GET /api/jobs - List jobs with optional filters and pagination
	// GET /api/jobs/{id} - Get specific job by ID
	mux.HandleFunc("/api/jobs/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		log.Printf("[Router] Path: %s, Method: %s", path, r.Method)

		if path == "/api/jobs" || path == "/api/jobs/" {
			// /api/jobs endpoint
			switch r.Method {
			case http.MethodPost:
				handlers.CreateJob(w, r)
			case http.MethodGet:
				// List jobs with optional filters
				handlers.ListJobs(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			// /api/jobs/{id} endpoint
			if r.Method == http.MethodGet {
				handlers.GetJobByID(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}
	})

	mux.HandleFunc("/api/jobs/retry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.RetryJob(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/dlq", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetDLQJobs(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handlers.GetMetrics(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// RegisterInsightsRoutes registers all insights-related routes
func RegisterInsightsRoutes(mux *http.ServeMux, handlers *InsightsHandlers) {
	// GET /api/insights - List insights with optional filters and pagination
	// GET /api/insights/{id} - Get specific insight by ID
	mux.HandleFunc("/api/insights/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract ID from path if present
		path := r.URL.Path
		if len(path) > len("/api/insights/") {
			// Path has an ID: /api/insights/{id}
			handlers.GetInsightByID(w, r)
		} else {
			// No ID in path: /api/insights or /api/insights/
			// Check for job_id filter
			if r.URL.Query().Get("job_id") != "" {
				handlers.GetInsightByJobID(w, r)
			} else {
				handlers.ListInsights(w, r)
			}
		}
	})

	mux.HandleFunc("/api/insights/analyze", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.AnalyzeJob(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
