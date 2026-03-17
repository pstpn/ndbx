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

	user.ID = result.InsertedID.(primitive.ObjectID).Hex()
	return user, nil
}

func (s *UserStorage) GetByUsername(ctx context.Context, req *dto.GetUserByUsernameReq) (*dto.GetUserByUsernameResp, error) {
	coll := s.db.Collection(collectionUsers)

	var user dto.User
	err := coll.FindOne(ctx, bson.M{"username": req.Username}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	return &dto.GetUserByUsernameResp{
		ID:           user.ID,
		FullName:     user.FullName,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
	}, nil
}
