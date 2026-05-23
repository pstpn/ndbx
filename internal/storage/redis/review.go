package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"

	"ndbx/internal/storage/redis/dto"
)

const (
	reviewFieldCount  = "count"
	reviewFieldRating = "rating"
)

type EventReviewCacheStorage struct {
	client *redis.Client
}

func NewEventReviewCacheStorage(c *redis.Client) *EventReviewCacheStorage {
	return &EventReviewCacheStorage{client: c}
}

func (s *EventReviewCacheStorage) Get(ctx context.Context, req *dto.GetReviewsReq) (*dto.GetReviewsResp, error) {
	res, err := s.client.HMGet(ctx, eventReviewsKey(req.TitleHash), reviewFieldCount, reviewFieldRating).Result()
	if err != nil {
		return nil, fmt.Errorf("hmget reviews: %w", err)
	}
	if len(res) != 2 || res[0] == nil || res[1] == nil {
		return nil, redis.Nil
	}

	count, err := parseInt64Field(res[0], reviewFieldCount)
	if err != nil {
		return nil, err
	}
	rating, err := parseFloat64Field(res[1], reviewFieldRating)
	if err != nil {
		return nil, err
	}

	return &dto.GetReviewsResp{Reviews: dto.Reviews{Count: count, Rating: rating}}, nil
}

func (s *EventReviewCacheStorage) Set(ctx context.Context, req *dto.SetReviewsReq) error {
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, eventReviewsKey(req.TitleHash), map[string]any{
		reviewFieldCount:  req.Reviews.Count,
		reviewFieldRating: req.Reviews.Rating,
	})
	pipe.Expire(ctx, eventReviewsKey(req.TitleHash), req.TTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set reviews cache: %w", err)
	}

	return nil
}

func eventReviewsKey(titleHash string) string {
	return "event:" + titleHash + ":reviews"
}

func parseFloat64Field(v any, field string) (float64, error) {
	str, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("invalid %q field type", field)
	}

	parsed, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %q field: %w", field, err)
	}

	return parsed, nil
}

func IsReviewNotFound(err error) bool {
	return errors.Is(err, redis.Nil)
}
