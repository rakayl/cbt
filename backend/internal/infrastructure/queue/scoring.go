package queue

import (
	"context"
	"encoding/json"
	"time"

	"cbt-enterprise/internal/infrastructure/redis"
)

const ScoringQueueKey = "queue:scoring"

type ScoringJob struct {
	AttemptID string `json:"attempt_id"`
	UjianID   string `json:"ujian_id"`
	PesertaID string `json:"peserta_id"`
}

type ScoringQueue struct {
	redis *redis.Client
}

func NewScoringQueue(rc *redis.Client) *ScoringQueue {
	return &ScoringQueue{redis: rc}
}

func (q *ScoringQueue) Push(ctx context.Context, job ScoringJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.redis.LPush(ctx, ScoringQueueKey, data)
}

func (q *ScoringQueue) Pop(ctx context.Context) (*ScoringJob, error) {
	result, err := q.redis.BRPop(ctx, 5*time.Second, ScoringQueueKey)
	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, nil
	}
	var job ScoringJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}
