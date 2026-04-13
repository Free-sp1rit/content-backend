package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"content-backend/internal/auth"
	"content-backend/internal/middleware"
	"content-backend/internal/model"
)

const (
	testJWTSecret = "test-secret"
	testJWTIssuer = "test-issuer"
)

func performHandlerRequest(method, target, body string, next http.HandlerFunc) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, target, bodyReader)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	return rec
}

func performAuthenticatedHandlerRequest(t *testing.T, userID int64, method, target, body string, next http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	tokenManager := auth.NewTokenManager(testJWTSecret, testJWTIssuer, time.Hour)
	token, err := tokenManager.Generate(userID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, target, bodyReader)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	protected := authMiddleware.RequireLogin(next)
	protected.ServeHTTP(rec, req)

	return rec
}

func decodeJSONResponse(t *testing.T, body io.Reader, dest any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(dest); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func assertErrIs(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got err %v, want %v", got, want)
	}
}

func assertJSONContentType(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("got Content-Type %q, want %q", rec.Header().Get("Content-Type"), "application/json")
	}
}

func assertArticleResponse(t *testing.T, got ArticleResponse, want model.Article) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("got id %d, want %d", got.ID, want.ID)
	}
	if got.Title != want.Title {
		t.Fatalf("got title %q, want %q", got.Title, want.Title)
	}
	if got.State != want.State {
		t.Fatalf("got state %q, want %q", got.State, want.State)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("got created at %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("got updated at %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func assertArticleDetailResponse(t *testing.T, got ArticleDetailResponse, want model.Article) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("got id %d, want %d", got.ID, want.ID)
	}
	if got.AuthorID != want.AuthorID {
		t.Fatalf("got author id %d, want %d", got.AuthorID, want.AuthorID)
	}
	if got.Title != want.Title {
		t.Fatalf("got title %q, want %q", got.Title, want.Title)
	}
	if got.Content != want.Content {
		t.Fatalf("got content %q, want %q", got.Content, want.Content)
	}
	if got.State != want.State {
		t.Fatalf("got state %q, want %q", got.State, want.State)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("got created at %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("got updated at %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func assertArticleResponses(t *testing.T, got []ArticleResponse, want []model.Article) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d articles, want %d", len(got), len(want))
	}

	for i := range got {
		assertArticleResponse(t, got[i], want[i])
	}
}
