package cassandra

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

const (
	reactionsTable = "event_reactions"
)

type EventReactionStorage struct {
	session *gocql.Session
}

type ReactionCounts struct {
	Likes    int64
	Dislikes int64
}

func NewEventReactionStorage(session *gocql.Session) *EventReactionStorage {
	return &EventReactionStorage{session: session}
}

func (s *EventReactionStorage) EnsureSchema(ctx context.Context) error {
	queries := []string{
		"CREATE TABLE IF NOT EXISTS event_reactions " +
			"(event_id text, created_by text, like_value tinyint, created_at timestamp, PRIMARY KEY ((event_id), created_by))",
		"CREATE INDEX IF NOT EXISTS event_reactions_like_value_idx ON event_reactions (like_value)",
		"CREATE INDEX IF NOT EXISTS event_reactions_created_by_idx ON event_reactions (created_by)",
	}

	for _, q := range queries {
		if err := s.session.Query(q).WithContext(ctx).Exec(); err != nil {
			return fmt.Errorf("execute cassandra schema query: %w", err)
		}
	}

	return nil
}

func (s *EventReactionStorage) UpsertReaction(ctx context.Context, eventID string, userID string, likeValue int8, createdAt time.Time) error {
	if err := s.session.Query(
		"INSERT INTO "+reactionsTable+" (event_id, created_by, like_value, created_at) VALUES (?, ?, ?, ?)",
		eventID,
		userID,
		likeValue,
		createdAt.UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("upsert reaction: %w", err)
	}

	return nil
}

func (s *EventReactionStorage) CountByEventIDs(ctx context.Context, eventIDs []string) (ReactionCounts, error) {
	counts := ReactionCounts{}

	for _, eventID := range eventIDs {
		iter := s.session.Query(
			"SELECT like_value FROM "+reactionsTable+" WHERE event_id = ?",
			eventID,
		).WithContext(ctx).Iter()

		var likeValue int8
		for iter.Scan(&likeValue) {
			switch likeValue {
			case 1:
				counts.Likes++
			case -1:
				counts.Dislikes++
			}
		}
		if err := iter.Close(); err != nil {
			return ReactionCounts{}, fmt.Errorf("iterate reactions by event id: %w", err)
		}
	}

	return counts, nil
}
