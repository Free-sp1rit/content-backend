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
	userRepo          userRepository
	tokenManager      tokenGenerator
	emailLoginLimiter loginRateLimiter
	ipLoginLimiter    loginRateLimiter
}

func NewAuthService(userRepo userRepository, tokenManager tokenGenerator) *AuthService {
	return &AuthService{userRepo: userRepo, tokenManager: tokenManager}
}

func NewAuthServiceWithLoginLimiter(userRepo userRepository, tokenManager tokenGenerator, loginLimiter loginRateLimiter) *AuthService {
	return NewAuthServiceWithLoginLimiters(userRepo, tokenManager, loginLimiter, nil)
}

func NewAuthServiceWithLoginLimiters(userRepo userRepository, tokenManager tokenGenerator, emailLoginLimiter, ipLoginLimiter loginRateLimiter) *AuthService {
	return &AuthService{
		userRepo:          userRepo,
		tokenManager:      tokenManager,
		emailLoginLimiter: emailLoginLimiter,
		ipLoginLimiter:    ipLoginLimiter,
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

func (s *AuthService) Login(ctx context.Context, email, password, clientIP string) (string, error) {
	emailLimiterKey := loginRateLimitKey(email)
	ipLimiterKey := loginIPRateLimitKey(clientIP)

	if s.tooManyLoginAttempts(ctx, s.emailLoginLimiter, emailLimiterKey) {
		return "", ErrLoginRateLimited
	}
	if ipLimiterKey != "" && s.tooManyLoginAttempts(ctx, s.ipLoginLimiter, ipLimiterKey) {
		return "", ErrLoginRateLimited
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		s.recordLoginFailures(ctx, emailLimiterKey, ipLimiterKey)
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}

	err = comparePassword(user.PasswordHash, password)
	if err != nil {
		s.recordLoginFailures(ctx, emailLimiterKey, ipLimiterKey)
		return "", ErrInvalidCredentials
	}

	s.resetLoginLimiter(ctx, s.emailLoginLimiter, emailLimiterKey)

	token, err := s.tokenManager.Generate(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}

const loginRateLimitKeyPrefix = "login:failures:email:"

const loginIPRateLimitKeyPrefix = "login:failures:ip:"

func loginRateLimitKey(email string) string {
	return loginRateLimitKeyPrefix + strings.ToLower(strings.TrimSpace(email))
}

func loginIPRateLimitKey(clientIP string) string {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return ""
	}

	return loginIPRateLimitKeyPrefix + clientIP
}

func (s *AuthService) tooManyLoginAttempts(ctx context.Context, limiter loginRateLimiter, key string) bool {
	if limiter == nil {
		return false
	}

	tooManyAttempts, err := limiter.TooManyAttempts(ctx, key)
	if err != nil {
		log.Printf("check login rate limit: %v", err)
		return false
	}
	return tooManyAttempts
}

func (s *AuthService) recordLoginFailures(ctx context.Context, emailKey, ipKey string) {
	s.recordLoginFailure(ctx, s.emailLoginLimiter, emailKey)
	if ipKey != "" {
		s.recordLoginFailure(ctx, s.ipLoginLimiter, ipKey)
	}
}

func (s *AuthService) recordLoginFailure(ctx context.Context, limiter loginRateLimiter, key string) {
	if limiter == nil {
		return
	}

	err := limiter.RecordFailure(ctx, key)
	if err != nil {
		log.Printf("record login failure: %v", err)
	}
}

func (s *AuthService) resetLoginLimiter(ctx context.Context, limiter loginRateLimiter, key string) {
	if limiter == nil {
		return
	}

	err := limiter.Reset(ctx, key)
	if err != nil {
		log.Printf("reset login limiter: %v", err)
	}
}
