package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"

	"content-backend/internal/model"

	"golang.org/x/sync/singleflight"
)

var ErrArticleNotFound = errors.New("article not found")
var ErrPermissionDenied = errors.New("permission denied")
var ErrArticleNotEditable = errors.New("article not editable")
var ErrArticleNotPublishable = errors.New("article not publishable")

type articleRepository interface {
	Create(ctx context.Context, article model.Article) (int64, error)
	GetByID(ctx context.Context, id int64) (model.Article, error)
	UpdateStateIfAuthorAndState(ctx context.Context, id, authorID int64, currentState, nextState string) (bool, error)
	ListByState(ctx context.Context, state string) ([]model.Article, error)
	ListByAuthorID(ctx context.Context, authorID int64) ([]model.Article, error)
	UpdateContentIfAuthorAndState(ctx context.Context, id, authorID int64, state, title, content string) (bool, error)
}

type articleCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type articleViewCounter interface {
	Increment(ctx context.Context, articleID int64) error
	IncrementAuthenticated(ctx context.Context, articleID, userID int64) error
}

type ArticleViewer struct {
	UserID        int64
	Authenticated bool
}

type ArticleService struct {
	articleRepo            articleRepository
	cache                  articleCache
	viewCounter            articleViewCounter
	publishedArticlesGroup singleflight.Group
}

func NewArticleService(articleRepo articleRepository) *ArticleService {
	return &ArticleService{articleRepo: articleRepo}
}

func NewArticleServiceWithCache(articleRepo articleRepository, cache articleCache) *ArticleService {
	return &ArticleService{
		articleRepo: articleRepo,
		cache:       cache,
	}
}

func NewArticleServiceWithViewCounter(articleRepo articleRepository, viewCounter articleViewCounter) *ArticleService {
	return &ArticleService{
		articleRepo: articleRepo,
		viewCounter: viewCounter,
	}
}

func NewArticleServiceWithCacheAndViewCounter(articleRepo articleRepository, cache articleCache, viewCounter articleViewCounter) *ArticleService {
	return &ArticleService{
		articleRepo: articleRepo,
		cache:       cache,
		viewCounter: viewCounter,
	}
}

func (s *ArticleService) CreateArticle(ctx context.Context, authorID int64, title, content string) (int64, error) {
	article := model.Article{
		AuthorID: authorID,
		Title:    title,
		Content:  content,
	}

	id, err := s.articleRepo.Create(ctx, article)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ArticleService) PublishArticle(ctx context.Context, articleID, currentUserID int64) error {
	updated, err := s.articleRepo.UpdateStateIfAuthorAndState(
		ctx,
		articleID,
		currentUserID,
		model.ArticleStateDraft,
		model.ArticleStatePublished,
	)
	if err != nil {
		return err
	}
	if !updated {
		return s.explainPublishArticleFailure(ctx, articleID, currentUserID)
	}

	s.deletePublishedArticlesCache(ctx)

	return nil
}

func (s *ArticleService) explainPublishArticleFailure(ctx context.Context, articleID, currentUserID int64) error {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrArticleNotFound
	}
	if err != nil {
		return err
	}

	if article.AuthorID != currentUserID {
		return ErrPermissionDenied
	}

	return ErrArticleNotPublishable
}

func (s *ArticleService) ListPublishedArticles(ctx context.Context) ([]model.Article, error) {
	if s.cache != nil {
		cachedArticles, ok := s.getPublishedArticlesFromCache(ctx)
		if ok {
			return cachedArticles, nil
		}
	}

	result, err, _ := s.publishedArticlesGroup.Do(publishedArticlesCacheKey, func() (any, error) {
		if s.cache != nil {
			cachedArticles, ok := s.getPublishedArticlesFromCache(ctx)
			if ok {
				return cachedArticles, nil
			}
		}

		return s.listPublishedArticlesFromRepository(ctx)
	})
	if err != nil {
		return nil, err
	}

	articles, ok := result.([]model.Article)
	if !ok {
		return nil, errors.New("unexpected published articles result type")
	}

	return articles, nil
}

func (s *ArticleService) listPublishedArticlesFromRepository(ctx context.Context) ([]model.Article, error) {
	articles, err := s.articleRepo.ListByState(ctx, model.ArticleStatePublished)
	if err != nil {
		return nil, err
	}
	if articles == nil {
		articles = []model.Article{}
	}

	if s.cache != nil {
		s.setPublishedArticlesCache(ctx, articles)
	}

	return articles, nil
}

func (s *ArticleService) ListMyArticles(ctx context.Context, authorID int64) ([]model.Article, error) {
	articles, err := s.articleRepo.ListByAuthorID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *ArticleService) GetArticle(ctx context.Context, articleID int64, viewer ArticleViewer) (model.Article, error) {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Article{}, ErrArticleNotFound
	}
	if err != nil {
		return model.Article{}, err
	}
	if article.State != model.ArticleStatePublished {
		return model.Article{}, ErrArticleNotFound
	}

	s.incrementArticleViewCount(ctx, article.ID, viewer)

	return article, nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, articleID, currentUserID int64, title string, content string) error {
	updated, err := s.articleRepo.UpdateContentIfAuthorAndState(
		ctx,
		articleID,
		currentUserID,
		model.ArticleStateDraft,
		title,
		content,
	)
	if err != nil {
		return err
	}
	if !updated {
		return s.explainUpdateArticleFailure(ctx, articleID, currentUserID)
	}

	return nil
}

func (s *ArticleService) explainUpdateArticleFailure(ctx context.Context, articleID, currentUserID int64) error {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrArticleNotFound
	}
	if err != nil {
		return err
	}
	if article.AuthorID != currentUserID {
		return ErrPermissionDenied
	}

	return ErrArticleNotEditable
}

const publishedArticlesCacheKey = "articles:published"

const publishedArticlesCacheTTL = 5 * time.Minute

func (s *ArticleService) getPublishedArticlesFromCache(ctx context.Context) ([]model.Article, bool) {
	value, err := s.cache.Get(ctx, publishedArticlesCacheKey)
	if err != nil {
		return nil, false
	}
	if value == "" {
		return nil, false
	}

	var articles []model.Article
	err = json.Unmarshal([]byte(value), &articles)
	if err != nil {
		log.Printf("decode published articles cache: %v", err)
		return nil, false
	}
	if articles == nil {
		articles = []model.Article{}
	}

	return articles, true
}

func (s *ArticleService) setPublishedArticlesCache(ctx context.Context, articles []model.Article) {
	data, err := json.Marshal(articles)
	if err != nil {
		log.Printf("encode published articles cache: %v", err)
		return
	}

	err = s.cache.Set(ctx, publishedArticlesCacheKey, string(data), publishedArticlesCacheTTL)
	if err != nil {
		log.Printf("set published articles cache: %v", err)
	}
}

func (s *ArticleService) deletePublishedArticlesCache(ctx context.Context) {
	if s.cache == nil {
		return
	}

	err := s.cache.Delete(ctx, publishedArticlesCacheKey)
	if err != nil {
		log.Printf("delete published articles cache: %v", err)
	}
}

func (s *ArticleService) incrementArticleViewCount(ctx context.Context, articleID int64, viewer ArticleViewer) {
	if s.viewCounter == nil {
		return
	}

	err := s.viewCounter.Increment(ctx, articleID)
	if err != nil {
		log.Printf("increment article view count: %v", err)
	}

	if !viewer.Authenticated {
		return
	}

	err = s.viewCounter.IncrementAuthenticated(ctx, articleID, viewer.UserID)
	if err != nil {
		log.Printf("increment authenticated article view count: %v", err)
	}
}
