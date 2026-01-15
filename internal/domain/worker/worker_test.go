package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkerConfig(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			queueName     string
			maxAttempts   int
			baseBackoffMs int
		}
		want struct {
			err error
		}
	}{
		{
			name: "Given valid queue name and positive max attempts, When creating worker config, Then should succeed",
			in: struct {
				queueName     string
				maxAttempts   int
				baseBackoffMs int
			}{
				queueName:     "default",
				maxAttempts:   3,
				baseBackoffMs: 500,
			},
			want: struct {
				err error
			}{
				err: nil,
			},
		},
		{
			name: "Given empty queue name, When creating worker config, Then should return ErrQueueNameRequired",
			in: struct {
				queueName     string
				maxAttempts   int
				baseBackoffMs int
			}{
				queueName:     "",
				maxAttempts:   3,
				baseBackoffMs: 500,
			},
			want: struct {
				err error
			}{
				err: ErrQueueNameRequired,
			},
		},
		{
			name: "Given max attempts set to 0, When creating worker config, Then should return ErrMaxAttemptsInvalid",
			in: struct {
				queueName     string
				maxAttempts   int
				baseBackoffMs int
			}{
				queueName:     "default",
				maxAttempts:   0,
				baseBackoffMs: 500,
			},
			want: struct {
				err error
			}{
				err: ErrMaxAttemptsInvalid,
			},
		},
		{
			name: "Given negative max attempts, When creating worker config, Then should return ErrMaxAttemptsInvalid",
			in: struct {
				queueName     string
				maxAttempts   int
				baseBackoffMs int
			}{
				queueName:     "default",
				maxAttempts:   -1,
				baseBackoffMs: 500,
			},
			want: struct {
				err error
			}{
				err: ErrMaxAttemptsInvalid,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewWorkerConfig(tt.in.queueName, tt.in.maxAttempts, tt.in.baseBackoffMs)

			if tt.want.err != nil {
				assert.ErrorIs(t, err, tt.want.err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.in.queueName, config.QueueName)
				assert.Equal(t, tt.in.maxAttempts, config.MaxAttempts)
				assert.Equal(t, tt.in.baseBackoffMs, config.BaseBackoffMs)
				assert.Equal(t, 5*time.Second, config.PollInterval)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name string
		in   struct {
			attempt int
			baseMs  int
		}
		want struct {
			duration time.Duration
		}
	}{
		{
			name: "Given attempt 0 and base 500ms, When calculating backoff, Then should return 500ms",
			in: struct {
				attempt int
				baseMs  int
			}{
				attempt: 0,
				baseMs:  500,
			},
			want: struct {
				duration time.Duration
			}{
				duration: 500 * time.Millisecond,
			},
		},
		{
			name: "Given attempt 1 and base 500ms, When calculating backoff, Then should return 1000ms",
			in: struct {
				attempt int
				baseMs  int
			}{
				attempt: 1,
				baseMs:  500,
			},
			want: struct {
				duration time.Duration
			}{
				duration: 1000 * time.Millisecond,
			},
		},
		{
			name: "Given attempt 2 and base 500ms, When calculating backoff, Then should return 2000ms",
			in: struct {
				attempt int
				baseMs  int
			}{
				attempt: 2,
				baseMs:  500,
			},
			want: struct {
				duration time.Duration
			}{
				duration: 2000 * time.Millisecond,
			},
		},
		{
			name: "Given attempt 3 and base 500ms, When calculating backoff, Then should return 4000ms",
			in: struct {
				attempt int
				baseMs  int
			}{
				attempt: 3,
				baseMs:  500,
			},
			want: struct {
				duration time.Duration
			}{
				duration: 4000 * time.Millisecond,
			},
		},
		{
			name: "Given negative attempt number, When calculating backoff, Then should treat as 0 and return base",
			in: struct {
				attempt int
				baseMs  int
			}{
				attempt: -1,
				baseMs:  500,
			},
			want: struct {
				duration time.Duration
			}{
				duration: 500 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBackoff(tt.in.attempt, tt.in.baseMs)

			assert.Equal(t, tt.want.duration, result)
		})
	}
}
