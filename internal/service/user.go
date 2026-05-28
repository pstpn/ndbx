package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	sdto "ndbx/internal/service/dto"
	"ndbx/internal/storage/mongodb"
	mdto "ndbx/internal/storage/mongodb/dto"
	"ndbx/pkg/logger"
)

type UserStorage interface {
	Create(ctx context.Context, req *mdto.CreateUserReq) (*mdto.User, error)
	GetByUsername(ctx context.Context, req *mdto.GetUserByUsernameReq) (*mdto.GetUserByUsernameResp, error)
	GetUsers(ctx context.Context, req *mdto.GetUsersReq) (*mdto.GetUsersResp, error)
	GetByID(ctx context.Context, req *mdto.GetUserByIDReq) (*mdto.GetUserByIDResp, error)
}

type UserService struct {
	l            logger.Interface
	storage      UserStorage
	graphStorage GraphStorageInterface
}

func NewUserService(l logger.Interface, storage UserStorage, graphStorage GraphStorageInterface) *UserService {
	return &UserService{
		l:            l,
		storage:      storage,
		graphStorage: graphStorage,
	}
}

func (s *UserService) Register(ctx context.Context, req *sdto.RegisterReq) (*sdto.RegisterResp, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.storage.Create(ctx, &mdto.CreateUserReq{
		FullName: req.FullName,
		Username: req.Username,
		Password: string(hashedPassword),
	})
	if err != nil {
		if errors.Is(err, mongodb.ErrAlreadyExists) {
			err = ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	if err := s.graphStorage.CreateUser(ctx, user.ID); err != nil {
		s.l.Errorf("failed to create user in neo4j: %s", err.Error())
	}

	return &sdto.RegisterResp{ID: user.ID}, nil
}

func (s *UserService) Authenticate(ctx context.Context, req *sdto.AuthenticateReq) (*sdto.AuthenticateResp, error) {
	user, err := s.storage.GetByUsername(ctx, &mdto.GetUserByUsernameReq{Username: req.Username})
	if err != nil {
		if errors.Is(err, mongodb.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get username: %w", err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &sdto.AuthenticateResp{ID: user.ID}, nil
}

func (s *UserService) GetUsers(ctx context.Context, req *sdto.GetUsersReq) (*sdto.GetUsersResp, error) {
	resp, err := s.storage.GetUsers(ctx, &mdto.GetUsersReq{
		ID:     req.ID,
		Name:   req.Name,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	users := make([]sdto.UserData, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = sdto.UserData{
			ID:       u.ID,
			FullName: u.FullName,
			Username: u.Username,
		}
	}

	return &sdto.GetUsersResp{Users: users}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *sdto.GetUserReq) (*sdto.GetUserResp, error) {
	resp, err := s.storage.GetByID(ctx, &mdto.GetUserByIDReq{ID: req.ID})
	if err != nil {
		if errors.Is(err, mongodb.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &sdto.GetUserResp{User: sdto.UserData{
		ID:       resp.ID,
		FullName: resp.FullName,
		Username: resp.Username,
	}}, nil
}
