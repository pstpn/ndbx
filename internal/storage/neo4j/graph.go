package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type RecommendedEvent struct {
	ID    string
	Title string
	Score int64
}

type GraphStorage struct {
	driver neo4j.DriverWithContext
}

func NewGraphStorage(driver neo4j.DriverWithContext) *GraphStorage {
	return &GraphStorage{driver: driver}
}

func (s *GraphStorage) CreateUser(ctx context.Context, userID string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			"MERGE (u:User {id: $userId})",
			map[string]any{"userId": userID},
		)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("create user in neo4j: %w", err)
	}

	return nil
}

func (s *GraphStorage) CreateEvent(ctx context.Context, eventID string, title string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			"MERGE (e:Event {id: $eventId}) SET e.title = $title",
			map[string]any{"eventId": eventID, "title": title},
		)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("create event in neo4j: %w", err)
	}

	return nil
}

func (s *GraphStorage) AddLike(ctx context.Context, userID string, eventID string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			"MATCH (u:User {id: $userId}) MATCH (e:Event {id: $eventId}) MERGE (u)-[:LIKED]->(e)",
			map[string]any{"userId": userID, "eventId": eventID},
		)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("add like in neo4j: %w", err)
	}

	return nil
}

func (s *GraphStorage) GetRecommendedEventIDs(ctx context.Context, userID string) ([]RecommendedEvent, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx,
			`MATCH (u:User {id: $userId})-[:LIKED]->(e:Event)<-[:LIKED]-(other:User)
			 WHERE other <> u
			 MATCH (other)-[:LIKED]->(rec:Event)
			 WHERE NOT (u)-[:LIKED]->(rec)
			 RETURN rec.id AS id, rec.title AS title, count(*) AS score
			 ORDER BY score DESC`,
			map[string]any{"userId": userID},
		)
		if err != nil {
			return nil, err
		}

		var recommendations []RecommendedEvent
		for records.Next(ctx) {
			rec, ok := parseRecommendationRecord(records.Record())
			if !ok {
				continue
			}
			recommendations = append(recommendations, rec)
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		return recommendations, nil
	})
	if err != nil {
		return nil, fmt.Errorf("get recommended event ids from neo4j: %w", err)
	}

	recommendations, ok := result.([]RecommendedEvent)
	if !ok {
		return nil, nil
	}

	return recommendations, nil
}

func parseRecommendationRecord(record *neo4j.Record) (RecommendedEvent, bool) {
	idVal, ok := record.Get("id")
	if !ok || idVal == nil {
		return RecommendedEvent{}, false
	}
	id, ok := idVal.(string)
	if !ok {
		return RecommendedEvent{}, false
	}

	titleVal, ok := record.Get("title")
	if !ok || titleVal == nil {
		return RecommendedEvent{}, false
	}
	title, ok := titleVal.(string)
	if !ok {
		return RecommendedEvent{}, false
	}

	scoreVal, ok := record.Get("score")
	if !ok || scoreVal == nil {
		return RecommendedEvent{}, false
	}
	score, ok := scoreVal.(int64)
	if !ok {
		return RecommendedEvent{}, false
	}

	return RecommendedEvent{ID: id, Title: title, Score: score}, true
}
