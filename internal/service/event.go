package service

import (
	"context"
	"crypto/md5" //nolint:gosec // md5 explicitly required by the task for cache key
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	sdto "ndbx/internal/service/dto"
	cstorage "ndbx/internal/storage/cassandra"
	"ndbx/internal/storage/mongodb"
	mdto "ndbx/internal/storage/mongodb/dto"
	rstorage "ndbx/internal/storage/redis"
	rdto "ndbx/internal/storage/redis/dto"
	"ndbx/pkg/logger"
)

const (
	likeValue    int8 = 1
	dislikeValue int8 = -1
)

type EventStorageInterface interface {
	Create(ctx context.Context, req *mdto.CreateEventReq) (*mdto.Event, error)
	GetEvents(ctx context.Context, req *mdto.GetEventsReq) (*mdto.GetEventsResp, error)
	GetEvent(ctx context.Context, req *mdto.GetEventReq) (*mdto.Event, error)
	GetEventsByTitle(ctx context.Context, req *mdto.GetEventsByTitleReq) (*mdto.GetEventsResp, error)
	PatchEvent(ctx context.Context, req *mdto.PatchEventReq) error
}

type EventReactionStorageInterface interface {
	UpsertReaction(ctx context.Context, eventID string, userID string, likeValue int8, createdAt time.Time) error
	CountByEventIDs(ctx context.Context, eventIDs []string) (cstorage.ReactionCounts, error)
}

type EventReactionCacheInterface interface {
	Get(ctx context.Context, req *rdto.GetReactionsReq) (*rdto.GetReactionsResp, error)
	Set(ctx context.Context, req *rdto.SetReactionsReq) error
}

type ReviewServiceInterface interface {
	GetReviewsByTitle(ctx context.Context, title string) (sdto.EventReviewsSummary, error)
}

type EventService struct {
	l               logger.Interface
	eventStorage    EventStorageInterface
	reactionStorage EventReactionStorageInterface
	reactionCache   EventReactionCacheInterface
	reviewService   ReviewServiceInterface
	graphStorage    GraphStorageInterface
	reactionTTL     time.Duration
}

func NewEventService(
	l logger.Interface,
	eventStorage EventStorageInterface,
	reactionStorage EventReactionStorageInterface,
	reactionCache EventReactionCacheInterface,
	reviewService ReviewServiceInterface,
	graphStorage GraphStorageInterface,
	reactionTTLSeconds int,
) *EventService {
	return &EventService{
		l:               l,
		eventStorage:    eventStorage,
		reactionStorage: reactionStorage,
		reactionCache:   reactionCache,
		reviewService:   reviewService,
		graphStorage:    graphStorage,
		reactionTTL:     time.Duration(reactionTTLSeconds) * time.Second,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, req *sdto.CreateEventReq) (*sdto.CreateEventResp, error) {
	event, err := s.eventStorage.Create(ctx, &mdto.CreateEventReq{
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

	if err := s.graphStorage.CreateEvent(ctx, event.ID, event.Title); err != nil {
		s.l.Errorf("failed to create event in neo4j: %s", err.Error())
	}

	return &sdto.CreateEventResp{ID: event.ID}, nil
}

func (s *EventService) GetEvents(ctx context.Context, req *sdto.GetEventsReq) (*sdto.GetEventsResp, error) {
	resp, err := s.eventStorage.GetEvents(ctx, &mdto.GetEventsReq{
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
		event := e
		events[i] = mapEventData(&event)
	}

	if req.IncludeReactions {
		reactionsByTitle, err := s.reactionsByTitles(ctx, events)
		if err != nil {
			return nil, fmt.Errorf("get reactions by titles: %w", err)
		}

		for i := range events {
			events[i].Reactions = reactionsByTitle[events[i].Title]
		}
	}

	if req.IncludeReviews {
		reviewsByTitle, err := s.reviewsByTitles(ctx, events)
		if err != nil {
			return nil, fmt.Errorf("get reviews by titles: %w", err)
		}

		for i := range events {
			events[i].Reviews = reviewsByTitle[events[i].Title]
		}
	}

	return &sdto.GetEventsResp{Events: events}, nil
}

func (s *EventService) GetEvent(ctx context.Context, req *sdto.GetEventReq) (*sdto.GetEventResp, error) {
	event, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: req.ID})
	if err != nil {
		if errors.Is(err, mongodb.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get event: %w", err)
	}

	resp := &sdto.GetEventResp{Event: mapEventData(event)}
	if req.IncludeReactions {
		reactions, err := s.reactionsByTitle(ctx, event.Title)
		if err != nil {
			return nil, fmt.Errorf("get reactions by title: %w", err)
		}
		resp.Event.Reactions = reactions
	}

	if req.IncludeReviews {
		reviews, err := s.reviewService.GetReviewsByTitle(ctx, event.Title)
		if err != nil {
			return nil, fmt.Errorf("get reviews by title: %w", err)
		}
		resp.Event.Reviews = reviews
	}

	return resp, nil
}

func (s *EventService) PatchEvent(ctx context.Context, req *sdto.PatchEventReq) error {
	err := s.eventStorage.PatchEvent(ctx, &mdto.PatchEventReq{
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

func (s *EventService) LikeEvent(ctx context.Context, req *sdto.ReactEventReq) error {
	return s.reactToEvent(ctx, req, likeValue)
}

func (s *EventService) DislikeEvent(ctx context.Context, req *sdto.ReactEventReq) error {
	return s.reactToEvent(ctx, req, dislikeValue)
}

func (s *EventService) reactToEvent(ctx context.Context, req *sdto.ReactEventReq, value int8) error {
	event, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: req.ID})
	if err != nil {
		if errors.Is(err, mongodb.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("get event before reaction: %w", err)
	}

	if err := s.reactionStorage.UpsertReaction(ctx, req.ID, req.UserID, value, time.Now().UTC()); err != nil {
		return fmt.Errorf("upsert reaction: %w", err)
	}

	if value == likeValue {
		if err := s.graphStorage.AddLike(ctx, req.UserID, req.ID, event.Title); err != nil {
			s.l.Errorf("failed to add like in neo4j: %s", err.Error())
		}
	}

	reactions, err := s.reactionsByTitleNoCache(ctx, event.Title)
	if err != nil {
		return fmt.Errorf("calculate title reactions after reaction update: %w", err)
	}

	if err := s.reactionCache.Set(ctx, &rdto.SetReactionsReq{
		TitleHash: titleMD5(event.Title),
		Reactions: rdto.Reactions{Likes: reactions.Likes, Dislikes: reactions.Dislikes},
		TTL:       s.reactionTTL,
	}); err != nil {
		return fmt.Errorf("set reactions cache after reaction update: %w", err)
	}

	return nil
}

func (s *EventService) reactionsByTitles(ctx context.Context, events []sdto.EventData) (map[string]sdto.EventReactions, error) {
	reactionsByTitle := make(map[string]sdto.EventReactions, len(events))
	for _, event := range events {
		if _, ok := reactionsByTitle[event.Title]; ok {
			continue
		}

		reactions, err := s.reactionsByTitle(ctx, event.Title)
		if err != nil {
			return nil, err
		}
		reactionsByTitle[event.Title] = reactions
	}

	return reactionsByTitle, nil
}

func (s *EventService) reviewsByTitles(ctx context.Context, events []sdto.EventData) (map[string]sdto.EventReviewsSummary, error) {
	reviewsByTitle := make(map[string]sdto.EventReviewsSummary, len(events))
	for _, event := range events {
		if _, ok := reviewsByTitle[event.Title]; ok {
			continue
		}

		reviews, err := s.reviewService.GetReviewsByTitle(ctx, event.Title)
		if err != nil {
			return nil, err
		}
		reviewsByTitle[event.Title] = reviews
	}

	return reviewsByTitle, nil
}

func (s *EventService) reactionsByTitle(ctx context.Context, title string) (sdto.EventReactions, error) {
	titleHash := titleMD5(title)

	cached, err := s.reactionCache.Get(ctx, &rdto.GetReactionsReq{TitleHash: titleHash})
	if err == nil {
		return sdto.EventReactions{Likes: cached.Reactions.Likes, Dislikes: cached.Reactions.Dislikes}, nil
	}
	if !rstorage.IsNotFound(err) {
		return sdto.EventReactions{}, fmt.Errorf("get reactions from cache: %w", err)
	}

	reactions, err := s.reactionsByTitleNoCache(ctx, title)
	if err != nil {
		return sdto.EventReactions{}, err
	}

	if reactions.Likes > 0 || reactions.Dislikes > 0 {
		if err := s.reactionCache.Set(ctx, &rdto.SetReactionsReq{
			TitleHash: titleHash,
			Reactions: rdto.Reactions{Likes: reactions.Likes, Dislikes: reactions.Dislikes},
			TTL:       s.reactionTTL,
		}); err != nil {
			return sdto.EventReactions{}, fmt.Errorf("set reactions to cache: %w", err)
		}
	}

	return reactions, nil
}

func (s *EventService) reactionsByTitleNoCache(ctx context.Context, title string) (sdto.EventReactions, error) {
	eventsByTitle, err := s.eventStorage.GetEventsByTitle(ctx, &mdto.GetEventsByTitleReq{Title: title})
	if err != nil {
		return sdto.EventReactions{}, fmt.Errorf("get events by title: %w", err)
	}

	eventIDs := make([]string, 0, len(eventsByTitle.Events))
	for _, event := range eventsByTitle.Events {
		eventIDs = append(eventIDs, event.ID)
	}

	counts, err := s.reactionStorage.CountByEventIDs(ctx, eventIDs)
	if err != nil {
		return sdto.EventReactions{}, fmt.Errorf("count reactions by event ids: %w", err)
	}

	return sdto.EventReactions{Likes: counts.Likes, Dislikes: counts.Dislikes}, nil
}

func mapEventData(event *mdto.Event) sdto.EventData {
	return sdto.EventData{
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
	}
}

func titleMD5(title string) string {
	//nolint:gosec // md5 is required by the task for cache key format compatibility
	hash := md5.Sum([]byte(title))
	return hex.EncodeToString(hash[:])
}
