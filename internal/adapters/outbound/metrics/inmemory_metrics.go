package metrics

import (
	"sync"
)

// InMemoryMetricsService implements queue.MetricsService with in-memory storage
type InMemoryMetricsService struct {
	mu      sync.RWMutex
	metrics map[string]int64
}

// NewInMemoryMetricsService creates a new in-memory metrics service
func NewInMemoryMetricsService() *InMemoryMetricsService {
	return &InMemoryMetricsService{
		metrics: make(map[string]int64),
	}
}

func (s *InMemoryMetricsService) RecordJobCreated(queue, jobType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "created:" + queue + ":" + jobType
	s.metrics[key]++
}

func (s *InMemoryMetricsService) RecordJobCompleted(queue, jobType string, duration float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "completed:" + queue + ":" + jobType
	s.metrics[key]++
}

func (s *InMemoryMetricsService) RecordJobFailed(queue, jobType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "failed:" + queue + ":" + jobType
	s.metrics[key]++
}

func (s *InMemoryMetricsService) RecordJobRetried(queue, jobType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := "retried:" + queue + ":" + jobType
	s.metrics[key]++
}

func (s *InMemoryMetricsService) GetMetrics() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range s.metrics {
		result[k] = v
	}
	return result
}
