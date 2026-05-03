package service

import (
	"context"
	"crypto/md5" //nolint:gosec // md5 explicitly required by the task for cache key
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	sdto "ndbx/internal/service/dto"
	cstorage "ndbx/internal/storage/cassandra"
	mstorage "ndbx/internal/storage/mongodb"
	mdto "ndbx/internal/storage/mongodb/dto"
	rstorage "ndbx/internal/storage/redis"
	rdto "ndbx/internal/storage/redis/dto"
	"ndbx/pkg/logger"
)

type ReviewStorageInterface interface {
	CreateReview(ctx context.Context, eventID string, userID string, rating int8, comment string, createdAt time.Time) (string, error)
	GetReviewsByEventID(ctx context.Context, eventID string) ([]cstorage.Review, error)
	GetReview(ctx context.Context, eventID string, reviewID string) (*cstorage.Review, error)
	UpdateReview(ctx context.Context, eventID string, userID string, rating *int8, comment *string, updatedAt time.Time) error
	GetReviewStatsByEventIDs(ctx context.Context, eventIDs []string) (cstorage.ReviewStats, error)
}

type ReviewCacheInterface interface {
	Get(ctx context.Context, req *rdto.GetReviewsReq) (*rdto.GetReviewsResp, error)
	Set(ctx context.Context, req *rdto.SetReviewsReq) error
}

type ReviewService struct {
	l             logger.Interface
	eventStorage  EventStorageInterface
	reviewStorage ReviewStorageInterface
	reviewCache   ReviewCacheInterface
	reviewTTL     time.Duration
}

func NewReviewService(
	l logger.Interface,
	eventStorage EventStorageInterface,
	reviewStorage ReviewStorageInterface,
	reviewCache ReviewCacheInterface,
	reviewTTLSeconds int,
) *ReviewService {
	return &ReviewService{
		l:             l,
		eventStorage:  eventStorage,
		reviewStorage: reviewStorage,
		reviewCache:   reviewCache,
		reviewTTL:     time.Duration(reviewTTLSeconds) * time.Second,
	}
}

func (s *ReviewService) CreateReview(ctx context.Context, req *sdto.CreateReviewReq) (*sdto.CreateReviewResp, error) {
	_, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: req.EventID})
	if err != nil {
		if errors.Is(err, mstorage.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get event before review: %w", err)
	}

	id, err := s.reviewStorage.CreateReview(ctx, req.EventID, req.UserID, req.Rating, req.Comment, time.Now().UTC())
	if err != nil {
		if errors.Is(err, cstorage.ErrReviewAlreadyExists) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("create review: %w", err)
	}

	if err := s.updateReviewCache(ctx, req.EventID); err != nil {
		s.l.Errorf("failed to update review cache after create: %s", err.Error())
	}

	return &sdto.CreateReviewResp{ID: id}, nil
}

func (s *ReviewService) GetReviews(ctx context.Context, req *sdto.GetReviewsReq) (*sdto.GetReviewsResp, error) {
	_, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: req.EventID})
	if err != nil {
		if errors.Is(err, mstorage.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get event for reviews: %w", err)
	}

	reviews, err := s.reviewStorage.GetReviewsByEventID(ctx, req.EventID)
	if err != nil {
		return nil, fmt.Errorf("get reviews by event id: %w", err)
	}

	result := make([]sdto.ReviewData, 0, len(reviews))
	for _, r := range reviews {
		result = append(result, sdto.ReviewData{
			ID:        r.ID,
			EventID:   r.EventID,
			Rating:    r.Rating,
			Comment:   r.Comment,
			CreatedAt: r.CreatedAt,
			CreatedBy: r.CreatedBy,
			UpdatedAt: r.UpdatedAt,
		})
	}

	return &sdto.GetReviewsResp{Reviews: result, Count: int64(len(result))}, nil
}

func (s *ReviewService) UpdateReview(ctx context.Context, req *sdto.UpdateReviewReq) error {
	_, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: req.EventID})
	if err != nil {
		if errors.Is(err, mstorage.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("get event before review update: %w", err)
	}

	review, err := s.reviewStorage.GetReview(ctx, req.EventID, req.ReviewID)
	if err != nil {
		if errors.Is(err, cstorage.ErrReviewNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("get review for update: %w", err)
	}

	if review.CreatedBy != req.UserID {
		return ErrForbidden
	}

	if err := s.reviewStorage.UpdateReview(ctx, req.EventID, req.UserID, req.Rating, req.Comment, time.Now().UTC()); err != nil {
		if errors.Is(err, cstorage.ErrReviewNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("update review: %w", err)
	}

	if err := s.updateReviewCache(ctx, req.EventID); err != nil {
		s.l.Errorf("failed to update review cache after update: %s", err.Error())
	}

	return nil
}

func (s *ReviewService) GetReviewsByTitle(ctx context.Context, title string) (sdto.EventReviewsSummary, error) {
	titleHash := reviewTitleMD5(title)

	cached, err := s.reviewCache.Get(ctx, &rdto.GetReviewsReq{TitleHash: titleHash})
	if err == nil {
		return sdto.EventReviewsSummary{Count: cached.Reviews.Count, Rating: cached.Reviews.Rating}, nil
	}
	if !rstorage.IsReviewNotFound(err) && !rstorage.IsNotFound(err) {
		return sdto.EventReviewsSummary{}, fmt.Errorf("get reviews from cache: %w", err)
	}

	summary, err := s.reviewsByTitleNoCache(ctx, title)
	if err != nil {
		return sdto.EventReviewsSummary{}, err
	}

	if summary.Count > 0 {
		if err := s.reviewCache.Set(ctx, &rdto.SetReviewsReq{
			TitleHash: titleHash,
			Reviews:   rdto.Reviews{Count: summary.Count, Rating: summary.Rating},
			TTL:       s.reviewTTL,
		}); err != nil {
			return sdto.EventReviewsSummary{}, fmt.Errorf("set reviews to cache: %w", err)
		}
	}

	return summary, nil
}

func (s *ReviewService) reviewsByTitleNoCache(ctx context.Context, title string) (sdto.EventReviewsSummary, error) {
	eventsByTitle, err := s.eventStorage.GetEventsByTitle(ctx, &mdto.GetEventsByTitleReq{Title: title})
	if err != nil {
		return sdto.EventReviewsSummary{}, fmt.Errorf("get events by title: %w", err)
	}

	eventIDs := make([]string, 0, len(eventsByTitle.Events))
	for _, event := range eventsByTitle.Events {
		eventIDs = append(eventIDs, event.ID)
	}

	stats, err := s.reviewStorage.GetReviewStatsByEventIDs(ctx, eventIDs)
	if err != nil {
		return sdto.EventReviewsSummary{}, fmt.Errorf("get review stats: %w", err)
	}

	var avgRating float64
	if stats.Count > 0 {
		avgRating = math.Round(float64(stats.Sum)/float64(stats.Count)*10) / 10 //nolint:mnd // rounding to 1 decimal place
	}

	return sdto.EventReviewsSummary{Count: stats.Count, Rating: avgRating}, nil
}

func (s *ReviewService) updateReviewCache(ctx context.Context, eventID string) error {
	event, err := s.eventStorage.GetEvent(ctx, &mdto.GetEventReq{ID: eventID})
	if err != nil {
		return fmt.Errorf("get event for review cache update: %w", err)
	}

	summary, err := s.reviewsByTitleNoCache(ctx, event.Title)
	if err != nil {
		return fmt.Errorf("calculate reviews summary: %w", err)
	}

	if err := s.reviewCache.Set(ctx, &rdto.SetReviewsReq{
		TitleHash: reviewTitleMD5(event.Title),
		Reviews:   rdto.Reviews{Count: summary.Count, Rating: summary.Rating},
		TTL:       s.reviewTTL,
	}); err != nil {
		return fmt.Errorf("set reviews cache: %w", err)
	}

	return nil
}

func reviewTitleMD5(title string) string {
	//nolint:gosec // md5 is required by the task for cache key format compatibility
	hash := md5.Sum([]byte(title))
	return hex.EncodeToString(hash[:])
}
