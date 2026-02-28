package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ogen-go/ogen/json"
	"github.com/redis/go-redis/v9"

	"ndbx/internal/storage/redis/dto"
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

	var oldValue dto.SessionValue
	if err := s.client.Get(ctx, sessionKey(req.SID)).Scan(&oldValue); err != nil {
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

		return nil, fmt.Errorf("get: %w", err)
	}

	if err := s.setValue(ctx, &dto.SetReq{
		SID:   req.SID,
		Value: dto.SessionValue{CreatedAt: oldValue.CreatedAt, UpdatedAt: req.NewValue.UpdatedAt},
		TTL:   req.TTL,
	}); err != nil {
		return nil, err
	}

	return &dto.SetOrUpdateResp{SID: req.SID, IsCreated: false}, nil
}

func (s *SessionStorage) Get(ctx context.Context, req *dto.GetReq) (*dto.GetResp, error) {
	var value dto.SessionValue
	if err := s.client.Get(ctx, sessionKey(req.SID)).Scan(&value); err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	return &dto.GetResp{SessionValue: value}, nil
}

func (s *SessionStorage) setValue(ctx context.Context, req *dto.SetReq) error {
	rawData, err := json.Marshal(req.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	if err := s.client.Set(ctx, sessionKey(req.SID), rawData, req.TTL).Err(); err != nil {
		return fmt.Errorf("set: %w", err)
	}

	return nil
}

func sessionKey(sid string) string {
	return "sid:" + sid
}
