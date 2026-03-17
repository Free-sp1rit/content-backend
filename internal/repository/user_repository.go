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
	panic("not implemented")
}

func (r *UserRepository) Create(ctx context.Context, user model.User) (int64, error) {
	panic("not implemented")
}
