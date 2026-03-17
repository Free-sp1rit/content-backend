package repository

import (
	"context"
	"database/sql"

	"content-backend/internal/model"
)

type ArticleRepository struct {
	db *sql.DB
}

func NewArticleRepository(db *sql.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) Create(ctx context.Context, article model.Article) (int64, error) {
	panic("not implemented")
}

func (r *ArticleRepository) GetByID(ctx context.Context, id int64) (model.Article, error) {
	panic("not implemented")
}

func (r *ArticleRepository) UpdateState(ctx context.Context, id int64, state string) error {
	panic("not implemented")
}

func (r *ArticleRepository) UpdateContent(ctx context.Context, id int64, title string, content string) error {
	panic("not implemented")
}

func (r *ArticleRepository) ListByState(ctx context.Context, state string) ([]model.Article, error) {
	panic("not implemented")
}

func (r *ArticleRepository) ListByAuthorID(ctx context.Context, authorID int64) ([]model.Article, error) {
	panic("not implemented")
}
