package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"content-backend/internal/service"
)

type fakeAuthService struct {
	registerFunc func(ctx context.Context, email, password string) (int64, error)
	loginFunc    func(ctx context.Context, email, password, clientIP string) (string, error)
}

func (s *fakeAuthService) Register(ctx context.Context, email, password string) (int64, error) {
	if s.registerFunc != nil {
		return s.registerFunc(ctx, email, password)
	}
	panic("unexpected call to Register")
}

func (s *fakeAuthService) Login(ctx context.Context, email, password, clientIP string) (string, error) {
	if s.loginFunc != nil {
		return s.loginFunc(ctx, email, password, clientIP)
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
		name           string
		serviceErr     error
		wantStatus     int
		wantRetryAfter string
	}{
		{name: "invalid credentials", serviceErr: service.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized},
		{name: "rate limited", serviceErr: service.ErrLoginRateLimited, wantStatus: http.StatusTooManyRequests},
		{
			name:           "rate limited with retry after",
			serviceErr:     &service.LoginRateLimitedError{RetryAfter: 90 * time.Second},
			wantStatus:     http.StatusTooManyRequests,
			wantRetryAfter: "90",
		},
		{name: "internal error", serviceErr: errors.New("login failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range loginErrorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := &fakeAuthService{
				loginFunc: func(ctx context.Context, email, password, clientIP string) (string, error) {
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
			if rec.Header().Get("Retry-After") != tc.wantRetryAfter {
				t.Fatalf("got Retry-After %q, want %q", rec.Header().Get("Retry-After"), tc.wantRetryAfter)
			}
			if !called {
				t.Fatal("expected Login to be called")
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &fakeAuthService{
			loginFunc: func(ctx context.Context, email, password, clientIP string) (string, error) {
				called = true
				if email != "user@example.com" {
					t.Fatalf("got email %q, want %q", email, "user@example.com")
				}
				if password != "secret" {
					t.Fatalf("got password %q, want %q", password, "secret")
				}
				if clientIP != "192.0.2.1" {
					t.Fatalf("got client ip %q, want %q", clientIP, "192.0.2.1")
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

	t.Run("client ip without port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"user@example.com","password":"secret"}`))
		req.RemoteAddr = "203.0.113.10"
		rec := httptest.NewRecorder()
		svc := &fakeAuthService{
			loginFunc: func(ctx context.Context, email, password, clientIP string) (string, error) {
				if clientIP != "203.0.113.10" {
					t.Fatalf("got client ip %q, want %q", clientIP, "203.0.113.10")
				}
				return "test-token", nil
			},
		}
		handler := NewAuthHandler(svc)

		handler.Login(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestClientIPFromRequest(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		want          string
	}{
		{
			name:       "direct request with port",
			remoteAddr: "198.51.100.10:12345",
			want:       "198.51.100.10",
		},
		{
			name:          "direct request ignores forwarded header",
			remoteAddr:    "198.51.100.10:12345",
			xForwardedFor: "203.0.113.10",
			want:          "198.51.100.10",
		},
		{
			name:          "trusted proxy uses forwarded header",
			remoteAddr:    "172.18.0.3:12345",
			xForwardedFor: "203.0.113.10",
			want:          "203.0.113.10",
		},
		{
			name:       "trusted proxy falls back to real ip header",
			remoteAddr: "127.0.0.1:12345",
			xRealIP:    "203.0.113.11",
			want:       "203.0.113.11",
		},
		{
			name:          "trusted proxy ignores invalid forwarded header",
			remoteAddr:    "172.18.0.3:12345",
			xForwardedFor: "not-an-ip",
			want:          "172.18.0.3",
		},
		{
			name:       "remote address without port",
			remoteAddr: "203.0.113.10",
			want:       "203.0.113.10",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := clientIPFromRequest(req)

			if got != tt.want {
				t.Fatalf("got client ip %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want int64
	}{
		{name: "round seconds up", in: 1500 * time.Millisecond, want: 2},
		{name: "exact seconds", in: 2 * time.Second, want: 2},
		{name: "minimum one second", in: time.Millisecond, want: 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := retryAfterSeconds(tt.in)
			if got != tt.want {
				t.Fatalf("got retry after seconds %d, want %d", got, tt.want)
			}
		})
	}
}
