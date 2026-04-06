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
}

type UserService struct {
	l       logger.Interface
	storage UserStorage
}

func NewUserService(l logger.Interface, storage UserStorage) *UserService {
	return &UserService{
		l:       l,
		storage: storage,
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

	return &sdto.RegisterResp{ID: user.ID}, nil
}

func (s *UserService) Authenticate(ctx context.Context, req *sdto.AuthenticateReq) (*sdto.AuthenticateResp, error) {
	user, err := s.storage.GetByUsername(ctx, &mdto.GetUserByUsernameReq{Username: req.Username})
	if err != nil {
		return nil, fmt.Errorf("get username: %w", err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &sdto.AuthenticateResp{ID: user.ID}, nil
}
