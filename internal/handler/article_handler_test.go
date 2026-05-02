package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"content-backend/internal/model"
	"content-backend/internal/service"
)

type fakeArticleService struct {
	createArticleFunc         func(ctx context.Context, authorID int64, title, content string) (int64, error)
	publishArticleFunc        func(ctx context.Context, articleID, currentUserID int64) error
	listPublishedArticlesFunc func(ctx context.Context) ([]model.Article, error)
	listMyArticlesFunc        func(ctx context.Context, authorID int64) ([]model.Article, error)
	getArticleFunc            func(ctx context.Context, articleID int64, viewer service.ArticleViewer) (model.Article, error)
	updateArticleFunc         func(ctx context.Context, articleID, currentUserID int64, title, content string) error
}

func (s *fakeArticleService) CreateArticle(ctx context.Context, authorID int64, title, content string) (int64, error) {
	if s.createArticleFunc != nil {
		return s.createArticleFunc(ctx, authorID, title, content)
	}
	panic("unexpected call to CreateArticle")
}

func (s *fakeArticleService) PublishArticle(ctx context.Context, articleID, currentUserID int64) error {
	if s.publishArticleFunc != nil {
		return s.publishArticleFunc(ctx, articleID, currentUserID)
	}
	panic("unexpected call to PublishArticle")
}

func (s *fakeArticleService) ListPublishedArticles(ctx context.Context) ([]model.Article, error) {
	if s.listPublishedArticlesFunc != nil {
		return s.listPublishedArticlesFunc(ctx)
	}
	panic("unexpected call to ListPublishedArticles")
}

func (s *fakeArticleService) ListMyArticles(ctx context.Context, authorID int64) ([]model.Article, error) {
	if s.listMyArticlesFunc != nil {
		return s.listMyArticlesFunc(ctx, authorID)
	}
	panic("unexpected call to ListMyArticles")
}

func (s *fakeArticleService) GetArticle(ctx context.Context, articleID int64, viewer service.ArticleViewer) (model.Article, error) {
	if s.getArticleFunc != nil {
		return s.getArticleFunc(ctx, articleID, viewer)
	}
	panic("unexpected call to GetArticle")
}

func (s *fakeArticleService) UpdateArticle(ctx context.Context, articleID, currentUserID int64, title string, content string) error {
	if s.updateArticleFunc != nil {
		return s.updateArticleFunc(ctx, articleID, currentUserID, title, content)
	}
	panic("unexpected call to UpdateArticle")
}

func TestArticleHandler_CreateArticle(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodGet, "/articles", "", http.HandlerFunc(handler.CreateArticle))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/articles", `{"title":,"content":"c"}`, http.HandlerFunc(handler.CreateArticle))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing auth context", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/articles", `{"title":"t","content":"c"}`, http.HandlerFunc(handler.CreateArticle))

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &fakeArticleService{
			createArticleFunc: func(ctx context.Context, authorID int64, title, content string) (int64, error) {
				called = true
				if authorID != 7 {
					t.Fatalf("got authorID %d, want %d", authorID, int64(7))
				}
				if title != "t" {
					t.Fatalf("got title %q, want %q", title, "t")
				}
				if content != "c" {
					t.Fatalf("got content %q, want %q", content, "c")
				}
				return int64(123), nil
			},
		}
		handler := NewArticleHandler(svc)

		rec := performAuthenticatedHandlerRequest(
			t,
			7,
			http.MethodPost,
			"/articles",
			`{"title":"t","content":"c"}`,
			http.HandlerFunc(handler.CreateArticle),
		)

		if rec.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusCreated)
		}
		if !called {
			t.Fatal("expected CreateArticle to be called")
		}
		assertJSONContentType(t, rec)

		var res CreateArticleResponse
		decodeJSONResponse(t, rec.Body, &res)

		if res.ID != 123 {
			t.Fatalf("got articleID %d, want %d", res.ID, int64(123))
		}
	})
}

func TestArticleHandler_GetArticle(t *testing.T) {
	t.Run("article not found", func(t *testing.T) {
		svc := &fakeArticleService{
			getArticleFunc: func(ctx context.Context, articleID int64, viewer service.ArticleViewer) (model.Article, error) {
				return model.Article{}, service.ErrArticleNotFound
			},
		}
		handler := NewArticleHandler(svc)
		rec := performHandlerRequest(http.MethodGet, "/articles/1024", "", http.HandlerFunc(handler.GetArticle))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("success", func(t *testing.T) {
		now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		wantArticle := model.Article{
			ID:        1024,
			AuthorID:  7,
			Title:     "title",
			Content:   "content",
			State:     model.ArticleStatePublished,
			CreatedAt: now,
			UpdatedAt: now,
		}

		svc := &fakeArticleService{
			getArticleFunc: func(ctx context.Context, articleID int64, viewer service.ArticleViewer) (model.Article, error) {
				if articleID != 1024 {
					t.Fatalf("got article id %d, want %d", articleID, int64(1024))
				}
				if viewer.Authenticated {
					t.Fatal("expected anonymous viewer")
				}
				return wantArticle, nil
			},
		}
		handler := NewArticleHandler(svc)
		rec := performHandlerRequest(http.MethodGet, "/articles/1024", "", http.HandlerFunc(handler.GetArticle))

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		assertJSONContentType(t, rec)

		var got ArticleDetailResponse
		decodeJSONResponse(t, rec.Body, &got)
		assertArticleDetailResponse(t, got, wantArticle)
	})

	t.Run("success with authenticated viewer", func(t *testing.T) {
		now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		wantArticle := model.Article{
			ID:        1024,
			AuthorID:  7,
			Title:     "title",
			Content:   "content",
			State:     model.ArticleStatePublished,
			CreatedAt: now,
			UpdatedAt: now,
		}

		svc := &fakeArticleService{
			getArticleFunc: func(ctx context.Context, articleID int64, viewer service.ArticleViewer) (model.Article, error) {
				if articleID != 1024 {
					t.Fatalf("got article id %d, want %d", articleID, int64(1024))
				}
				if !viewer.Authenticated {
					t.Fatal("expected authenticated viewer")
				}
				if viewer.UserID != 99 {
					t.Fatalf("got viewer user id %d, want %d", viewer.UserID, int64(99))
				}
				return wantArticle, nil
			},
		}
		handler := NewArticleHandler(svc)
		rec := performAuthenticatedHandlerRequest(
			t,
			99,
			http.MethodGet,
			"/articles/1024",
			"",
			http.HandlerFunc(handler.GetArticle),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		assertJSONContentType(t, rec)

		var got ArticleDetailResponse
		decodeJSONResponse(t, rec.Body, &got)
		assertArticleDetailResponse(t, got, wantArticle)
	})
}

func TestArticleHandler_UpdateArticle(t *testing.T) {
	t.Run("bad path", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performAuthenticatedHandlerRequest(
			t,
			7,
			http.MethodPut,
			"/me/articles/abc",
			`{"title":"t","content":"c"}`,
			http.HandlerFunc(handler.UpdateArticle),
		)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performAuthenticatedHandlerRequest(
			t,
			7,
			http.MethodPut,
			"/me/articles/8",
			`{"title":`,
			http.HandlerFunc(handler.UpdateArticle),
		)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	errorCases := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "article not editable", serviceErr: service.ErrArticleNotEditable, wantStatus: http.StatusConflict},
		{name: "article not found", serviceErr: service.ErrArticleNotFound, wantStatus: http.StatusNotFound},
		{name: "permission denied", serviceErr: service.ErrPermissionDenied, wantStatus: http.StatusForbidden},
	}

	for _, tc := range errorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := &fakeArticleService{
				updateArticleFunc: func(ctx context.Context, articleID, currentUserID int64, title, content string) error {
					called = true
					return tc.serviceErr
				},
			}
			handler := NewArticleHandler(svc)

			rec := performAuthenticatedHandlerRequest(
				t,
				7,
				http.MethodPut,
				"/me/articles/123",
				`{"title":"t","content":"c"}`,
				http.HandlerFunc(handler.UpdateArticle),
			)

			if rec.Code != tc.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
			if !called {
				t.Fatal("expected UpdateArticle to be called")
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &fakeArticleService{
			updateArticleFunc: func(ctx context.Context, articleID, currentUserID int64, title, content string) error {
				called = true
				if articleID != 123 {
					t.Fatalf("got article id %d, want %d", articleID, int64(123))
				}
				if currentUserID != 7 {
					t.Fatalf("got current user id %d, want %d", currentUserID, int64(7))
				}
				if title != "t" {
					t.Fatalf("got title %q, want %q", title, "t")
				}
				if content != "c" {
					t.Fatalf("got content %q, want %q", content, "c")
				}
				return nil
			},
		}
		handler := NewArticleHandler(svc)

		rec := performAuthenticatedHandlerRequest(
			t,
			7,
			http.MethodPut,
			"/me/articles/123",
			`{"title":"t","content":"c"}`,
			http.HandlerFunc(handler.UpdateArticle),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !called {
			t.Fatal("expected UpdateArticle to be called")
		}
	})
}

func TestArticleHandler_ListPublishedArticles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		wantArticles := []model.Article{
			{
				ID:        1,
				AuthorID:  7,
				Title:     "first",
				Content:   "content-1",
				State:     model.ArticleStatePublished,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        2,
				AuthorID:  8,
				Title:     "second",
				Content:   "content-2",
				State:     model.ArticleStatePublished,
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		called := false
		svc := &fakeArticleService{
			listPublishedArticlesFunc: func(ctx context.Context) ([]model.Article, error) {
				called = true
				return wantArticles, nil
			},
		}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodGet, "/articles", "", http.HandlerFunc(handler.ListPublishedArticles))

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !called {
			t.Fatal("expected ListPublishedArticles to be called")
		}
		assertJSONContentType(t, rec)

		var got []ArticleResponse
		decodeJSONResponse(t, rec.Body, &got)
		assertArticleResponses(t, got, wantArticles)
	})

	t.Run("method not allowed", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/articles", "", http.HandlerFunc(handler.ListPublishedArticles))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestArticleHandler_ListMyArticles(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		wantArticles := []model.Article{
			{
				ID:        1,
				AuthorID:  7,
				Title:     "draft",
				Content:   "content-1",
				State:     model.ArticleStateDraft,
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        2,
				AuthorID:  7,
				Title:     "published",
				Content:   "content-2",
				State:     model.ArticleStatePublished,
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		called := false
		svc := &fakeArticleService{
			listMyArticlesFunc: func(ctx context.Context, authorID int64) ([]model.Article, error) {
				called = true
				if authorID != 7 {
					t.Fatalf("got author id %d, want %d", authorID, int64(7))
				}
				return wantArticles, nil
			},
		}
		handler := NewArticleHandler(svc)

		rec := performAuthenticatedHandlerRequest(
			t,
			7,
			http.MethodGet,
			"/me/articles",
			"",
			http.HandlerFunc(handler.ListMyArticles),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !called {
			t.Fatal("expected ListMyArticles to be called")
		}
		assertJSONContentType(t, rec)

		var got []ArticleResponse
		decodeJSONResponse(t, rec.Body, &got)
		assertArticleResponses(t, got, wantArticles)
	})

	t.Run("method not allowed", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/me/articles", "", http.HandlerFunc(handler.ListMyArticles))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestArticleHandler_PublishArticle(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodGet, "/articles/publish", "", http.HandlerFunc(handler.PublishArticle))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		svc := &fakeArticleService{}
		handler := NewArticleHandler(svc)

		rec := performHandlerRequest(http.MethodPost, "/articles/publish", `{"article_id":`, http.HandlerFunc(handler.PublishArticle))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	publishErrorCases := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "article not found", serviceErr: service.ErrArticleNotFound, wantStatus: http.StatusNotFound},
		{name: "permission denied", serviceErr: service.ErrPermissionDenied, wantStatus: http.StatusForbidden},
		{name: "article not publishable", serviceErr: service.ErrArticleNotPublishable, wantStatus: http.StatusConflict},
	}

	for _, tc := range publishErrorCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			called := false
			svc := &fakeArticleService{
				publishArticleFunc: func(ctx context.Context, articleID, currentUserID int64) error {
					called = true
					return tc.serviceErr
				},
			}
			handler := NewArticleHandler(svc)

			rec := performAuthenticatedHandlerRequest(
				t,
				7,
				http.MethodPost,
				"/articles/publish",
				`{"article_id":123}`,
				http.HandlerFunc(handler.PublishArticle),
			)

			if rec.Code != tc.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tc.wantStatus)
			}
			if !called {
				t.Fatal("expected PublishArticle to be called")
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		called := false
		svc := &fakeArticleService{
			publishArticleFunc: func(ctx context.Context, articleID, currentUserID int64) error {
				called = true
				if articleID != 123 {
					t.Fatalf("got article id %d, want %d", articleID, int64(123))
				}
				if currentUserID != 7 {
					t.Fatalf("got current user id %d, want %d", currentUserID, int64(7))
				}
				return nil
			},
		}
		handler := NewArticleHandler(svc)

		rec := performAuthenticatedHandlerRequest(
			t,
			7,
			http.MethodPost,
			"/articles/publish",
			`{"article_id":123}`,
			http.HandlerFunc(handler.PublishArticle),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !called {
			t.Fatal("expected PublishArticle to be called")
		}
	})
}
