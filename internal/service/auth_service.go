package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"content-backend/internal/model"

	"golang.org/x/crypto/bcrypt"
)

var ErrEmailAlreadyRegistered = errors.New("email already registered")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrLoginRateLimited = errors.New("login rate limited")

type userRepository interface {
	GetByEmail(ctx context.Context, email string) (model.User, error)
	Create(ctx context.Context, user model.User) (int64, error)
}

type tokenGenerator interface {
	Generate(userID int64) (string, error)
}

type loginRateLimiter interface {
	TooManyAttempts(ctx context.Context, key string) (bool, error)
	RecordFailure(ctx context.Context, key string) error
	Reset(ctx context.Context, key string) error
}

type AuthService struct {
	userRepo     userRepository
	tokenManager tokenGenerator
	loginLimiter loginRateLimiter
}

func NewAuthService(userRepo userRepository, tokenManager tokenGenerator) *AuthService {
	return &AuthService{userRepo: userRepo, tokenManager: tokenManager}
}

func NewAuthServiceWithLoginLimiter(userRepo userRepository, tokenManager tokenGenerator, loginLimiter loginRateLimiter) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		tokenManager: tokenManager,
		loginLimiter: loginLimiter,
	}
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
	limiterKey := loginRateLimitKey(email)
	if s.loginLimiter != nil {
		tooManyAttempts, err := s.loginLimiter.TooManyAttempts(ctx, limiterKey)
		if err != nil {
			log.Printf("check login rate limit: %v", err)
		} else if tooManyAttempts {
			return "", ErrLoginRateLimited
		}
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		s.recordLoginFailure(ctx, limiterKey)
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	err = comparePassword(user.PasswordHash, password)
	if err != nil {
		s.recordLoginFailure(ctx, limiterKey)
		return "", ErrInvalidCredentials
	}

	s.resetLoginLimiter(ctx, limiterKey)

	token, err := s.tokenManager.Generate(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}

const loginRateLimitKeyPrefix = "login:failures:"

func loginRateLimitKey(email string) string {
	return loginRateLimitKeyPrefix + strings.ToLower(strings.TrimSpace(email))
}

func (s *AuthService) recordLoginFailure(ctx context.Context, key string) {
	if s.loginLimiter == nil {
		return
	}

	err := s.loginLimiter.RecordFailure(ctx, key)
	if err != nil {
		log.Printf("record login failure: %v", err)
	}
}

func (s *AuthService) resetLoginLimiter(ctx context.Context, key string) {
	if s.loginLimiter == nil {
		return
	}

	err := s.loginLimiter.Reset(ctx, key)
	if err != nil {
		log.Printf("reset login limiter: %v", err)
	}
}
