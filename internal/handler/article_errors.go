package handler

import (
	"content-backend/internal/service"
	"errors"
	"net/http"
)

func statusFromArticleServiceError(err error) (int, bool) {
	switch {
	case errors.Is(err, service.ErrArticleNotFound):
		return http.StatusNotFound, true
	case errors.Is(err, service.ErrPermissionDenied):
		return http.StatusForbidden, true
	case errors.Is(err, service.ErrArticleNotEditable):
		return http.StatusConflict, true
	case errors.Is(err, service.ErrArticleNotPublishable):
		return http.StatusConflict, true
	default:
		return 0, false
	}
}

func writeArticleServiceError(w http.ResponseWriter, err error) bool {
	if status, ok := statusFromArticleServiceError(err); ok {
		w.WriteHeader(status)
		return true
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	}
	return false
}
