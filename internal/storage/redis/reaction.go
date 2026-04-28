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
	reactionFieldLikes    = "likes"
	reactionFieldDislikes = "dislikes"
)

type EventReactionStorage struct {
	client *redis.Client
}

func NewEventReactionStorage(c *redis.Client) *EventReactionStorage {
	return &EventReactionStorage{client: c}
}

func (s *EventReactionStorage) Get(ctx context.Context, req *dto.GetReactionsReq) (*dto.GetReactionsResp, error) {
	res, err := s.client.HMGet(ctx, eventReactionsKey(req.TitleHash), reactionFieldLikes, reactionFieldDislikes).Result()
	if err != nil {
		return nil, fmt.Errorf("hmget reactions: %w", err)
	}
	if len(res) != 2 || res[0] == nil || res[1] == nil {
		return nil, redis.Nil
	}

	likes, err := parseInt64Field(res[0], reactionFieldLikes)
	if err != nil {
		return nil, err
	}
	dislikes, err := parseInt64Field(res[1], reactionFieldDislikes)
	if err != nil {
		return nil, err
	}

	return &dto.GetReactionsResp{Reactions: dto.Reactions{Likes: likes, Dislikes: dislikes}}, nil
}

func (s *EventReactionStorage) Set(ctx context.Context, req *dto.SetReactionsReq) error {
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, eventReactionsKey(req.TitleHash), map[string]any{
		reactionFieldLikes:    req.Reactions.Likes,
		reactionFieldDislikes: req.Reactions.Dislikes,
	})
	pipe.Expire(ctx, eventReactionsKey(req.TitleHash), req.TTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set reactions cache: %w", err)
	}

	return nil
}

func eventReactionsKey(titleHash string) string {
	return "event:" + titleHash + ":reactions"
}

func parseInt64Field(v any, field string) (int64, error) {
	str, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("invalid %q field type", field)
	}

	parsed, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %q field: %w", field, err)
	}

	return parsed, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, redis.Nil)
}
