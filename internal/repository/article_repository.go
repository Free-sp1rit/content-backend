package repository

import (
	"context"

	"content-backend/internal/model"
)

type ArticleRepository interface {
	Create(ctx context.Context, article *model.Article) error
	Update(ctx context.Context, article *model.Article) error
	GetByID(ctx context.Context, id int64) (*model.Article, error)
	ListByAuthor(ctx context.Context, authorID int64, limit, offset int) ([]model.Article, error)
}
