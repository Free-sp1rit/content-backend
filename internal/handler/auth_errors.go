package handler

import (
	"content-backend/internal/service"
	"errors"
	"net/http"
)

func statusFromAuthServiceError(err error) (int, bool) {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		return http.StatusConflict, true
	case errors.Is(err, service.ErrInvalidCredentials):
		return http.StatusUnauthorized, true
	default:
		return 0, false
	}
}

func writeAuthServiceError(w http.ResponseWriter, err error) bool {
	if status, ok := statusFromAuthServiceError(err); ok {
		w.WriteHeader(status)
		return true
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	}
	return false
}
