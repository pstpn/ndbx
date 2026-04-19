package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"ndbx/internal/storage/mongodb/dto"
)

const collectionUsers = "users"

type UserStorage struct {
	db *mongo.Database
}

func NewUserStorage(db *mongo.Database) *UserStorage {
	return &UserStorage{db: db}
}

func (s *UserStorage) CreateIndex(ctx context.Context) error {
	coll := s.db.Collection(collectionUsers)
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "username", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return nil
}

func (s *UserStorage) Create(ctx context.Context, req *dto.CreateUserReq) (*dto.User, error) {
	coll := s.db.Collection(collectionUsers)

	user := &dto.User{
		FullName:     req.FullName,
		Username:     req.Username,
		PasswordHash: req.Password,
	}

	result, err := coll.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	user.ID = stringifyID(result.InsertedID)
	return user, nil
}

func (s *UserStorage) GetByUsername(ctx context.Context, req *dto.GetUserByUsernameReq) (*dto.GetUserByUsernameResp, error) {
	coll := s.db.Collection(collectionUsers)

	var userDoc bson.M
	err := coll.FindOne(ctx, bson.M{"username": req.Username}).Decode(&userDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	fullName, _ := userDoc["full_name"].(string)
	username, _ := userDoc["username"].(string)
	passwordHash, _ := userDoc["password_hash"].(string)

	return &dto.GetUserByUsernameResp{
		ID:           stringifyID(userDoc["_id"]),
		FullName:     fullName,
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}

func (s *UserStorage) GetUsers(ctx context.Context, req *dto.GetUsersReq) (*dto.GetUsersResp, error) {
	filter := bson.M{}
	if req.ID != "" {
		filter["_id"] = bson.M{"$in": idAlternatives(req.ID)}
	}
	if req.Name != "" {
		filter["full_name"] = primitive.Regex{Pattern: req.Name, Options: "i"}
	}

	opts := options.Find()
	if req.Limit > 0 {
		opts.SetLimit(req.Limit)
	}
	if req.Offset > 0 {
		opts.SetSkip(req.Offset)
	}

	cursor, err := s.db.Collection(collectionUsers).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find users: %w", err)
	}
	defer cursor.Close(ctx)

	users := make([]dto.User, 0)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode user: %w", err)
		}
		fullName, _ := doc["full_name"].(string)
		username, _ := doc["username"].(string)
		users = append(users, dto.User{
			ID:       stringifyID(doc["_id"]),
			FullName: fullName,
			Username: username,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return &dto.GetUsersResp{Users: users}, nil
}

func (s *UserStorage) GetByID(ctx context.Context, req *dto.GetUserByIDReq) (*dto.GetUserByIDResp, error) {
	filter := bson.M{"_id": bson.M{"$in": idAlternatives(req.ID)}}

	var userDoc bson.M
	err := s.db.Collection(collectionUsers).FindOne(ctx, filter).Decode(&userDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	fullName, _ := userDoc["full_name"].(string)
	username, _ := userDoc["username"].(string)

	return &dto.GetUserByIDResp{
		ID:       stringifyID(userDoc["_id"]),
		FullName: fullName,
		Username: username,
	}, nil
}
