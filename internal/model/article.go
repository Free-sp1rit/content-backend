package model

import "time"

const (
	ArticleStateDraft     = "draft"
	ArticleStatePublished = "published"
)

type Article struct {
	ID        int64
	AuthorID  int64
	Title     string
	Content   string
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
