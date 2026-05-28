package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"ndbx/internal/storage/redis/dto"
)

type RecommendationCacheStorage struct {
	client *redis.Client
}

func NewRecommendationCacheStorage(c *redis.Client) *RecommendationCacheStorage {
	return &RecommendationCacheStorage{client: c}
}

func (s *RecommendationCacheStorage) Get(ctx context.Context, req *dto.GetRecommendationReq) (*dto.GetRecommendationResp, error) {
	result, err := s.client.HGetAll(ctx, recommendationKey(req.UserID)).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall recommendations: %w", err)
	}

	if len(result) == 0 {
		return nil, redis.Nil
	}

	eventsJSON, ok := result["events"]
	if !ok {
		return nil, redis.Nil
	}

	var events []dto.RecommendationEvent
	if err := json.Unmarshal([]byte(eventsJSON), &events); err != nil {
		return nil, fmt.Errorf("unmarshal recommendation events: %w", err)
	}

	return &dto.GetRecommendationResp{Events: events}, nil
}

func (s *RecommendationCacheStorage) Set(ctx context.Context, req *dto.SetRecommendationReq) error {
	eventsJSON, err := json.Marshal(req.Events)
	if err != nil {
		return fmt.Errorf("marshal recommendation events: %w", err)
	}

	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, recommendationKey(req.UserID), "events", string(eventsJSON))
	pipe.Expire(ctx, recommendationKey(req.UserID), req.TTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set recommendations cache: %w", err)
	}

	return nil
}

func recommendationKey(userID string) string {
	return "user:" + userID + ":recomms"
}

func IsRecommendationNotFound(err error) bool {
	return errors.Is(err, redis.Nil)
}
