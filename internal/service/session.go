package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"ndbx/internal/service/dto"
	rdto "ndbx/internal/storage/redis/dto"
	"ndbx/pkg/cryptic"
	"ndbx/pkg/logger"
)

type SessionStorage interface {
	Get(ctx context.Context, req *rdto.GetReq) (*rdto.GetResp, error)
	Set(ctx context.Context, req *rdto.SetReq) error
	SetOrUpdate(ctx context.Context, req *rdto.SetOrUpdateReq) (*rdto.SetOrUpdateResp, error)
	Delete(ctx context.Context, req *rdto.DeleteReq) error
}

type SessionService struct {
	l                 logger.Interface
	sessionStorage    SessionStorage
	sessionTTLSeconds int
}

func NewSessionService(l logger.Interface, s SessionStorage, sessionTTLSeconds int) *SessionService {
	return &SessionService{
		l:                 l,
		sessionStorage:    s,
		sessionTTLSeconds: sessionTTLSeconds,
	}
}

func (s *SessionService) GetSession(ctx context.Context, req *dto.GetSessionReq) (*dto.GetSessionResp, error) {
	session, err := s.sessionStorage.Get(ctx, &rdto.GetReq{SID: req.SID})
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	return (*dto.GetSessionResp)(&session.SessionValue), nil
}

func (s *SessionService) CreateSession(ctx context.Context, req *dto.CreateSessionReq) (*dto.CreateSessionResp, error) {
	sid := cryptic.MustSID()
	now := time.Now().UTC()

	if err := s.sessionStorage.Set(ctx, &rdto.SetReq{
		SID: sid,
		Value: rdto.SessionValue{
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    req.UserID,
		},
		TTL: time.Duration(s.sessionTTLSeconds) * time.Second,
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &dto.CreateSessionResp{SID: sid, TTL: time.Duration(s.sessionTTLSeconds) * time.Second}, nil
}

func (s *SessionService) CreateOrExtendSession(ctx context.Context, req *dto.CreateOrExtendSessionReq) (*dto.CreateOrExtendSessionResp, error) {
	newSID := cryptic.MustSID()
	now := time.Now().UTC()

	res, err := s.sessionStorage.SetOrUpdate(ctx, &rdto.SetOrUpdateReq{
		SID:    req.SID,
		NewSID: newSID,
		NewValue: rdto.SessionValue{
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    req.UserID,
		},
		TTL: time.Duration(s.sessionTTLSeconds) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create or extend session: %w", err)
	}

	return &dto.CreateOrExtendSessionResp{
		SID:       res.SID,
		TTL:       time.Duration(s.sessionTTLSeconds) * time.Second,
		IsCreated: res.IsCreated,
	}, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, req *dto.DeleteSessionReq) error {
	if err := s.sessionStorage.Delete(ctx, &rdto.DeleteReq{SID: req.SID}); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
