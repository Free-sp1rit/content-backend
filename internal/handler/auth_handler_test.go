package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"content-backend/internal/service"
)

type fakeAuthService struct {
	registerFunc func(ctx context.Context, email, password string) (int64, error)
	loginFunc    func(ctx context.Context, email, password string) (string, error)
}

func (s *fakeAuthService) Register(ctx context.Context, email, password string) (int64, error) {
	if s.registerFunc != nil {
		return s.registerFunc(ctx, email, password)
	}
	panic("unexpected call to Register")
}

func (s *fakeAuthService) Login(ctx context.Context, email, password string) (string, error) {
	if s.loginFunc != nil {
		return s.loginFunc(ctx, email, password)
	}
	panic("unexpected call to Login")
}

func TestAuthHandler_Register(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		svc := &fakeAuthService{}
		handler := NewAuthHandler(svc)

		rec := performHandlerRequest(http.MethodGet, "/register", "", http.HandlerFunc(handler.Register))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := &fakeAuthService{}
		handler := NewAuthHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/register", `{"email":`, http.HandlerFunc(handler.Register))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	errorCases := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "email already registered", serviceErr: service.ErrEmailAlreadyRegistered, wantStatus: http.StatusConflict},
		{name: "internal error", serviceErr: errors.New("register failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := &fakeAuthService{
				registerFunc: func(ctx context.Context, email, password string) (int64, error) {
					called = true
					return 0, tc.serviceErr
				},
			}
			handler := NewAuthHandler(svc)

			rec := performHandlerRequest(
				http.MethodPost,
				"/register",
				`{"email":"user@example.com","password":"secret"}`,
				http.HandlerFunc(handler.Register),
			)

			if rec.Code != tc.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
			if !called {
				t.Fatal("expected Register to be called")
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &fakeAuthService{
			registerFunc: func(ctx context.Context, email, password string) (int64, error) {
				called = true
				if email != "user@example.com" {
					t.Fatalf("got email %q, want %q", email, "user@example.com")
				}
				if password != "secret" {
					t.Fatalf("got password %q, want %q", password, "secret")
				}
				return 42, nil
			},
		}
		handler := NewAuthHandler(svc)

		rec := performHandlerRequest(
			http.MethodPost,
			"/register",
			`{"email":"user@example.com","password":"secret"}`,
			http.HandlerFunc(handler.Register),
		)

		if rec.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusCreated)
		}
		if !called {
			t.Fatal("expected Register to be called")
		}
		assertJSONContentType(t, rec)

		var res RegisterResponse
		decodeJSONResponse(t, rec.Body, &res)
		if res.ID != 42 {
			t.Fatalf("got id %d, want %d", res.ID, int64(42))
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		svc := &fakeAuthService{}
		handler := NewAuthHandler(svc)

		rec := performHandlerRequest(http.MethodGet, "/login", "", http.HandlerFunc(handler.Login))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := &fakeAuthService{}
		handler := NewAuthHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/login", `{"email":`, http.HandlerFunc(handler.Login))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	loginErrorCases := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "invalid credentials", serviceErr: service.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized},
		{name: "rate limited", serviceErr: service.ErrLoginRateLimited, wantStatus: http.StatusTooManyRequests},
		{name: "internal error", serviceErr: errors.New("login failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range loginErrorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := &fakeAuthService{
				loginFunc: func(ctx context.Context, email, password string) (string, error) {
					called = true
					return "", tc.serviceErr
				},
			}
			handler := NewAuthHandler(svc)

			rec := performHandlerRequest(
				http.MethodPost,
				"/login",
				`{"email":"user@example.com","password":"secret"}`,
				http.HandlerFunc(handler.Login),
			)

			if rec.Code != tc.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
			if !called {
				t.Fatal("expected Login to be called")
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &fakeAuthService{
			loginFunc: func(ctx context.Context, email, password string) (string, error) {
				called = true
				if email != "user@example.com" {
					t.Fatalf("got email %q, want %q", email, "user@example.com")
				}
				if password != "secret" {
					t.Fatalf("got password %q, want %q", password, "secret")
				}
				return "test-token", nil
			},
		}
		handler := NewAuthHandler(svc)

		rec := performHandlerRequest(
			http.MethodPost,
			"/login",
			`{"email":"user@example.com","password":"secret"}`,
			http.HandlerFunc(handler.Login),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !called {
			t.Fatal("expected Login to be called")
		}
		assertJSONContentType(t, rec)

		var res LoginResponse
		decodeJSONResponse(t, rec.Body, &res)
		if res.Token != "test-token" {
			t.Fatalf("got token %q, want %q", res.Token, "test-token")
		}
	})
}
