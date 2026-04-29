package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"content-backend/internal/model"

	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepo struct {
	getByEmailFunc func(ctx context.Context, email string) (model.User, error)
	createFunc     func(ctx context.Context, user model.User) (int64, error)
}

func (r *fakeUserRepo) GetByEmail(ctx context.Context, email string) (model.User, error) {
	if r.getByEmailFunc != nil {
		return r.getByEmailFunc(ctx, email)
	}
	panic("unexpected call to GetByEmail")
}

func (r *fakeUserRepo) Create(ctx context.Context, user model.User) (int64, error) {
	if r.createFunc != nil {
		return r.createFunc(ctx, user)
	}
	panic("unexpected call to Create")
}

type fakeTokenGenerator struct {
	generateFunc func(userID int64) (string, error)
}

func (g *fakeTokenGenerator) Generate(userID int64) (string, error) {
	if g.generateFunc != nil {
		return g.generateFunc(userID)
	}
	panic("unexpected call to Generate")
}

type fakeLoginRateLimiter struct {
	tooManyAttemptsFunc func(ctx context.Context, key string) (bool, time.Duration, error)
	recordFailureFunc   func(ctx context.Context, key string) error
	resetFunc           func(ctx context.Context, key string) error
}

func (l *fakeLoginRateLimiter) TooManyAttempts(ctx context.Context, key string) (bool, time.Duration, error) {
	if l.tooManyAttemptsFunc != nil {
		return l.tooManyAttemptsFunc(ctx, key)
	}
	panic("unexpected call to TooManyAttempts")
}

func (l *fakeLoginRateLimiter) RecordFailure(ctx context.Context, key string) error {
	if l.recordFailureFunc != nil {
		return l.recordFailureFunc(ctx, key)
	}
	panic("unexpected call to RecordFailure")
}

func (l *fakeLoginRateLimiter) Reset(ctx context.Context, key string) error {
	if l.resetFunc != nil {
		return l.resetFunc(ctx, key)
	}
	panic("unexpected call to Reset")
}

func TestAuthService_Register(t *testing.T) {
	t.Run("email already registered", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 1, Email: email}, nil
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Register(context.Background(), "user@example.com", "secret")
		assertErrIs(t, err, ErrEmailAlreadyRegistered)
	})

	t.Run("lookup error", func(t *testing.T) {
		wantErr := errors.New("lookup failed")
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, wantErr
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Register(context.Background(), "user@example.com", "secret")
		assertErrIs(t, err, wantErr)
	})

	t.Run("password hashing error", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, sql.ErrNoRows
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Register(context.Background(), "user@example.com", strings.Repeat("x", 73))
		assertErrIs(t, err, bcrypt.ErrPasswordTooLong)
	})

	t.Run("create error", func(t *testing.T) {
		wantErr := errors.New("create failed")
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, sql.ErrNoRows
			},
			createFunc: func(ctx context.Context, user model.User) (int64, error) {
				if user.Email != "user@example.com" {
					t.Fatalf("got email %q, want %q", user.Email, "user@example.com")
				}
				if user.PasswordHash == "" {
					t.Fatal("expected password hash to be set")
				}
				assertPasswordMatches(t, user.PasswordHash, "secret")
				return 0, wantErr
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Register(context.Background(), "user@example.com", "secret")
		assertErrIs(t, err, wantErr)
	})

	t.Run("success", func(t *testing.T) {
		createCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, sql.ErrNoRows
			},
			createFunc: func(ctx context.Context, user model.User) (int64, error) {
				createCalled = true
				if user.Email != "user@example.com" {
					t.Fatalf("got email %q, want %q", user.Email, "user@example.com")
				}
				if user.PasswordHash == "" {
					t.Fatal("expected password hash to be set")
				}
				assertPasswordMatches(t, user.PasswordHash, "secret")
				return 42, nil
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		id, err := service.Register(context.Background(), "user@example.com", "secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != 42 {
			t.Fatalf("got id %d, want %d", id, int64(42))
		}
		if !createCalled {
			t.Fatal("expected Create to be called")
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	passwordHash := mustHashPassword(t, "secret")

	t.Run("rate limited", func(t *testing.T) {
		limiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				if key != loginRateLimitKeyPrefix+"user@example.com" {
					t.Fatalf("got limiter key %q, want %q", key, loginRateLimitKeyPrefix+"user@example.com")
				}
				return true, 90 * time.Second, nil
			},
		}

		service := NewAuthServiceWithLoginLimiter(&fakeUserRepo{}, &fakeTokenGenerator{}, limiter)

		_, err := service.Login(context.Background(), "User@Example.COM", "secret", "")
		assertErrIs(t, err, ErrLoginRateLimited)
		assertLoginRetryAfter(t, err, 90*time.Second)
	})

	t.Run("ip rate limited", func(t *testing.T) {
		emailLimiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				if key != loginRateLimitKeyPrefix+"user@example.com" {
					t.Fatalf("got email limiter key %q, want %q", key, loginRateLimitKeyPrefix+"user@example.com")
				}
				return false, 0, nil
			},
		}
		ipLimiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				if key != loginIPRateLimitKeyPrefix+"192.0.2.1" {
					t.Fatalf("got ip limiter key %q, want %q", key, loginIPRateLimitKeyPrefix+"192.0.2.1")
				}
				return true, 2 * time.Minute, nil
			},
		}

		service := NewAuthServiceWithLoginLimiters(&fakeUserRepo{}, &fakeTokenGenerator{}, emailLimiter, ipLimiter)

		_, err := service.Login(context.Background(), "user@example.com", "secret", "192.0.2.1")
		assertErrIs(t, err, ErrLoginRateLimited)
		assertLoginRetryAfter(t, err, 2*time.Minute)
	})

	t.Run("limiter check error does not block login", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		tokenGen := &fakeTokenGenerator{
			generateFunc: func(userID int64) (string, error) {
				return "test-token", nil
			},
		}
		limiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, errors.New("redis unavailable")
			},
			resetFunc: func(ctx context.Context, key string) error {
				return nil
			},
		}

		service := NewAuthServiceWithLoginLimiter(repo, tokenGen, limiter)

		token, err := service.Login(context.Background(), "user@example.com", "secret", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "test-token" {
			t.Fatalf("got token %q, want %q", token, "test-token")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, sql.ErrNoRows
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Login(context.Background(), "user@example.com", "secret", "")
		assertErrIs(t, err, ErrInvalidCredentials)
	})

	t.Run("user not found records failure", func(t *testing.T) {
		recordFailureCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, sql.ErrNoRows
			},
		}
		limiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
			recordFailureFunc: func(ctx context.Context, key string) error {
				recordFailureCalled = true
				if key != loginRateLimitKeyPrefix+"user@example.com" {
					t.Fatalf("got limiter key %q, want %q", key, loginRateLimitKeyPrefix+"user@example.com")
				}
				return nil
			},
		}

		service := NewAuthServiceWithLoginLimiter(repo, &fakeTokenGenerator{}, limiter)

		_, err := service.Login(context.Background(), "user@example.com", "secret", "")
		assertErrIs(t, err, ErrInvalidCredentials)
		if !recordFailureCalled {
			t.Fatal("expected RecordFailure to be called")
		}
	})

	t.Run("lookup error", func(t *testing.T) {
		wantErr := errors.New("lookup failed")
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, wantErr
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Login(context.Background(), "user@example.com", "secret", "")
		assertErrIs(t, err, wantErr)
	})

	t.Run("invalid password", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Login(context.Background(), "user@example.com", "wrong-password", "")
		assertErrIs(t, err, ErrInvalidCredentials)
	})

	t.Run("invalid password records failure", func(t *testing.T) {
		recordFailureCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		limiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
			recordFailureFunc: func(ctx context.Context, key string) error {
				recordFailureCalled = true
				if key != loginRateLimitKeyPrefix+"user@example.com" {
					t.Fatalf("got limiter key %q, want %q", key, loginRateLimitKeyPrefix+"user@example.com")
				}
				return nil
			},
		}

		service := NewAuthServiceWithLoginLimiter(repo, &fakeTokenGenerator{}, limiter)

		_, err := service.Login(context.Background(), "user@example.com", "wrong-password", "")
		assertErrIs(t, err, ErrInvalidCredentials)
		if !recordFailureCalled {
			t.Fatal("expected RecordFailure to be called")
		}
	})

	t.Run("invalid password records email and ip failures", func(t *testing.T) {
		emailFailureCalled := false
		ipFailureCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		emailLimiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
			recordFailureFunc: func(ctx context.Context, key string) error {
				emailFailureCalled = true
				if key != loginRateLimitKeyPrefix+"user@example.com" {
					t.Fatalf("got email limiter key %q, want %q", key, loginRateLimitKeyPrefix+"user@example.com")
				}
				return nil
			},
		}
		ipLimiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
			recordFailureFunc: func(ctx context.Context, key string) error {
				ipFailureCalled = true
				if key != loginIPRateLimitKeyPrefix+"192.0.2.1" {
					t.Fatalf("got ip limiter key %q, want %q", key, loginIPRateLimitKeyPrefix+"192.0.2.1")
				}
				return nil
			},
		}

		service := NewAuthServiceWithLoginLimiters(repo, &fakeTokenGenerator{}, emailLimiter, ipLimiter)

		_, err := service.Login(context.Background(), "user@example.com", "wrong-password", "192.0.2.1")
		assertErrIs(t, err, ErrInvalidCredentials)
		if !emailFailureCalled {
			t.Fatal("expected email RecordFailure to be called")
		}
		if !ipFailureCalled {
			t.Fatal("expected ip RecordFailure to be called")
		}
	})

	t.Run("token generation error", func(t *testing.T) {
		wantErr := errors.New("generate failed")
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		tokenGen := &fakeTokenGenerator{
			generateFunc: func(userID int64) (string, error) {
				if userID != 7 {
					t.Fatalf("got user id %d, want %d", userID, int64(7))
				}
				return "", wantErr
			},
		}

		service := NewAuthService(repo, tokenGen)

		_, err := service.Login(context.Background(), "user@example.com", "secret", "")
		assertErrIs(t, err, wantErr)
	})

	t.Run("success", func(t *testing.T) {
		generateCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				if email != "user@example.com" {
					t.Fatalf("got email %q, want %q", email, "user@example.com")
				}
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		tokenGen := &fakeTokenGenerator{
			generateFunc: func(userID int64) (string, error) {
				generateCalled = true
				if userID != 7 {
					t.Fatalf("got user id %d, want %d", userID, int64(7))
				}
				return "test-token", nil
			},
		}

		service := NewAuthService(repo, tokenGen)

		token, err := service.Login(context.Background(), "user@example.com", "secret", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "test-token" {
			t.Fatalf("got token %q, want %q", token, "test-token")
		}
		if !generateCalled {
			t.Fatal("expected Generate to be called")
		}
	})

	t.Run("success resets limiter", func(t *testing.T) {
		resetCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		tokenGen := &fakeTokenGenerator{
			generateFunc: func(userID int64) (string, error) {
				return "test-token", nil
			},
		}
		limiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
			resetFunc: func(ctx context.Context, key string) error {
				resetCalled = true
				if key != loginRateLimitKeyPrefix+"user@example.com" {
					t.Fatalf("got limiter key %q, want %q", key, loginRateLimitKeyPrefix+"user@example.com")
				}
				return nil
			},
		}

		service := NewAuthServiceWithLoginLimiter(repo, tokenGen, limiter)

		token, err := service.Login(context.Background(), "user@example.com", "secret", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "test-token" {
			t.Fatalf("got token %q, want %q", token, "test-token")
		}
		if !resetCalled {
			t.Fatal("expected Reset to be called")
		}
	})

	t.Run("success resets email limiter but keeps ip failure count", func(t *testing.T) {
		emailResetCalled := false
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}
		tokenGen := &fakeTokenGenerator{
			generateFunc: func(userID int64) (string, error) {
				return "test-token", nil
			},
		}
		emailLimiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
			resetFunc: func(ctx context.Context, key string) error {
				emailResetCalled = true
				return nil
			},
		}
		ipLimiter := &fakeLoginRateLimiter{
			tooManyAttemptsFunc: func(ctx context.Context, key string) (bool, time.Duration, error) {
				return false, 0, nil
			},
		}

		service := NewAuthServiceWithLoginLimiters(repo, tokenGen, emailLimiter, ipLimiter)

		token, err := service.Login(context.Background(), "user@example.com", "secret", "192.0.2.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "test-token" {
			t.Fatalf("got token %q, want %q", token, "test-token")
		}
		if !emailResetCalled {
			t.Fatal("expected email Reset to be called")
		}
	})
}
