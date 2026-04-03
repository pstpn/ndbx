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
	GetEvent(ctx context.Context, req *mdto.GetEventReq) (*mdto.Event, error)
	PatchEvent(ctx context.Context, req *mdto.PatchEventReq) error
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
		ID:        req.ID,
		Title:     req.Title,
		Category:  req.Category,
		PriceFrom: req.PriceFrom,
		PriceTo:   req.PriceTo,
		Address:   req.Address,
		City:      req.City,
		DateFrom:  req.DateFrom,
		DateTo:    req.DateTo,
		UserID:    req.UserID,
		User:      req.User,
		Limit:     req.Limit,
		Offset:    req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	events := make([]sdto.EventData, len(resp.Events))
	for i, e := range resp.Events {
		events[i] = sdto.EventData{
			ID:          e.ID,
			Title:       e.Title,
			Category:    e.Category,
			Price:       e.Price,
			Description: e.Description,
			Location: sdto.EventLocation{
				Address: e.Location.Address,
				City:    e.Location.City,
			},
			CreatedAt:  e.CreatedAt,
			CreatedBy:  e.CreatedBy,
			StartedAt:  e.StartedAt,
			FinishedAt: e.FinishedAt,
		}
	}

	return &sdto.GetEventsResp{Events: events}, nil
}

func (s *EventService) GetEvent(ctx context.Context, req *sdto.GetEventReq) (*sdto.GetEventResp, error) {
	event, err := s.storage.GetEvent(ctx, &mdto.GetEventReq{ID: req.ID})
	if err != nil {
		if errors.Is(err, mongodb.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get event: %w", err)
	}

	return &sdto.GetEventResp{Event: sdto.EventData{
		ID:          event.ID,
		Title:       event.Title,
		Category:    event.Category,
		Price:       event.Price,
		Description: event.Description,
		Location: sdto.EventLocation{
			Address: event.Location.Address,
			City:    event.Location.City,
		},
		CreatedAt:  event.CreatedAt,
		CreatedBy:  event.CreatedBy,
		StartedAt:  event.StartedAt,
		FinishedAt: event.FinishedAt,
	}}, nil
}

func (s *EventService) PatchEvent(ctx context.Context, req *sdto.PatchEventReq) error {
	err := s.storage.PatchEvent(ctx, &mdto.PatchEventReq{
		ID:        req.ID,
		CreatedBy: req.CreatedBy,
		Category:  req.Category,
		City:      req.City,
		Price:     req.Price,
	})
	if err != nil {
		if errors.Is(err, mongodb.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("patch event: %w", err)
	}

	return nil
}
