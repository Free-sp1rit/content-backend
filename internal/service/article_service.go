package service

import (
	"context"
	"database/sql"
	"errors"

	"content-backend/internal/model"
)

var ErrArticleNotFound = errors.New("article not found")
var ErrPermissionDenied = errors.New("permission denied")
var ErrArticleNotEditable = errors.New("article not editable")

type articleRepository interface {
	Create(ctx context.Context, article model.Article) (int64, error)
	GetByID(ctx context.Context, id int64) (model.Article, error)
	UpdateState(ctx context.Context, id int64, state string) error
	ListByState(ctx context.Context, state string) ([]model.Article, error)
	ListByAuthorID(ctx context.Context, authorID int64) ([]model.Article, error)
	UpdateContent(ctx context.Context, id int64, title, content string) error
}

type ArticleService struct {
	articleRepo articleRepository
}

func NewArticleService(articleRepo articleRepository) *ArticleService {
	return &ArticleService{articleRepo: articleRepo}
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

	if article.State == model.ArticleStatePublished {
		return nil
	}

	err = s.articleRepo.UpdateState(ctx, articleID, model.ArticleStatePublished)
	if err != nil {
		return err
	}

	return nil
}

func (s *ArticleService) ListPublishedArticles(ctx context.Context) ([]model.Article, error) {
	articles, err := s.articleRepo.ListByState(ctx, model.ArticleStatePublished)
	if err != nil {
		return nil, err
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

func (s *ArticleService) GetArticle(ctx context.Context, articleID int64) (model.Article, error) {
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

	return article, nil
}

func (s *ArticleService) UpdateArticle(ctx context.Context, articleID, currentUserID int64, title string, content string) error {
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
	if article.State != model.ArticleStateDraft {
		return ErrArticleNotEditable
	}

	err = s.articleRepo.UpdateContent(ctx, articleID, title, content)
	if err != nil {
		return err
	}
	return nil
}
