package repository

import (
	"context"
	"database/sql"

	"content-backend/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (r *UserRepository) Create(ctx context.Context, user model.User) (int64, error) {
	const query = `
		INSERT INTO users(email, password_hash)
		VALUES($1, $2)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}
