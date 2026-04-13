package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

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

	t.Run("user not found", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, sql.ErrNoRows
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Login(context.Background(), "user@example.com", "secret")
		assertErrIs(t, err, ErrInvalidCredentials)
	})

	t.Run("lookup error", func(t *testing.T) {
		wantErr := errors.New("lookup failed")
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{}, wantErr
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Login(context.Background(), "user@example.com", "secret")
		assertErrIs(t, err, wantErr)
	})

	t.Run("invalid password", func(t *testing.T) {
		repo := &fakeUserRepo{
			getByEmailFunc: func(ctx context.Context, email string) (model.User, error) {
				return model.User{ID: 7, Email: email, PasswordHash: passwordHash}, nil
			},
		}

		service := NewAuthService(repo, &fakeTokenGenerator{})

		_, err := service.Login(context.Background(), "user@example.com", "wrong-password")
		assertErrIs(t, err, ErrInvalidCredentials)
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

		_, err := service.Login(context.Background(), "user@example.com", "secret")
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

		token, err := service.Login(context.Background(), "user@example.com", "secret")
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
}
