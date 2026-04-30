package handler

import (
	"content-backend/internal/middleware"
	"content-backend/internal/model"
	"content-backend/internal/service"
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type articleService interface {
	CreateArticle(ctx context.Context, authorID int64, title, content string) (int64, error)
	PublishArticle(ctx context.Context, articleID, currentUserID int64) error
	ListPublishedArticles(ctx context.Context) ([]model.Article, error)
	ListMyArticles(ctx context.Context, authorID int64) ([]model.Article, error)
	GetArticle(ctx context.Context, articleID int64, viewer service.ArticleViewer) (model.Article, error)
	UpdateArticle(ctx context.Context, articleID, currentUserID int64, title string, content string) error
}

type ArticleHandler struct {
	articleService articleService
}

func NewArticleHandler(articleService articleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

func (h *ArticleHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req CreateArticleRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	id, err := h.articleService.CreateArticle(r.Context(), currentUserID, req.Title, req.Content)
	if writeArticleServiceError(w, err) {
		return
	}

	res := CreateArticleResponse{ID: id}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *ArticleHandler) PublishArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req PublishArticleRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = h.articleService.PublishArticle(r.Context(), req.ArticleID, currentUserID)
	if writeArticleServiceError(w, err) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ArticleHandler) ListPublishedArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	articles, err := h.articleService.ListPublishedArticles(r.Context())
	if writeArticleServiceError(w, err) {
		return
	}

	responses := toArticleResponses(articles)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responses)
}

func (h *ArticleHandler) ListMyArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	currentUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	articles, err := h.articleService.ListMyArticles(r.Context(), currentUserID)
	if writeArticleServiceError(w, err) {
		return
	}

	responses := toArticleResponses(articles)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responses)
}

func (h *ArticleHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id, err := parseArticleID(r.URL.Path)
	if errors.Is(err, ErrInvalidArticleID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	viewer := service.ArticleViewer{}
	if currentUserID, ok := middleware.UserIDFromContext(r.Context()); ok {
		viewer = service.ArticleViewer{
			UserID:        currentUserID,
			Authenticated: true,
		}
	}

	article, err := h.articleService.GetArticle(r.Context(), id, viewer)
	if writeArticleServiceError(w, err) {
		return
	}

	res := toArticleDetailResponse(article)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func (h *ArticleHandler) UpdateArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	currentUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	articleID, err := parseMyArticleID(r.URL.Path)
	if errors.Is(err, ErrInvalidArticleID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req UpdateArticleRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = h.articleService.UpdateArticle(r.Context(), articleID, currentUserID, req.Title, req.Content)
	if writeArticleServiceError(w, err) {
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}
