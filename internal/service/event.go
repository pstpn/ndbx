package service

import (
	"context"
	"errors"
	"fmt"

	sdto "ndbx/internal/service/dto"
	"ndbx/internal/storage/mongodb"
	mdto "ndbx/internal/storage/mongodb/dto"
	"ndbx/pkg/logger"
)

type EventStorageInterface interface {
	Create(ctx context.Context, req *mdto.CreateEventReq) (*mdto.Event, error)
	GetEvents(ctx context.Context, req *mdto.GetEventsReq) (*mdto.GetEventsResp, error)
}

type EventService struct {
	l       logger.Interface
	storage EventStorageInterface
}

func NewEventService(l logger.Interface, storage EventStorageInterface) *EventService {
	return &EventService{
		l:       l,
		storage: storage,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, req *sdto.CreateEventReq) (*sdto.CreateEventResp, error) {
	event, err := s.storage.Create(ctx, &mdto.CreateEventReq{
		Title:       req.Title,
		Description: req.Description,
		Address:     req.Address,
		StartedAt:   req.StartedAt,
		FinishedAt:  req.FinishedAt,
		CreatedBy:   req.CreatedBy,
	})
	if err != nil {
		if errors.Is(err, mongodb.ErrAlreadyExists) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("create event: %w", err)
	}

	return &sdto.CreateEventResp{ID: event.ID}, nil
}

func (s *EventService) GetEvents(ctx context.Context, req *sdto.GetEventsReq) (*sdto.GetEventsResp, error) {
	resp, err := s.storage.GetEvents(ctx, &mdto.GetEventsReq{
		Title:  req.Title,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	events := make([]sdto.EventData, len(resp.Events))
	for i, e := range resp.Events {
		events[i] = sdto.EventData{
			ID:          e.ID,
			Title:       e.Title,
			Description: e.Description,
			Location: sdto.EventLocation{
				Address: e.Location.Address,
			},
			CreatedAt:  e.CreatedAt,
			CreatedBy:  e.CreatedBy,
			StartedAt:  e.StartedAt,
			FinishedAt: e.FinishedAt,
		}
	}

	return &sdto.GetEventsResp{Events: events}, nil
}
