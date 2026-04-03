package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"ndbx/internal/storage/mongodb/dto"
)

const collectionEvents = "events"

type EventStorage struct {
	db *mongo.Database
}

func NewEventStorage(db *mongo.Database) *EventStorage {
	return &EventStorage{db: db}
}

func (s *EventStorage) CreateIndexes(ctx context.Context) error {
	coll := s.db.Collection(collectionEvents)

	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "title", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "title", Value: 1},
				{Key: "created_by", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "created_by", Value: 1},
				{Key: "title", Value: 1},
			},
			Options: options.Index().
				SetName("uniq_created_by_title_valid").
				SetUnique(true).
				SetPartialFilterExpression(bson.D{
					{Key: "created_by", Value: bson.D{{Key: "$type", Value: "string"}}},
					{Key: "title", Value: bson.D{{Key: "$type", Value: "string"}}},
				}),
		},
	}

	_, err := coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}

func (s *EventStorage) Create(ctx context.Context, req *dto.CreateEventReq) (*dto.Event, error) {
	coll := s.db.Collection(collectionEvents)
	now := time.Now().UTC()

	eventDoc := bson.M{
		"title":       req.Title,
		"description": req.Description,
		"location": bson.M{
			"address": req.Address,
		},
		"created_at":  now.Format(time.RFC3339),
		"started_at":  req.StartedAt.UTC().Format(time.RFC3339),
		"finished_at": req.FinishedAt.UTC().Format(time.RFC3339),
		"created_by":  req.CreatedBy,
	}

	result, err := coll.InsertOne(ctx, eventDoc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert event: %w", err)
	}

	event := &dto.Event{
		ID:          stringifyID(result.InsertedID),
		Title:       req.Title,
		Description: req.Description,
		Location: dto.Location{
			Address: req.Address,
		},
		CreatedAt:  now,
		StartedAt:  req.StartedAt,
		FinishedAt: req.FinishedAt,
		CreatedBy:  req.CreatedBy,
	}
	return event, nil
}

func (s *EventStorage) GetEvents(ctx context.Context, req *dto.GetEventsReq) (*dto.GetEventsResp, error) {
	coll := s.db.Collection(collectionEvents)

	filter, empty, err := buildEventsFilter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}
	if empty {
		return &dto.GetEventsResp{Events: []dto.Event{}}, nil
	}

	opts := &options.FindOptions{}
	if req.Limit > 0 {
		opts.SetLimit(req.Limit)
	}
	if req.Offset >= 0 {
		opts.SetSkip(req.Offset)
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find events: %w", err)
	}
	defer cursor.Close(ctx)

	events := make([]dto.Event, 0)
	for cursor.Next(ctx) {
		var eventDoc bson.M
		if err := cursor.Decode(&eventDoc); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		events = append(events, mapEvent(eventDoc))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return &dto.GetEventsResp{Events: events}, nil
}

func buildEventsFilter(ctx context.Context, db *mongo.Database, req *dto.GetEventsReq) (bson.M, bool, error) {
	filter := bson.M{}
	applyEventIDFilter(filter, req.ID)
	applyEventTitleFilter(filter, req.Title)
	applyEventCategoryFilter(filter, req.Category)
	applyEventPriceFilter(filter, req.PriceFrom, req.PriceTo)
	applyEventAddressFilter(filter, req.Address)
	applyEventCityFilter(filter, req.City)
	applyEventDateFilter(filter, req.DateFrom, req.DateTo)
	applyEventUserIDFilter(filter, req.UserID)

	if req.User != "" {
		createdBy, found, err := resolveEventUser(ctx, db, req.User)
		if err != nil {
			return nil, false, err
		}
		if !found {
			return nil, true, nil
		}
		filter["created_by"] = createdBy
	}

	return filter, false, nil
}

func applyEventIDFilter(filter bson.M, id string) {
	if id != "" {
		filter["_id"] = bson.M{"$in": idAlternatives(id)}
	}
}

func applyEventTitleFilter(filter bson.M, title string) {
	if title != "" {
		filter["title"] = primitive.Regex{Pattern: title, Options: "i"}
	}
}

func applyEventCategoryFilter(filter bson.M, category string) {
	if category != "" {
		filter["category"] = category
	}
}

func applyEventPriceFilter(filter bson.M, priceFrom *int64, priceTo *int64) {
	if priceFrom == nil && priceTo == nil {
		return
	}

	priceFilter := bson.M{}
	if priceFrom != nil {
		priceFilter["$gte"] = *priceFrom
	}
	if priceTo != nil {
		priceFilter["$lte"] = *priceTo
	}
	filter["price"] = priceFilter
}

func applyEventAddressFilter(filter bson.M, address string) {
	if address != "" {
		filter["location.address"] = primitive.Regex{Pattern: address, Options: "i"}
	}
}

func applyEventCityFilter(filter bson.M, city string) {
	if city != "" {
		filter["location.city"] = city
	}
}

func applyEventDateFilter(filter bson.M, dateFrom *time.Time, dateTo *time.Time) {
	if dateFrom == nil && dateTo == nil {
		return
	}

	dateFilter := bson.M{}
	if dateFrom != nil {
		dateFilter["$gte"] = dateFrom.UTC().Format(time.RFC3339)
	}
	if dateTo != nil {
		dateFilter["$lte"] = dateTo.UTC().Format(time.RFC3339)
	}
	filter["started_at"] = dateFilter
}

func applyEventUserIDFilter(filter bson.M, userID string) {
	if userID != "" {
		filter["created_by"] = userID
	}
}

func resolveEventUser(ctx context.Context, db *mongo.Database, username string) (string, bool, error) {
	var user bson.M
	err := db.Collection(collectionUsers).FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("find user by username: %w", err)
	}

	return stringifyID(user["_id"]), true, nil
}

func (s *EventStorage) GetEvent(ctx context.Context, req *dto.GetEventReq) (*dto.Event, error) {
	var eventDoc bson.M
	err := s.db.Collection(collectionEvents).FindOne(ctx, bson.M{"_id": bson.M{"$in": idAlternatives(req.ID)}}).Decode(&eventDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find event: %w", err)
	}

	event := mapEvent(eventDoc)
	return &event, nil
}

func (s *EventStorage) PatchEvent(ctx context.Context, req *dto.PatchEventReq) error {
	filter := bson.M{
		"_id":        bson.M{"$in": idAlternatives(req.ID)},
		"created_by": req.CreatedBy,
	}

	setFields := bson.M{}
	unsetFields := bson.M{}

	if req.Category != nil {
		setFields["category"] = *req.Category
	}
	if req.Price != nil {
		setFields["price"] = *req.Price
	}
	if req.City != nil {
		if *req.City == "" {
			unsetFields["location.city"] = ""
		} else {
			setFields["location.city"] = *req.City
		}
	}

	if len(setFields) == 0 && len(unsetFields) == 0 {
		err := s.db.Collection(collectionEvents).FindOne(ctx, filter).Err()
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return ErrNotFound
			}
			return fmt.Errorf("find event before patch: %w", err)
		}
		return nil
	}

	update := bson.M{}
	if len(setFields) > 0 {
		update["$set"] = setFields
	}
	if len(unsetFields) > 0 {
		update["$unset"] = unsetFields
	}

	result, err := s.db.Collection(collectionEvents).UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func mapEvent(doc bson.M) dto.Event {
	location := dto.Location{}
	if v, ok := doc["location"].(bson.M); ok {
		location.Address, _ = v["address"].(string)
		location.City, _ = v["city"].(string)
	}

	category, _ := doc["category"].(string)
	description, _ := doc["description"].(string)
	title, _ := doc["title"].(string)
	createdBy, _ := doc["created_by"].(string)

	return dto.Event{
		ID:          stringifyID(doc["_id"]),
		Title:       title,
		Category:    category,
		Price:       toInt64(doc["price"]),
		Description: description,
		Location:    location,
		CreatedAt:   toTime(doc["created_at"]),
		CreatedBy:   createdBy,
		StartedAt:   toTime(doc["started_at"]),
		FinishedAt:  toTime(doc["finished_at"]),
	}
}

func toTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case primitive.DateTime:
		return t.Time()
	case string:
		parsed, err := time.Parse(time.RFC3339, t)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	}
	return 0
}
