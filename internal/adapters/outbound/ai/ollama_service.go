package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/erickfunier/ai-smart-queue/internal/domain/insights"
)

// OllamaAIService implements insights.AIService using Ollama
type OllamaAIService struct {
	baseURL string
	client  *http.Client
}

// NewOllamaAIService creates a new Ollama AI service
func NewOllamaAIService(baseURL string) *OllamaAIService {
	return &OllamaAIService{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (s *OllamaAIService) Analyze(ctx context.Context, request *insights.AnalysisRequest) (*insights.AnalysisResponse, error) {
	prompt := map[string]string{
		"model": "phi3:mini",
		"prompt": `
			You are an expert in distributed systems debugging.
			Return ONLY valid JSON. No comments, no markdown, no explanations.

			Job ID: ` + request.JobID + `
			Error: ` + request.Error + `
			Payload: ` + request.Payload + `

			Return EXACTLY this JSON structure, with no extra text:

			{
				"diagnosis": "<short reason>",
				"recommendation": "<human-readable advice>",
				"suggested_fix": {
					"timeout_seconds": <int>,
					"max_retries": <int>,
					"payload_patch": { }
				}
			}
		`,
	}

	body, err := json.Marshal(prompt)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/generate", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("ollama request failed")
	}

	// Ollama streams responses, we need to collect all chunks
	var fullResponse string
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk map[string]any
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if part, ok := chunk["response"].(string); ok {
			fullResponse += part
		}
	}

	// Extract JSON from the response
	fullResponse = strings.TrimSpace(fullResponse)

	// Find JSON boundaries
	start := strings.Index(fullResponse, "{")
	end := strings.LastIndex(fullResponse, "}")
	if start == -1 || end == -1 {
		return nil, errors.New("no valid JSON found in response")
	}
	jsonStr := fullResponse[start : end+1]

	var analysisResp insights.AnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &analysisResp); err != nil {
		return nil, err
	}

	return &analysisResp, nil
}
