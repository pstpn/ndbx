package cassandra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

const (
	reviewsTable = "event_reviews"
)

type EventReviewStorage struct {
	session *gocql.Session
}

func NewEventReviewStorage(session *gocql.Session) *EventReviewStorage {
	return &EventReviewStorage{session: session}
}

type Review struct {
	ID        string
	EventID   string
	Rating    int8
	Comment   string
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
}

type ReviewStats struct {
	Count int64
	Sum   int64
}

func (s *EventReviewStorage) CreateReview(
	ctx context.Context, eventID string, userID string, rating int8, comment string, createdAt time.Time,
) (string, error) {
	var existingID string
	if err := s.session.Query(
		"SELECT id FROM "+reviewsTable+" WHERE event_id = ? AND created_by = ?",
		eventID, userID,
	).WithContext(ctx).Consistency(gocql.One).Scan(&existingID); err == nil {
		return "", ErrReviewAlreadyExists
	}

	id := uuid.New().String()
	now := createdAt.UTC()

	if err := s.session.Query(
		"INSERT INTO "+reviewsTable+" (event_id, created_by, id, rating, comment, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		eventID, userID, id, rating, comment, now, now,
	).WithContext(ctx).Exec(); err != nil {
		return "", fmt.Errorf("insert review: %w", err)
	}

	return id, nil
}

func (s *EventReviewStorage) GetReviewsByEventID(ctx context.Context, eventID string) ([]Review, error) {
	iter := s.session.Query(
		"SELECT id, event_id, rating, comment, created_at, created_by, updated_at FROM "+reviewsTable+" WHERE event_id = ?",
		eventID,
	).WithContext(ctx).Iter()

	var reviews []Review
	for {
		var r Review
		r.EventID = eventID
		if !iter.Scan(&r.ID, &r.EventID, &r.Rating, &r.Comment, &r.CreatedAt, &r.CreatedBy, &r.UpdatedAt) {
			break
		}
		reviews = append(reviews, r)
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("iterate reviews by event id: %w", err)
	}

	return reviews, nil
}

func (s *EventReviewStorage) GetReview(ctx context.Context, eventID string, reviewID string) (*Review, error) {
	iter := s.session.Query(
		"SELECT id, event_id, rating, comment, created_at, created_by, updated_at FROM "+reviewsTable+" WHERE event_id = ?",
		eventID,
	).WithContext(ctx).Iter()

	for {
		var r Review
		r.EventID = eventID
		if !iter.Scan(&r.ID, &r.EventID, &r.Rating, &r.Comment, &r.CreatedAt, &r.CreatedBy, &r.UpdatedAt) {
			break
		}
		if r.ID == reviewID {
			if err := iter.Close(); err != nil {
				return nil, fmt.Errorf("close iterator after finding review: %w", err)
			}
			return &r, nil
		}
	}
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("iterate reviews: %w", err)
	}

	return nil, ErrReviewNotFound
}

func (s *EventReviewStorage) UpdateReview(ctx context.Context, eventID string, userID string, rating *int8, comment *string, updatedAt time.Time) error {
	var existing Review
	if err := s.session.Query(
		"SELECT id, rating, comment, created_at FROM "+reviewsTable+" WHERE event_id = ? AND created_by = ?",
		eventID, userID,
	).WithContext(ctx).Consistency(gocql.One).Scan(&existing.ID, &existing.Rating, &existing.Comment, &existing.CreatedAt); err != nil {
		if errors.Is(err, gocql.ErrNotFound) {
			return ErrReviewNotFound
		}
		return fmt.Errorf("get review for update: %w", err)
	}

	newRating := existing.Rating
	if rating != nil {
		newRating = *rating
	}
	newComment := existing.Comment
	if comment != nil {
		newComment = *comment
	}

	if err := s.session.Query(
		"INSERT INTO "+reviewsTable+" (event_id, created_by, id, rating, comment, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		eventID, userID, existing.ID, newRating, newComment, existing.CreatedAt, updatedAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("update review: %w", err)
	}

	return nil
}

func (s *EventReviewStorage) GetReviewStatsByEventIDs(ctx context.Context, eventIDs []string) (ReviewStats, error) {
	stats := ReviewStats{}

	for _, eventID := range eventIDs {
		iter := s.session.Query(
			"SELECT rating FROM "+reviewsTable+" WHERE event_id = ?",
			eventID,
		).WithContext(ctx).Iter()

		var rating int8
		for iter.Scan(&rating) {
			stats.Count++
			stats.Sum += int64(rating)
		}
		if err := iter.Close(); err != nil {
			return ReviewStats{}, fmt.Errorf("iterate reviews stats: %w", err)
		}
	}

	return stats, nil
}
