package middleware

import (
	"content-backend/internal/auth"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testJWTSecret = "test-secret"
	testJWTIssuer = "test-issuer"
)

func performProtectedRequest(tokenManager *auth.TokenManager, authHeader string, next http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/articles", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	rec := httptest.NewRecorder()
	handler := NewAuthMiddleware(tokenManager).RequireLogin(next)
	handler.ServeHTTP(rec, req)

	return rec
}

func TestAuthMiddleware_RequireLogin(t *testing.T) {
	errorCases := []struct {
		name         string
		tokenManager *auth.TokenManager
		authHeader   string
		wantStatus   int
	}{
		{name: "missing authorization", tokenManager: auth.NewTokenManager(testJWTSecret, testJWTIssuer, time.Hour), wantStatus: http.StatusUnauthorized},
		{name: "invalid token", tokenManager: auth.NewTokenManager(testJWTSecret, testJWTIssuer, time.Hour), authHeader: "Bearer bad-token", wantStatus: http.StatusUnauthorized},
		{name: "invalid token config", tokenManager: auth.NewTokenManager("", testJWTIssuer, time.Hour), authHeader: "Bearer any-token", wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
			})

			rec := performProtectedRequest(tc.tokenManager, tc.authHeader, next)

			if rec.Code != tc.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
			if nextCalled {
				t.Fatal("expected next not to be called")
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		tokenManager := auth.NewTokenManager(testJWTSecret, testJWTIssuer, time.Hour)

		expectedID := int64(1024)
		token, err := tokenManager.Generate(expectedID)
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}

		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true

			userID, ok := UserIDFromContext(r.Context())
			if !ok {
				t.Fatal("expected auth claims in context")
			}
			if userID != expectedID {
				t.Fatalf("got user id %d, want %d", userID, expectedID)
			}
		})

		rec := performProtectedRequest(tokenManager, "Bearer "+token, next)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !nextCalled {
			t.Fatal("expected next to be called")
		}
	})
}
