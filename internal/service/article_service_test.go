package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"content-backend/internal/model"
)

type fakeArticleRepo struct {
	createFunc                        func(ctx context.Context, article model.Article) (int64, error)
	getByIDFunc                       func(ctx context.Context, id int64) (model.Article, error)
	updateStateIfAuthorAndStateFunc   func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error)
	listByStateFunc                   func(ctx context.Context, state string) ([]model.Article, error)
	listByAuthorIDFunc                func(ctx context.Context, authorID int64) ([]model.Article, error)
	updateContentIfAuthorAndStateFunc func(ctx context.Context, id, authorID int64, state, title, content string) (bool, error)
	deleteIfAuthorAndNotDeletedFunc   func(ctx context.Context, id, authorID int64) (string, bool, error)
}

type fakeArticleCache struct {
	getFunc    func(ctx context.Context, key string) (string, error)
	setFunc    func(ctx context.Context, key string, value string, ttl time.Duration) error
	deleteFunc func(ctx context.Context, key string) error
}

type fakeArticleViewCounter struct {
	incrementFunc              func(ctx context.Context, articleID int64) error
	incrementAuthenticatedFunc func(ctx context.Context, articleID, userID int64) error
}

func (c *fakeArticleCache) Get(ctx context.Context, key string) (string, error) {
	if c.getFunc != nil {
		return c.getFunc(ctx, key)
	}
	panic("unexpected call to Get")
}

func (c *fakeArticleCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if c.setFunc != nil {
		return c.setFunc(ctx, key, value, ttl)
	}
	panic("unexpected call to Set")
}

func (c *fakeArticleCache) Delete(ctx context.Context, key string) error {
	if c.deleteFunc != nil {
		return c.deleteFunc(ctx, key)
	}
	panic("unexpected call to Delete")
}

func (c *fakeArticleViewCounter) Increment(ctx context.Context, articleID int64) error {
	if c.incrementFunc != nil {
		return c.incrementFunc(ctx, articleID)
	}
	panic("unexpected call to Increment")
}

func (c *fakeArticleViewCounter) IncrementAuthenticated(ctx context.Context, articleID, userID int64) error {
	if c.incrementAuthenticatedFunc != nil {
		return c.incrementAuthenticatedFunc(ctx, articleID, userID)
	}
	panic("unexpected call to IncrementAuthenticated")
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

func (r *fakeArticleRepo) UpdateStateIfAuthorAndState(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
	if r.updateStateIfAuthorAndStateFunc != nil {
		return r.updateStateIfAuthorAndStateFunc(ctx, id, authorID, currentState, nextState)
	}
	panic("unexpected call to UpdateStateIfAuthorAndState")
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

func (r *fakeArticleRepo) UpdateContentIfAuthorAndState(ctx context.Context, id, authorID int64, state, title, content string) (bool, error) {
	if r.updateContentIfAuthorAndStateFunc != nil {
		return r.updateContentIfAuthorAndStateFunc(ctx, id, authorID, state, title, content)
	}
	panic("unexpected call to UpdateContentIfAuthorAndState")
}

func (r *fakeArticleRepo) DeleteIfAuthorAndNotDeleted(ctx context.Context, id, authorID int64) (string, bool, error) {
	if r.deleteIfAuthorAndNotDeletedFunc != nil {
		return r.deleteIfAuthorAndNotDeletedFunc(ctx, id, authorID)
	}
	panic("unexpected call to DeleteIfAuthorAndNotDeleted")
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
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				return false, nil
			},
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
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				return false, nil
			},
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
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				return false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 10, State: model.ArticleStatePublished}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.PublishArticle(context.Background(), 1, 10)
		assertErrIs(t, err, ErrArticleNotPublishable)
	})

	t.Run("update error", func(t *testing.T) {
		wantErr := errors.New("update failed")
		repo := &fakeArticleRepo{
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				return false, wantErr
			},
		}

		service := NewArticleService(repo)

		err := service.PublishArticle(context.Background(), 1, 10)
		assertErrIs(t, err, wantErr)
	})

	t.Run("success", func(t *testing.T) {
		updateCalled := false
		deleteCacheCalled := false
		repo := &fakeArticleRepo{
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				updateCalled = true
				if id != 1 {
					t.Fatalf("got id %d, want 1", id)
				}
				if authorID != 10 {
					t.Fatalf("got author id %d, want 10", authorID)
				}
				if currentState != model.ArticleStateDraft {
					t.Fatalf("got current state %q, want %q", currentState, model.ArticleStateDraft)
				}
				if nextState != model.ArticleStatePublished {
					t.Fatalf("got next state %q, want %q", nextState, model.ArticleStatePublished)
				}
				return true, nil
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				deleteCacheCalled = true
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.PublishArticle(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Fatal("expected UpdateStateIfAuthorAndState to be called")
		}
		if !deleteCacheCalled {
			t.Fatal("expected cache Delete to be called")
		}
	})

	t.Run("failed condition does not delete cache", func(t *testing.T) {
		repo := &fakeArticleRepo{
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				return false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 10, State: model.ArticleStatePublished}, nil
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				t.Fatal("did not expect cache Delete to be called")
				return nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.PublishArticle(context.Background(), 1, 10)
		assertErrIs(t, err, ErrArticleNotPublishable)
	})

	t.Run("cache delete error does not block success", func(t *testing.T) {
		repo := &fakeArticleRepo{
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				return true, nil
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return errors.New("redis unavailable")
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.PublishArticle(context.Background(), 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestArticleService_ListPublishedArticles(t *testing.T) {
	t.Run("without cache", func(t *testing.T) {
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
	})

	t.Run("cache hit", func(t *testing.T) {
		wantArticles := []model.Article{
			{ID: 1, Title: "cached", State: model.ArticleStatePublished},
		}
		cachedData, err := json.Marshal(wantArticles)
		if err != nil {
			t.Fatalf("marshal cached articles: %v", err)
		}
		cache := &fakeArticleCache{
			getFunc: func(ctx context.Context, key string) (string, error) {
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return string(cachedData), nil
			},
		}
		repo := &fakeArticleRepo{}

		service := NewArticleServiceWithCache(repo, cache)

		gotArticles, err := service.ListPublishedArticles(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gotArticles) != len(wantArticles) {
			t.Fatalf("got %d articles, want %d", len(gotArticles), len(wantArticles))
		}
		if gotArticles[0].Title != wantArticles[0].Title {
			t.Fatalf("got title %q, want %q", gotArticles[0].Title, wantArticles[0].Title)
		}
	})

	t.Run("cache miss stores repository result", func(t *testing.T) {
		wantArticles := []model.Article{
			{ID: 1, Title: "a", State: model.ArticleStatePublished},
			{ID: 2, Title: "b", State: model.ArticleStatePublished},
		}
		setCalled := false
		cache := &fakeArticleCache{
			getFunc: func(ctx context.Context, key string) (string, error) {
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return "", nil
			},
			setFunc: func(ctx context.Context, key string, value string, ttl time.Duration) error {
				setCalled = true
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				if ttl != publishedArticlesCacheTTL {
					t.Fatalf("got ttl %v, want %v", ttl, publishedArticlesCacheTTL)
				}

				var gotArticles []model.Article
				err := json.Unmarshal([]byte(value), &gotArticles)
				if err != nil {
					t.Fatalf("unmarshal cached value: %v", err)
				}
				if len(gotArticles) != len(wantArticles) {
					t.Fatalf("got %d cached articles, want %d", len(gotArticles), len(wantArticles))
				}
				return nil
			},
		}
		repo := &fakeArticleRepo{
			listByStateFunc: func(ctx context.Context, state string) ([]model.Article, error) {
				if state != model.ArticleStatePublished {
					t.Fatalf("got state %q, want %q", state, model.ArticleStatePublished)
				}
				return wantArticles, nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		gotArticles, err := service.ListPublishedArticles(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gotArticles) != len(wantArticles) {
			t.Fatalf("got %d articles, want %d", len(gotArticles), len(wantArticles))
		}
		if !setCalled {
			t.Fatal("expected cache Set to be called")
		}
	})

	t.Run("cache miss stores empty list as json array", func(t *testing.T) {
		setCalled := false
		cache := &fakeArticleCache{
			getFunc: func(ctx context.Context, key string) (string, error) {
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return "", nil
			},
			setFunc: func(ctx context.Context, key string, value string, ttl time.Duration) error {
				setCalled = true
				if value != "[]" {
					t.Fatalf("got cached value %q, want []", value)
				}
				return nil
			},
		}
		repo := &fakeArticleRepo{
			listByStateFunc: func(ctx context.Context, state string) ([]model.Article, error) {
				return nil, nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		gotArticles, err := service.ListPublishedArticles(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotArticles == nil {
			t.Fatal("expected empty article slice, got nil")
		}
		if len(gotArticles) != 0 {
			t.Fatalf("got %d articles, want 0", len(gotArticles))
		}
		if !setCalled {
			t.Fatal("expected cache Set to be called")
		}
	})

	t.Run("concurrent cache misses share repository call", func(t *testing.T) {
		const requestCount = 20

		wantArticles := []model.Article{
			{ID: 1, Title: "shared", State: model.ArticleStatePublished},
		}
		repoStarted := make(chan struct{})
		releaseRepo := make(chan struct{})
		var repoStartedOnce sync.Once
		var repoCallCount atomic.Int32

		repo := &fakeArticleRepo{
			listByStateFunc: func(ctx context.Context, state string) ([]model.Article, error) {
				repoCallCount.Add(1)
				repoStartedOnce.Do(func() {
					close(repoStarted)
				})
				<-releaseRepo
				return wantArticles, nil
			},
		}

		var cacheMu sync.Mutex
		var getCallCount atomic.Int32
		cachedValue := ""
		initialGetsReady := make(chan struct{})
		releaseInitialGets := make(chan struct{})
		cache := &fakeArticleCache{
			getFunc: func(ctx context.Context, key string) (string, error) {
				getNumber := getCallCount.Add(1)
				if getNumber <= requestCount {
					if getNumber == requestCount {
						close(initialGetsReady)
					}
					<-releaseInitialGets
				}

				cacheMu.Lock()
				defer cacheMu.Unlock()
				return cachedValue, nil
			},
			setFunc: func(ctx context.Context, key string, value string, ttl time.Duration) error {
				cacheMu.Lock()
				defer cacheMu.Unlock()
				cachedValue = value
				return nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		start := make(chan struct{})
		ready := make(chan struct{})
		errCh := make(chan error, requestCount)
		var readyCount atomic.Int32
		var wg sync.WaitGroup

		for i := 0; i < requestCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if readyCount.Add(1) == requestCount {
					close(ready)
				}
				<-start

				gotArticles, err := service.ListPublishedArticles(context.Background())
				if err != nil {
					errCh <- err
					return
				}
				if len(gotArticles) != len(wantArticles) {
					errCh <- errors.New("unexpected article count")
				}
			}()
		}

		<-ready
		close(start)

		select {
		case <-initialGetsReady:
		case <-time.After(time.Second):
			close(releaseInitialGets)
			close(releaseRepo)
			t.Fatal("timed out waiting for initial cache misses")
		}
		close(releaseInitialGets)

		select {
		case <-repoStarted:
		case <-time.After(time.Second):
			close(releaseRepo)
			t.Fatal("timed out waiting for repository call")
		}

		close(releaseRepo)
		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if repoCallCount.Load() != 1 {
			t.Fatalf("got repository calls %d, want 1", repoCallCount.Load())
		}
	})
}

func TestArticleService_PublishedArticlesCacheConsistency(t *testing.T) {
	t.Run("publish invalidates warmed empty cache and reloads published article", func(t *testing.T) {
		ctx := context.Background()
		publishedArticle := model.Article{ID: 1, AuthorID: 10, Title: "published", State: model.ArticleStatePublished}
		articlesByState := []model.Article{}
		cache := newMemoryArticleCache(t)

		repo := &fakeArticleRepo{
			listByStateFunc: func(ctx context.Context, state string) ([]model.Article, error) {
				if state != model.ArticleStatePublished {
					t.Fatalf("got state %q, want %q", state, model.ArticleStatePublished)
				}
				return append([]model.Article(nil), articlesByState...), nil
			},
			updateStateIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error) {
				if id != publishedArticle.ID {
					t.Fatalf("got id %d, want %d", id, publishedArticle.ID)
				}
				if authorID != publishedArticle.AuthorID {
					t.Fatalf("got author id %d, want %d", authorID, publishedArticle.AuthorID)
				}
				if currentState != model.ArticleStateDraft {
					t.Fatalf("got current state %q, want %q", currentState, model.ArticleStateDraft)
				}
				if nextState != model.ArticleStatePublished {
					t.Fatalf("got next state %q, want %q", nextState, model.ArticleStatePublished)
				}
				articlesByState = []model.Article{publishedArticle}
				return true, nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		gotArticles, err := service.ListPublishedArticles(ctx)
		if err != nil {
			t.Fatalf("unexpected error warming cache: %v", err)
		}
		if len(gotArticles) != 0 {
			t.Fatalf("got %d warmed articles, want 0", len(gotArticles))
		}
		if gotCached := cache.value(publishedArticlesCacheKey); gotCached != "[]" {
			t.Fatalf("got warmed cache %q, want []", gotCached)
		}

		err = service.PublishArticle(ctx, publishedArticle.ID, publishedArticle.AuthorID)
		if err != nil {
			t.Fatalf("unexpected publish error: %v", err)
		}
		if gotCached := cache.value(publishedArticlesCacheKey); gotCached != "" {
			t.Fatalf("got cache after publish %q, want empty", gotCached)
		}

		gotArticles, err = service.ListPublishedArticles(ctx)
		if err != nil {
			t.Fatalf("unexpected error after publish: %v", err)
		}
		if !containsArticleID(gotArticles, publishedArticle.ID) {
			t.Fatalf("expected published article %d after cache invalidation", publishedArticle.ID)
		}
	})

	t.Run("delete published invalidates warmed cache and reloads without deleted article", func(t *testing.T) {
		ctx := context.Background()
		deletedArticle := model.Article{ID: 2, AuthorID: 20, Title: "deleted", State: model.ArticleStatePublished}
		remainingArticle := model.Article{ID: 3, AuthorID: 21, Title: "remaining", State: model.ArticleStatePublished}
		articlesByState := []model.Article{deletedArticle, remainingArticle}
		cache := newMemoryArticleCache(t)

		repo := &fakeArticleRepo{
			listByStateFunc: func(ctx context.Context, state string) ([]model.Article, error) {
				if state != model.ArticleStatePublished {
					t.Fatalf("got state %q, want %q", state, model.ArticleStatePublished)
				}
				return append([]model.Article(nil), articlesByState...), nil
			},
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				if id != deletedArticle.ID {
					t.Fatalf("got id %d, want %d", id, deletedArticle.ID)
				}
				if authorID != deletedArticle.AuthorID {
					t.Fatalf("got author id %d, want %d", authorID, deletedArticle.AuthorID)
				}
				articlesByState = []model.Article{remainingArticle}
				return model.ArticleStatePublished, true, nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		gotArticles, err := service.ListPublishedArticles(ctx)
		if err != nil {
			t.Fatalf("unexpected error warming cache: %v", err)
		}
		if !containsArticleID(gotArticles, deletedArticle.ID) {
			t.Fatalf("expected warmed cache result to include article %d", deletedArticle.ID)
		}
		if gotCached := cache.value(publishedArticlesCacheKey); !cacheValueContainsArticleID(t, gotCached, deletedArticle.ID) {
			t.Fatalf("expected warmed cache to include article %d, got %q", deletedArticle.ID, gotCached)
		}

		err = service.DeleteArticle(ctx, deletedArticle.ID, deletedArticle.AuthorID)
		if err != nil {
			t.Fatalf("unexpected delete error: %v", err)
		}
		if gotCached := cache.value(publishedArticlesCacheKey); gotCached != "" {
			t.Fatalf("got cache after delete %q, want empty", gotCached)
		}

		gotArticles, err = service.ListPublishedArticles(ctx)
		if err != nil {
			t.Fatalf("unexpected error after delete: %v", err)
		}
		if containsArticleID(gotArticles, deletedArticle.ID) {
			t.Fatalf("expected deleted article %d to be hidden after cache invalidation", deletedArticle.ID)
		}
		if !containsArticleID(gotArticles, remainingArticle.ID) {
			t.Fatalf("expected remaining article %d to stay visible", remainingArticle.ID)
		}
	})
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
		counter := &fakeArticleViewCounter{}

		service := NewArticleServiceWithViewCounter(repo, counter)

		_, err := service.GetArticle(context.Background(), 1, ArticleViewer{})
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("draft article is hidden", func(t *testing.T) {
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, State: model.ArticleStateDraft}, nil
			},
		}
		counter := &fakeArticleViewCounter{}

		service := NewArticleServiceWithViewCounter(repo, counter)

		_, err := service.GetArticle(context.Background(), 1, ArticleViewer{})
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("success", func(t *testing.T) {
		wantArticle := model.Article{ID: 1, AuthorID: 10, Title: "title", State: model.ArticleStatePublished}
		incrementCalled := false
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return wantArticle, nil
			},
		}
		counter := &fakeArticleViewCounter{
			incrementFunc: func(ctx context.Context, articleID int64) error {
				incrementCalled = true
				if articleID != wantArticle.ID {
					t.Fatalf("got article id %d, want %d", articleID, wantArticle.ID)
				}
				return nil
			},
		}

		service := NewArticleServiceWithViewCounter(repo, counter)

		gotArticle, err := service.GetArticle(context.Background(), 1, ArticleViewer{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotArticle.ID != wantArticle.ID {
			t.Fatalf("got article id %d, want %d", gotArticle.ID, wantArticle.ID)
		}
		if !incrementCalled {
			t.Fatal("expected view counter Increment to be called")
		}
	})

	t.Run("authenticated viewer increments user view count", func(t *testing.T) {
		wantArticle := model.Article{ID: 1, AuthorID: 10, Title: "title", State: model.ArticleStatePublished}
		incrementCalled := false
		incrementAuthenticatedCalled := false
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return wantArticle, nil
			},
		}
		counter := &fakeArticleViewCounter{
			incrementFunc: func(ctx context.Context, articleID int64) error {
				incrementCalled = true
				if articleID != wantArticle.ID {
					t.Fatalf("got article id %d, want %d", articleID, wantArticle.ID)
				}
				return nil
			},
			incrementAuthenticatedFunc: func(ctx context.Context, articleID, userID int64) error {
				incrementAuthenticatedCalled = true
				if articleID != wantArticle.ID {
					t.Fatalf("got article id %d, want %d", articleID, wantArticle.ID)
				}
				if userID != 99 {
					t.Fatalf("got user id %d, want %d", userID, int64(99))
				}
				return nil
			},
		}

		service := NewArticleServiceWithViewCounter(repo, counter)

		gotArticle, err := service.GetArticle(context.Background(), 1, ArticleViewer{UserID: 99, Authenticated: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotArticle.ID != wantArticle.ID {
			t.Fatalf("got article id %d, want %d", gotArticle.ID, wantArticle.ID)
		}
		if !incrementCalled {
			t.Fatal("expected view counter Increment to be called")
		}
		if !incrementAuthenticatedCalled {
			t.Fatal("expected view counter IncrementAuthenticated to be called")
		}
	})

	t.Run("view counter error does not block article detail", func(t *testing.T) {
		wantArticle := model.Article{ID: 1, AuthorID: 10, Title: "title", State: model.ArticleStatePublished}
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return wantArticle, nil
			},
		}
		counter := &fakeArticleViewCounter{
			incrementFunc: func(ctx context.Context, articleID int64) error {
				return errors.New("redis unavailable")
			},
		}

		service := NewArticleServiceWithViewCounter(repo, counter)

		gotArticle, err := service.GetArticle(context.Background(), 1, ArticleViewer{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotArticle.ID != wantArticle.ID {
			t.Fatalf("got article id %d, want %d", gotArticle.ID, wantArticle.ID)
		}
	})

	t.Run("authenticated view counter error does not block article detail", func(t *testing.T) {
		wantArticle := model.Article{ID: 1, AuthorID: 10, Title: "title", State: model.ArticleStatePublished}
		repo := &fakeArticleRepo{
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return wantArticle, nil
			},
		}
		counter := &fakeArticleViewCounter{
			incrementFunc: func(ctx context.Context, articleID int64) error {
				return nil
			},
			incrementAuthenticatedFunc: func(ctx context.Context, articleID, userID int64) error {
				return errors.New("redis unavailable")
			},
		}

		service := NewArticleServiceWithViewCounter(repo, counter)

		gotArticle, err := service.GetArticle(context.Background(), 1, ArticleViewer{UserID: 99, Authenticated: true})
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
			updateContentIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, state, title, content string) (bool, error) {
				return false, nil
			},
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
			updateContentIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, state, title, content string) (bool, error) {
				return false, nil
			},
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
			updateContentIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, state, title, content string) (bool, error) {
				return false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 100, State: model.ArticleStatePublished}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		assertErrIs(t, err, ErrArticleNotEditable)
	})

	t.Run("update error", func(t *testing.T) {
		wantErr := errors.New("update failed")
		repo := &fakeArticleRepo{
			updateContentIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, state, title, content string) (bool, error) {
				return false, wantErr
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		assertErrIs(t, err, wantErr)
	})

	t.Run("success", func(t *testing.T) {
		updateCalled := false
		repo := &fakeArticleRepo{
			updateContentIfAuthorAndStateFunc: func(ctx context.Context, id, authorID int64, state, title, content string) (bool, error) {
				updateCalled = true
				if id != 1 {
					t.Fatalf("got id %d, want 1", id)
				}
				if authorID != 100 {
					t.Fatalf("got author id %d, want 100", authorID)
				}
				if state != model.ArticleStateDraft {
					t.Fatalf("got state %q, want %q", state, model.ArticleStateDraft)
				}
				if title != "new title" {
					t.Fatalf("got title %q, want %q", title, "new title")
				}
				if content != "new content" {
					t.Fatalf("got content %q, want %q", content, "new content")
				}
				return true, nil
			},
		}

		service := NewArticleService(repo)

		err := service.UpdateArticle(context.Background(), 1, 100, "new title", "new content")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Fatal("expected UpdateContentIfAuthorAndState to be called")
		}
	})
}

func TestArticleService_DeleteArticle(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return "", false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{}, sql.ErrNoRows
			},
		}

		service := NewArticleService(repo)

		err := service.DeleteArticle(context.Background(), 1, 100)
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("not author", func(t *testing.T) {
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return "", false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 200, State: model.ArticleStatePublished}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.DeleteArticle(context.Background(), 1, 100)
		assertErrIs(t, err, ErrPermissionDenied)
	})

	t.Run("condition failed for current author returns not found", func(t *testing.T) {
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return "", false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{ID: id, AuthorID: 100, State: model.ArticleStatePublished}, nil
			},
		}

		service := NewArticleService(repo)

		err := service.DeleteArticle(context.Background(), 1, 100)
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("delete error", func(t *testing.T) {
		wantErr := errors.New("delete failed")
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return "", false, wantErr
			},
		}

		service := NewArticleService(repo)

		err := service.DeleteArticle(context.Background(), 1, 100)
		assertErrIs(t, err, wantErr)
	})

	t.Run("explain error", func(t *testing.T) {
		wantErr := errors.New("get failed")
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return "", false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{}, wantErr
			},
		}

		service := NewArticleService(repo)

		err := service.DeleteArticle(context.Background(), 1, 100)
		assertErrIs(t, err, wantErr)
	})

	t.Run("delete draft does not delete published articles cache", func(t *testing.T) {
		deleteCalled := false
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				if id != 1 {
					t.Fatalf("got id %d, want 1", id)
				}
				if authorID != 100 {
					t.Fatalf("got author id %d, want 100", authorID)
				}
				return model.ArticleStateDraft, true, nil
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				deleteCalled = true
				return nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.DeleteArticle(context.Background(), 1, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleteCalled {
			t.Fatal("did not expect cache Delete to be called")
		}
	})

	t.Run("delete published article deletes published articles cache", func(t *testing.T) {
		deleteCacheCalled := false
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				if id != 1 {
					t.Fatalf("got id %d, want 1", id)
				}
				if authorID != 100 {
					t.Fatalf("got author id %d, want 100", authorID)
				}
				return model.ArticleStatePublished, true, nil
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				deleteCacheCalled = true
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.DeleteArticle(context.Background(), 1, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deleteCacheCalled {
			t.Fatal("expected cache Delete to be called")
		}
	})

	t.Run("failed condition does not delete cache", func(t *testing.T) {
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return "", false, nil
			},
			getByIDFunc: func(ctx context.Context, id int64) (model.Article, error) {
				return model.Article{}, sql.ErrNoRows
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				t.Fatal("did not expect cache Delete to be called")
				return nil
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.DeleteArticle(context.Background(), 1, 100)
		assertErrIs(t, err, ErrArticleNotFound)
	})

	t.Run("cache delete error does not block published delete", func(t *testing.T) {
		repo := &fakeArticleRepo{
			deleteIfAuthorAndNotDeletedFunc: func(ctx context.Context, id, authorID int64) (string, bool, error) {
				return model.ArticleStatePublished, true, nil
			},
		}
		cache := &fakeArticleCache{
			deleteFunc: func(ctx context.Context, key string) error {
				if key != publishedArticlesCacheKey {
					t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
				}
				return errors.New("redis unavailable")
			},
		}

		service := NewArticleServiceWithCache(repo, cache)

		err := service.DeleteArticle(context.Background(), 1, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

type memoryArticleCache struct {
	t      *testing.T
	mu     sync.Mutex
	values map[string]string
}

func newMemoryArticleCache(t *testing.T) *memoryArticleCache {
	t.Helper()
	return &memoryArticleCache{
		t:      t,
		values: make(map[string]string),
	}
}

func (c *memoryArticleCache) Get(ctx context.Context, key string) (string, error) {
	if key != publishedArticlesCacheKey {
		c.t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[key], nil
}

func (c *memoryArticleCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if key != publishedArticlesCacheKey {
		c.t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
	}
	if ttl != publishedArticlesCacheTTL {
		c.t.Fatalf("got ttl %v, want %v", ttl, publishedArticlesCacheTTL)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
	return nil
}

func (c *memoryArticleCache) Delete(ctx context.Context, key string) error {
	if key != publishedArticlesCacheKey {
		c.t.Fatalf("got cache key %q, want %q", key, publishedArticlesCacheKey)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, key)
	return nil
}

func (c *memoryArticleCache) value(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[key]
}

func containsArticleID(articles []model.Article, id int64) bool {
	for _, article := range articles {
		if article.ID == id {
			return true
		}
	}
	return false
}

func cacheValueContainsArticleID(t *testing.T, value string, id int64) bool {
	t.Helper()
	var articles []model.Article
	if err := json.Unmarshal([]byte(value), &articles); err != nil {
		t.Fatalf("unmarshal cached value: %v", err)
	}
	return containsArticleID(articles, id)
}
