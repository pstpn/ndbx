package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"ndbx/internal/storage/redis/dto"
)

const (
	createdAtField = "created_at"
	updatedAtField = "updated_at"
)

type SessionStorage struct {
	mu     *sync.Mutex
	client *redis.Client
}

func NewSessionStorage(c *redis.Client) *SessionStorage {
	return &SessionStorage{
		mu:     &sync.Mutex{},
		client: c,
	}
}

func (s *SessionStorage) Set(ctx context.Context, req *dto.SetReq) error {
	return s.setValue(ctx, req)
}

func (s *SessionStorage) SetOrUpdate(ctx context.Context, req *dto.SetOrUpdateReq) (*dto.SetOrUpdateResp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	createdAtRaw, err := s.client.HGet(ctx, sessionKey(req.SID), createdAtField).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			if err := s.setValue(ctx, &dto.SetReq{
				SID:   req.NewSID,
				Value: req.NewValue,
				TTL:   req.TTL,
			}); err != nil {
				return nil, err
			}

			return &dto.SetOrUpdateResp{SID: req.NewSID, IsCreated: true}, nil
		}

		return nil, fmt.Errorf("hget: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return nil, fmt.Errorf("parse %q field: %w", createdAtField, err)
	}

	if err := s.setValue(ctx, &dto.SetReq{
		SID:   req.SID,
		Value: dto.SessionValue{CreatedAt: createdAt, UpdatedAt: req.NewValue.UpdatedAt},
		TTL:   req.TTL,
	}); err != nil {
		return nil, err
	}

	return &dto.SetOrUpdateResp{SID: req.SID, IsCreated: false}, nil
}

func (s *SessionStorage) Get(ctx context.Context, req *dto.GetReq) (*dto.GetResp, error) {
	res, err := s.client.HMGet(ctx, sessionKey(req.SID), createdAtField, updatedAtField).Result()
	if err != nil {
		return nil, fmt.Errorf("hmget: %w", err)
	}
	if len(res) != 2 || res[0] == nil || res[1] == nil {
		return nil, fmt.Errorf("invalid : %w", redis.Nil)
	}

	createdAt, err := parseTime(res[0], time.RFC3339, createdAtField)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(res[1], time.RFC3339, updatedAtField)
	if err != nil {
		return nil, err
	}

	return &dto.GetResp{SessionValue: dto.SessionValue{CreatedAt: createdAt, UpdatedAt: updatedAt}}, nil
}

func (s *SessionStorage) setValue(ctx context.Context, req *dto.SetReq) error {
	pipe := s.client.TxPipeline()
	pipe.HSet(ctx, sessionKey(req.SID),
		createdAtField, req.Value.CreatedAt.UTC().Format(time.RFC3339),
		updatedAtField, req.Value.UpdatedAt.UTC().Format(time.RFC3339),
	)
	pipe.Expire(ctx, sessionKey(req.SID), req.TTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("hset with expire: %w", err)
	}

	return nil
}

func sessionKey(sid string) string {
	return "sid:" + sid
}

func parseTime(t any, layout string, field string) (time.Time, error) {
	timeStr, ok := t.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid %q field type: expected: string", field)
	}

	parsedTime, err := time.Parse(layout, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %q field: %w", field, err)
	}

	return parsedTime, nil
}
