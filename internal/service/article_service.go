package service

import (
	"context"
	"database/sql"
	"errors"

	"content-backend/internal/model"
	"content-backend/internal/repository"
)

var ErrArticleNotFound = errors.New("article not found")
var ErrPermissionDenied = errors.New("permission denied")

type ArticleService struct {
	articleRepo *repository.ArticleRepository
}

func NewArticleService(articleRepo *repository.ArticleRepository) *ArticleService {
	return &ArticleService{articleRepo: articleRepo}
}

func (s *ArticleService) CreateArticle(ctx context.Context, authorID int64, title, content string) (int64, error) {
	article := model.Article{
		AuthorID: authorID,
		Title: title,
		Content: content,
	}

	id, err := s.articleRepo.Create(ctx, article)
	if(err != nil) {
		return 0, err
	}

	return id, nil
}

func (s *ArticleService) PublishArticle(ctx context.Context, articleID, currentUserID int64) error {
	article, err := s.articleRepo.GetByID(ctx, articleID)
	if err == sql.ErrNoRows {
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

