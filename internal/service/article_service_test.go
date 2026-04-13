package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"content-backend/internal/model"
)

type fakeArticleRepo struct {
	createFunc         func(ctx context.Context, article model.Article) (int64, error)
	getByIDFunc        func(ctx context.Context, id int64) (model.Article, error)
	updateStateFunc    func(ctx context.Context, id int64, state string) error
	listByStateFunc    func(ctx context.Context, state string) ([]model.Article, error)
	listByAuthorIDFunc func(ctx context.Context, authorID int64) ([]model.Article, error)
	updateContentFunc  func(ctx context.Context, id int64, title, content string) error
}

func (r *fakeArticleRepo) Create(ctx context.Context, article model.Article) (int64, error) {
	if r.createFunc != nil {
		return r.createFunc(ctx, article)
	}
	panic("unexpected call to Create")
}

func (r *fakeArticleRepo) GetByID(ctx context.Context, id int64) (model.Article, error) {
	if r.getByIDFunc != nil {
		return r.getByIDFunc(ctx, id)
	}
	panic("unexpected call to GetByID")
}

func (r *fakeArticleRepo) UpdateState(ctx context.Context, id int64, state string) error {
	if r.updateStateFunc != nil {
		return r.updateStateFunc(ctx, id, state)
	}
	panic("unexpected call to UpdateState")
}

func (r *fakeArticleRepo) ListByState(ctx context.Context, state string) ([]model.Article, error) {
	if r.listByStateFunc != nil {
		return r.listByStateFunc(ctx, state)
	}
	panic("unexpected call to ListByState")
}

func (r *fakeArticleRepo) ListByAuthorID(ctx context.Context, authorID int64) ([]model.Article, error) {
	if r.listByAuthorIDFunc != nil {
		return r.listByAuthorIDFunc(ctx, authorID)
	}
	panic("unexpected call to ListByAuthorID")
}

func (r *fakeArticleRepo) UpdateContent(ctx context.Context, id int64, title, content string) error {
	if r.updateContentFunc != nil {
		return r.updateContentFunc(ctx, id, title, content)
	}
	panic("unexpected call to UpdateContent")
}

func TestArticleService_CreateArticle(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &fakeArticleRepo{
			createFunc: func(ctx context.Context, article model.Article) (int64, error) {
				if article.AuthorID != 7 {
					t.Fatalf("got author id %d, want 7", article.AuthorID)
				}
				if article.Title != "title" {
					t.Fatalf("got title %q, want %q", article.Title, "title")
				}
				if article.Content != "content" {
					t.Fatalf("got content %q, want %q", article.Content, "content")
				}
				return 10, nil
			},
		}

		service := NewArticleService(repo)

		id, err := service.CreateArticle(context.Background(), 7, "title", "content")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != 10 {
			t.Fatalf("got id %d, want 10", id)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		wantErr := errors.New("create failed")
		repo := &fakeArticleRepo{
			createFunc: func(ctx context.Context, article model.Article) (int64, error) {
				return 0, wantErr
			},
		}

		service := NewArticleService(repo)

		_, err := service.CreateArticle(context.Background(), 7, "title", "content")
		assertErrIs(t, err, wantErr)
	})
}

func TestArticleService_PublishArticle(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{}, sql.ErrNoRows
			},
		}

		service := NewArticleService(repo)

		err := service.PublishArticle(context.Background(), 1, 10)
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("not author", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 99, State: model.ArticleStateDraft}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.PublishArticle(context.Background(), 1, 10)
		assertErrIs(t, err, ErrPermissionDenied)
	})

	t.Run("already published", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 10, State: model.ArticleStatePublished}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.PublishArticle(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		updateCalled := false
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 10, State: model.ArticleStateDraft}, nil
			},
			updateStateFunc: func(ctx context.Context, id int64, state string) error {
				updateCalled = true
				if id != 1 {
					t.Fatalf("got id %d, want 1", id)
				}
				if state != model.ArticleStatePublished {
					t.Fatalf("got state %q, want %q", state, model.ArticleStatePublished)
				}
				return nil
			},
		}

		service := NewArticleService(repo)

		err := service.PublishArticle(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Fatal("expected UpdateState to be called")
		}
	})
}

func TestArticleService_ListPublishedArticles(t *testing.T) {
	wantArticles := []model.Article{
		{ID: 1, Title: "a", State: model.ArticleStatePublished},
		{ID: 2, Title: "b", State: model.ArticleStatePublished},
	}

	repo := &fakeArticleRepo{
		listByStateFunc: func(ctx context.Context, state string) ([]model.Article, error) {
			if state != model.ArticleStatePublished {
				t.Fatalf("got state %q, want %q", state, model.ArticleStatePublished)
			}
			return wantArticles, nil
		},
	}

	service := NewArticleService(repo)

	gotArticles, err := service.ListPublishedArticles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArticles) != len(wantArticles) {
		t.Fatalf("got %d articles, want %d", len(gotArticles), len(wantArticles))
	}
}

func TestArticleService_ListMyArticles(t *testing.T) {
	wantArticles := []model.Article{
		{ID: 1, AuthorID: 10, State: model.ArticleStateDraft},
		{ID: 2, AuthorID: 10, State: model.ArticleStatePublished},
	}

	repo := &fakeArticleRepo{
		listByAuthorIDFunc: func(ctx context.Context, authorID int64) ([]model.Article, error) {
			if authorID != 10 {
				t.Fatalf("got author id %d, want 10", authorID)
			}
			return wantArticles, nil
		},
	}

	service := NewArticleService(repo)

	gotArticles, err := service.ListMyArticles(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArticles) != len(wantArticles) {
		t.Fatalf("got %d articles, want %d", len(gotArticles), len(wantArticles))
	}
}

func TestArticleService_GetArticle(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{}, sql.ErrNoRows
			},
		}

		service := NewArticleService(repo)

		_, err := service.GetArticle(context.Background(), 1)
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("draft article is hidden", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, State: model.ArticleStateDraft}, nil
			},
		}

		service := NewArticleService(repo)

		_, err := service.GetArticle(context.Background(), 1)
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("success", func(t *testing.T) {
		wantArticle := model.Article{ID: 1, AuthorID: 10, Title: "title", State: model.ArticleStatePublished}
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return wantArticle, nil
			},
		}

		service := NewArticleService(repo)

		gotArticle, err := service.GetArticle(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotArticle.ID != wantArticle.ID {
			t.Fatalf("got article id %d, want %d", gotArticle.ID, wantArticle.ID)
		}
	})
}

func TestArticleService_UpdateArticle(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{}, sql.ErrNoRows
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("not author", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 200, State: model.ArticleStateDraft}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		assertErrIs(t, err, ErrPermissionDenied)
	})

	t.Run("not editable", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 100, State: model.ArticleStatePublished}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		assertErrIs(t, err, ErrArticleNotEditable)
	})

	t.Run("success", func(t *testing.T) {
		updateCalled := false
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 100, State: model.ArticleStateDraft}, nil
			},
			updateContentFunc: func(ctx context.Context, id int64, title, content string) error {
				updateCalled = true
				if id != 1 {
					t.Fatalf("got id %d, want 1", id)
				}
				if title != "new title" {
					t.Fatalf("got title %q, want %q", title, "new title")
				}
				if content != "new content" {
					t.Fatalf("got content %q, want %q", content, "new content")
				}
				return nil
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Fatal("expected UpdateContent to be called")
		}
	})
}
