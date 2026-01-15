package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/erickfunier/ai-smart-queue/internal/domain/insights"
)

// HTTPClient is an adapter that calls a remote insights service via HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client for the insights service
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for AI analysis (first load can be slow)
		},
	}
}

// Analyze calls the remote insights API to analyze a job failure
func (c *HTTPClient) Analyze(ctx context.Context, request *insights.AnalysisRequest) (*insights.AnalysisResponse, error) {
	// The insights API expects job_id as a query parameter, not in the body
	url := fmt.Sprintf("%s/api/insights/analyze?job_id=%s", c.baseURL, request.JobID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call insights API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("insights API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var insight insights.Insight
	if err := json.NewDecoder(resp.Body).Decode(&insight); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert the insight to an AnalysisResponse
	return &insights.AnalysisResponse{
		Diagnosis:      insight.Diagnosis,
		Recommendation: insight.Recommendation,
		SuggestedFix:   insight.SuggestedFix,
	}, nil
}
