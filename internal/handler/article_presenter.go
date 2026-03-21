package handler

import (
	"content-backend/internal/model"
	"time"
	"errors"
	"strings"
	"strconv"
)

var ErrInvalidArticleID = errors.New("invalid article id")

type ArticleResponse struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ArticleDetailResponse struct {
	ID        int64     `json:"id"`
	AuthorID  int64     `json:"author_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toArticleResponse(article model.Article) ArticleResponse {
	return ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		State:     article.State,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
}

func toArticleResponses(articles []model.Article) []ArticleResponse {
	responses := make([]ArticleResponse, 0, len(articles))
	for _, article := range articles {
		responses = append(responses, toArticleResponse(article))
	}

	return responses
}

func toArticleDetailResponse(article model.Article) ArticleDetailResponse {
	return ArticleDetailResponse{
		ID:        article.ID,
		AuthorID:  article.AuthorID,
		Title:     article.Title,
		Content: 	 article.Content,
		State:     article.State,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
}

func parseArticleID(path string) (int64, error) {
	const prefix = "/articles/"

	if !strings.HasPrefix(path, prefix) {
		return 0, ErrInvalidArticleID
	}

	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" {
		return 0, ErrInvalidArticleID
	}

	// 不接受多余层级，例如 /articles/123/extra
	if strings.Contains(idPart, "/") {
		return 0, ErrInvalidArticleID
	}

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidArticleID
	}

	return id, nil
}