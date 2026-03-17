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
	userIDField    = "user_id"
)

type SessionStorage struct {
	client *redis.Client
	mu     sync.Mutex
}

func NewSessionStorage(c *redis.Client) *SessionStorage {
	return &SessionStorage{
		mu:     sync.Mutex{},
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
		Value: dto.SessionValue{CreatedAt: createdAt, UpdatedAt: req.NewValue.UpdatedAt, UserID: req.NewValue.UserID},
		TTL:   req.TTL,
	}); err != nil {
		return nil, err
	}

	return &dto.SetOrUpdateResp{SID: req.SID, IsCreated: false}, nil
}

func (s *SessionStorage) Get(ctx context.Context, req *dto.GetReq) (*dto.GetResp, error) {
	res, err := s.client.HMGet(ctx, sessionKey(req.SID), createdAtField, updatedAtField, userIDField).Result()
	if err != nil {
		return nil, fmt.Errorf("hmget: %w", err)
	}
	if len(res) != 3 || res[0] == nil || res[1] == nil {
		return nil, redis.Nil
	}

	createdAt, err := parseTime(res[0], time.RFC3339, createdAtField)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(res[1], time.RFC3339, updatedAtField)
	if err != nil {
		return nil, err
	}

	userID := ""
	if res[2] != nil {
		userIDValue, ok := res[2].(string)
		if !ok {
			return nil, fmt.Errorf("invalid %q field type: expected: string", userIDField)
		}
		userID = userIDValue
	}

	return &dto.GetResp{SessionValue: dto.SessionValue{CreatedAt: createdAt, UpdatedAt: updatedAt, UserID: userID}}, nil
}

func (s *SessionStorage) Delete(ctx context.Context, req *dto.DeleteReq) error {
	if err := s.client.Del(ctx, sessionKey(req.SID)).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *SessionStorage) setValue(ctx context.Context, req *dto.SetReq) error {
	if err := s.client.HSetEXWithArgs(ctx, sessionKey(req.SID),
		&redis.HSetEXOptions{
			ExpirationType: redis.HSetEXExpirationEX,
			ExpirationVal:  int64(req.TTL.Seconds()),
		},
		createdAtField, req.Value.CreatedAt.UTC().Format(time.RFC3339),
		updatedAtField, req.Value.UpdatedAt.UTC().Format(time.RFC3339),
		userIDField, req.Value.UserID,
	).Err(); err != nil {
		return fmt.Errorf("hsetex: %w", err)
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
