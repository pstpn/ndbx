package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	sdto "ndbx/internal/service/dto"
	mdto "ndbx/internal/storage/mongodb/dto"
	nstorage "ndbx/internal/storage/neo4j"
	rstorage "ndbx/internal/storage/redis"
	rdto "ndbx/internal/storage/redis/dto"
	"ndbx/pkg/logger"
)

type GraphStorageInterface interface {
	CreateUser(ctx context.Context, userID string) error
	CreateEvent(ctx context.Context, eventID string, title string) error
	AddLike(ctx context.Context, userID string, eventID string, title string) error
	GetRecommendedEventIDs(ctx context.Context, userID string) ([]nstorage.RecommendedEvent, error)
}

type RecommendationCacheInterface interface {
	Get(ctx context.Context, req *rdto.GetRecommendationReq) (*rdto.GetRecommendationResp, error)
	Set(ctx context.Context, req *rdto.SetRecommendationReq) error
}

type RecommendationEventStorageInterface interface {
	GetEvent(ctx context.Context, req *mdto.GetEventReq) (*mdto.Event, error)
}

type RecommendationService struct {
	l                   logger.Interface
	graphStorage        GraphStorageInterface
	eventStorage        RecommendationEventStorageInterface
	recommendationCache RecommendationCacheInterface
	recommendationsTTL  time.Duration
}

func NewRecommendationService(
	l logger.Interface,
	graphStorage GraphStorageInterface,
	eventStorage RecommendationEventStorageInterface,
	recommendationCache RecommendationCacheInterface,
	recommendationsTTLSeconds int,
) *RecommendationService {
	return &RecommendationService{
		l:                   l,
		graphStorage:        graphStorage,
		eventStorage:        eventStorage,
		recommendationCache: recommendationCache,
		recommendationsTTL:  time.Duration(recommendationsTTLSeconds) * time.Second,
	}
}

func (s *RecommendationService) GetRecommendations(ctx context.Context, req *sdto.GetRecommendationsReq) (*sdto.GetRecommendationsResp, error) {
	cached, err := s.recommendationCache.Get(ctx, &rdto.GetRecommendationReq{UserID: req.UserID})
	if err == nil {
		events := make([]sdto.EventData, len(cached.Events))
		for i, e := range cached.Events {
			events[i] = sdto.EventData{
				ID:          e.ID,
				Title:       e.Title,
				Category:    e.Category,
				Price:       e.Price,
				Description: e.Description,
				Location: sdto.EventLocation{
					Address: e.Address,
					City:    e.City,
				},
				CreatedBy:  e.CreatedBy,
				StartedAt:  mustParseTime(e.StartedAt),
				FinishedAt: mustParseTime(e.FinishedAt),
			}
			if e.CreatedAt != "" {
				events[i].CreatedAt = mustParseTime(e.CreatedAt)
			}
		}
		return &sdto.GetRecommendationsResp{Events: events}, nil
	}
	if !rstorage.IsRecommendationNotFound(err) {
		return nil, fmt.Errorf("get recommendations from cache: %w", err)
	}

	recommended, err := s.graphStorage.GetRecommendedEventIDs(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get recommended event ids: %w", err)
	}

	if len(recommended) == 0 {
		resp := &sdto.GetRecommendationsResp{Events: []sdto.EventData{}}
		if cacheErr := s.recommendationCache.Set(ctx, &rdto.SetRecommendationReq{
			UserID: req.UserID,
			Events: []rdto.RecommendationEvent{},
			TTL:    s.recommendationsTTL,
		}); cacheErr != nil {
			s.l.Errorf("failed to cache empty recommendations: %s", cacheErr.Error())
		}
		return resp, nil
	}

	events := make([]sdto.EventData, 0, len(recommended))
	for _, rec := range recommended {
		event, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: rec.ID})
		if err != nil {
			s.l.Errorf("failed to get event %s for recommendations: %s", rec.ID, err.Error())
			continue
		}
		events = append(events, mapEventData(event))
	}

	events = deduplicateByTitle(events)
	titleScore := make(map[string]int64, len(recommended))
	for _, rec := range recommended {
		titleScore[rec.Title] += rec.Score
	}
	sort.SliceStable(events, func(i, j int) bool {
		return titleScore[events[i].Title] > titleScore[events[j].Title]
	})

	cacheEvents := make([]rdto.RecommendationEvent, len(events))
	for i, e := range events {
		cacheEvents[i] = rdto.RecommendationEvent{
			ID:          e.ID,
			Title:       e.Title,
			Category:    e.Category,
			Price:       e.Price,
			Description: e.Description,
			City:        e.Location.City,
			Address:     e.Location.Address,
			CreatedAt:   e.CreatedAt.Format(time.RFC3339),
			CreatedBy:   e.CreatedBy,
			StartedAt:   e.StartedAt.Format(time.RFC3339),
			FinishedAt:  e.FinishedAt.Format(time.RFC3339),
		}
	}
	if cacheErr := s.recommendationCache.Set(ctx, &rdto.SetRecommendationReq{
		UserID: req.UserID,
		Events: cacheEvents,
		TTL:    s.recommendationsTTL,
	}); cacheErr != nil {
		s.l.Errorf("failed to cache recommendations: %s", cacheErr.Error())
	}

	return &sdto.GetRecommendationsResp{Events: events}, nil
}

func deduplicateByTitle(events []sdto.EventData) []sdto.EventData {
	seen := make(map[string]int, len(events)) // title -> index in result
	result := make([]sdto.EventData, 0, len(events))

	for _, event := range events {
		if idx, ok := seen[event.Title]; ok {
			if event.StartedAt.Before(result[idx].StartedAt) {
				result[idx] = event
			}
			continue
		}
		seen[event.Title] = len(result)
		result = append(result, event)
	}

	return result
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
