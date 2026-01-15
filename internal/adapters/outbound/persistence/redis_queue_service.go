package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisQueueService implements queue.QueueService using Redis
type RedisQueueService struct {
	client *redis.Client
}

// NewRedisQueueService creates a new Redis queue service
func NewRedisQueueService(client *redis.Client) *RedisQueueService {
	return &RedisQueueService{client: client}
}

func (s *RedisQueueService) Enqueue(ctx context.Context, job *queue.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("queue:%s", job.Queue)
	return s.client.LPush(ctx, key, data).Err()
}

func (s *RedisQueueService) Dequeue(ctx context.Context, queueName string) (*queue.Job, error) {
	key := fmt.Sprintf("queue:%s", queueName)

	result, err := s.client.BRPop(ctx, 0, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) < 2 {
		return nil, nil
	}

	var job queue.Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, err
	}

	return &job, nil
}

func (s *RedisQueueService) Acknowledge(ctx context.Context, jobID uuid.UUID) error {
	// Remove from processing set if we're tracking that
	key := fmt.Sprintf("processing:%s", jobID.String())
	return s.client.Del(ctx, key).Err()
}
