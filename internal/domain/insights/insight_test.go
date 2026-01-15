package insights

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewInsight(t *testing.T) {
	validJobID := uuid.New()
	validResponse := &AnalysisResponse{
		Diagnosis:      "Network timeout",
		Recommendation: "Increase timeout",
		SuggestedFix: SuggestedFix{
			TimeoutSeconds: 30,
			MaxRetries:     5,
		},
	}

	tests := []struct {
		name string
		in   struct {
			jobID    uuid.UUID
			response *AnalysisResponse
		}
		want struct {
			err error
		}
	}{
		{
			name: "Given valid job ID and analysis response, When creating insight, Then should succeed",
			in: struct {
				jobID    uuid.UUID
				response *AnalysisResponse
			}{
				jobID:    validJobID,
				response: validResponse,
			},
			want: struct {
				err error
			}{
				err: nil,
			},
		},
		{
			name: "Given nil job ID, When creating insight, Then should return ErrInvalidJobID",
			in: struct {
				jobID    uuid.UUID
				response *AnalysisResponse
			}{
				jobID:    uuid.Nil,
				response: validResponse,
			},
			want: struct {
				err error
			}{
				err: ErrInvalidJobID,
			},
		},
		{
			name: "Given nil analysis response, When creating insight, Then should return ErrInvalidAnalysisData",
			in: struct {
				jobID    uuid.UUID
				response *AnalysisResponse
			}{
				jobID:    validJobID,
				response: nil,
			},
			want: struct {
				err error
			}{
				err: ErrInvalidAnalysisData,
			},
		},
		{
			name: "Given response with empty diagnosis, When creating insight, Then should return ErrInvalidAnalysisData",
			in: struct {
				jobID    uuid.UUID
				response *AnalysisResponse
			}{
				jobID:    validJobID,
				response: &AnalysisResponse{Diagnosis: ""},
			},
			want: struct {
				err error
			}{
				err: ErrInvalidAnalysisData,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insight, err := NewInsight(tt.in.jobID, tt.in.response)

			if tt.want.err != nil {
				assert.ErrorIs(t, err, tt.want.err)
				assert.Nil(t, insight)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, insight)
				assert.NotEqual(t, uuid.Nil, insight.ID)
				assert.Equal(t, tt.in.jobID, insight.JobID)
				assert.Equal(t, tt.in.response.Diagnosis, insight.Diagnosis)
				assert.Equal(t, tt.in.response.Recommendation, insight.Recommendation)
				assert.False(t, insight.CreatedAt.IsZero())
			}
		})
	}
}

func TestInsight_ApplySuggestedFix(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			originalPayload string
			payloadPatch    map[string]any
		}
		want struct {
			expectedPayload string
			hasError        bool
		}
	}{
		{
			name: "Given payload with timeout 10 and patch with timeout 30, When applying fix, Then should merge patch",
			in: struct {
				originalPayload string
				payloadPatch    map[string]any
			}{
				originalPayload: `{"timeout":10,"url":"http://example.com"}`,
				payloadPatch:    map[string]any{"timeout": 30},
			},
			want: struct {
				expectedPayload string
				hasError        bool
			}{
				expectedPayload: `{"timeout":30,"url":"http://example.com"}`,
				hasError:        false,
			},
		},
		{
			name: "Given empty payload patch, When applying fix, Then should return original payload",
			in: struct {
				originalPayload string
				payloadPatch    map[string]any
			}{
				originalPayload: `{"timeout":10}`,
				payloadPatch:    map[string]any{},
			},
			want: struct {
				expectedPayload string
				hasError        bool
			}{
				expectedPayload: `{"timeout":10}`,
				hasError:        false,
			},
		},
		{
			name: "Given patch with new field, When applying fix, Then should add field to payload",
			in: struct {
				originalPayload string
				payloadPatch    map[string]any
			}{
				originalPayload: `{"url":"http://example.com"}`,
				payloadPatch:    map[string]any{"retry_count": 5},
			},
			want: struct {
				expectedPayload string
				hasError        bool
			}{
				expectedPayload: `{"retry_count":5,"url":"http://example.com"}`,
				hasError:        false,
			},
		},
		{
			name: "Given invalid JSON payload, When applying fix, Then should return error",
			in: struct {
				originalPayload string
				payloadPatch    map[string]any
			}{
				originalPayload: `{invalid json}`,
				payloadPatch:    map[string]any{"timeout": 30},
			},
			want: struct {
				expectedPayload string
				hasError        bool
			}{
				expectedPayload: "",
				hasError:        true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insight := &Insight{
				SuggestedFix: SuggestedFix{
					PayloadPatch: tt.in.payloadPatch,
				},
			}

			result, err := insight.ApplySuggestedFix([]byte(tt.in.originalPayload))

			if tt.want.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Compare JSON objects, not strings (to handle key ordering)
				var expected, actual map[string]any
				json.Unmarshal([]byte(tt.want.expectedPayload), &expected)
				json.Unmarshal(result, &actual)
				assert.Equal(t, expected, actual)
			}
		})
	}
}

func TestInsight_HasTimeoutRecommendation(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			timeoutSeconds int
		}
		want struct {
			hasTimeout bool
		}
	}{
		{
			name: "Given insight with timeout_seconds > 0, When checking timeout recommendation, Then should return true",
			in: struct {
				timeoutSeconds int
			}{
				timeoutSeconds: 30,
			},
			want: struct {
				hasTimeout bool
			}{
				hasTimeout: true,
			},
		},
		{
			name: "Given insight with timeout_seconds = 0, When checking timeout recommendation, Then should return false",
			in: struct {
				timeoutSeconds int
			}{
				timeoutSeconds: 0,
			},
			want: struct {
				hasTimeout bool
			}{
				hasTimeout: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insight := &Insight{
				SuggestedFix: SuggestedFix{
					TimeoutSeconds: tt.in.timeoutSeconds,
				},
			}

			result := insight.HasTimeoutRecommendation()

			assert.Equal(t, tt.want.hasTimeout, result)
		})
	}
}

func TestInsight_HasRetryRecommendation(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			maxRetries int
		}
		want struct {
			hasRetry bool
		}
	}{
		{
			name: "Given insight with max_retries > 0, When checking retry recommendation, Then should return true",
			in: struct {
				maxRetries int
			}{
				maxRetries: 5,
			},
			want: struct {
				hasRetry bool
			}{
				hasRetry: true,
			},
		},
		{
			name: "Given insight with max_retries = 0, When checking retry recommendation, Then should return false",
			in: struct {
				maxRetries int
			}{
				maxRetries: 0,
			},
			want: struct {
				hasRetry bool
			}{
				hasRetry: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insight := &Insight{
				SuggestedFix: SuggestedFix{
					MaxRetries: tt.in.maxRetries,
				},
			}

			result := insight.HasRetryRecommendation()

			assert.Equal(t, tt.want.hasRetry, result)
		})
	}
}
