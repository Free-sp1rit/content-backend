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
	const query = `
		INSERT INTO articles(author_id, title, content)
		VALUES($1, $2, $3)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		article.AuthorID,
		article.Title,
		article.Content,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *ArticleRepository) GetByID(ctx context.Context, id int64) (model.Article, error) {
	const query = `
		SELECT id, author_id, title, content, state, created_at, updated_at
		FROM articles
		WHERE id = $1
	`

	var article model.Article
	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&article.ID,
		&article.AuthorID,
		&article.Title,
		&article.Content,
		&article.State,
		&article.CreatedAt,
		&article.UpdatedAt,
	)

	if err != nil {
		return model.Article{}, err
	}

	return article, nil
}

func (r *ArticleRepository) UpdateState(ctx context.Context, id int64, state string) error {
	const query = `
		UPDATE articles
		SET state = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
		state,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *ArticleRepository) UpdateContent(ctx context.Context, id int64, title string, content string) error {
	const query = `
		UPDATE articles
		SET title = $2, content = $3, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		id,
		title,
		content,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *ArticleRepository) ListByState(ctx context.Context, state string) ([]model.Article, error) {
	const query = `
		SELECT id, author_id, title, content, state, created_at, updated_at
		FROM articles
		WHERE state = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		state,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var article model.Article
		err := rows.Scan(
			&article.ID,
			&article.AuthorID,
			&article.Title,
			&article.Content,
			&article.State,
			&article.CreatedAt,
			&article.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (r *ArticleRepository) ListByAuthorID(ctx context.Context, authorID int64) ([]model.Article, error) {
	const query = `
		SELECT id, author_id, title, content, state, created_at, updated_at
		FROM articles
		WHERE author_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		authorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []model.Article
	for rows.Next() {
		var article model.Article
		err := rows.Scan(
			&article.ID,
			&article.AuthorID,
			&article.Title,
			&article.Content,
			&article.State,
			&article.CreatedAt,
			&article.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return articles, nil
}
