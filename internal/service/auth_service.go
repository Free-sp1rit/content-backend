package service

import (
	"context"
	"database/sql"
	"errors"

	"content-backend/internal/model"

	"golang.org/x/crypto/bcrypt"
)

var ErrEmailAlreadyRegistered = errors.New("email already registered")
var ErrInvalidCredentials = errors.New("invalid credentials")

type userRepository interface {
	GetByEmail(ctx context.Context, email string) (model.User, error)
	Create(ctx context.Context, user model.User) (int64, error)
}

type tokenGenerator interface {
	Generate(userID int64) (string, error)
}

type AuthService struct {
	userRepo     userRepository
	tokenManager tokenGenerator
}

func NewAuthService(userRepo userRepository, tokenManager tokenGenerator) *AuthService {
	return &AuthService{userRepo: userRepo, tokenManager: tokenManager}
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func comparePassword(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *AuthService) Register(ctx context.Context, email, password string) (int64, error) {
	_, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return 0, ErrEmailAlreadyRegistered
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return 0, err
	}

	user := model.User{
		Email:        email,
		PasswordHash: passwordHash,
	}

	id, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	err = comparePassword(user.PasswordHash, password)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.tokenManager.Generate(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}
