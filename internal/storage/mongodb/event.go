package mongodb

import (
	"context"
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
			},
		},
	}

	indexModels[0].Options = options.Index().SetUnique(true)

	_, err := coll.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	return nil
}

func (s *EventStorage) Create(ctx context.Context, req *dto.CreateEventReq) (*dto.Event, error) {
	coll := s.db.Collection(collectionEvents)

	event := &dto.Event{
		Title:       req.Title,
		Description: req.Description,
		Location: dto.Location{
			Address: req.Address,
		},
		CreatedAt:  time.Now().UTC(),
		StartedAt:  req.StartedAt,
		FinishedAt: req.FinishedAt,
		CreatedBy:  req.CreatedBy,
	}

	result, err := coll.InsertOne(ctx, event)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert event: %w", err)
	}

	event.ID = result.InsertedID.(primitive.ObjectID).Hex()
	return event, nil
}

func (s *EventStorage) GetEvents(ctx context.Context, req *dto.GetEventsReq) (*dto.GetEventsResp, error) {
	coll := s.db.Collection(collectionEvents)

	filter := bson.M{}
	if req.Title != "" {
		filter["title"] = primitive.Regex{
			Pattern: req.Title,
			Options: "i",
		}
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

	var events []dto.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}

	return &dto.GetEventsResp{Events: events}, nil
}
