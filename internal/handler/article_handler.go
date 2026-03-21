package handler

import (
	"content-backend/internal/service"
	"encoding/json"
	"errors"
	"net/http"
)

type ArticleHandler struct {
	articleService *service.ArticleService
}

func NewArticleHandler(articleService *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

func (h *ArticleHandler) Articles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.CreateArticle(w, r)
	case http.MethodGet:
		h.ListPublishedArticles(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type CreateArticleRequest struct {
	AuthorID int64 `json:"author_id"`
	Title string `json:"title"`
	Content string `json:"content"`
}

type CreateArticleResponse struct {
	ID int64 `json:"id"`
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

	id, err := h.articleService.CreateArticle(r.Context(), req.AuthorID, req.Title, req.Content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := CreateArticleResponse{ID: id}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}


type PublishArticleRequest struct {
	ArticleID int64 `json:"article_id"`
	CurrentUserID int64 `json:"current_user_id"`
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

	err = h.articleService.PublishArticle(r.Context(), req.ArticleID, req.CurrentUserID)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if errors.Is(err ,service.ErrPermissionDenied) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
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
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ =	json.NewEncoder(w).Encode(articles)
}


type ListMyArticlesRequest struct {
	AuthorID int64 `json:"author_id"`
} 

func (h *ArticleHandler) ListMyArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req ListMyArticlesRequest
	json.NewDecoder(r.Body).Decode(&req)
	articles, err := h.articleService.ListMyArticles(r.Context(), req.AuthorID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ =	json.NewEncoder(w).Encode(articles)
}