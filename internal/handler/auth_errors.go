package handler

import (
	"content-backend/internal/service"
	"errors"
	"net/http"
	"strconv"
	"time"
)

func statusFromAuthServiceError(err error) (int, bool) {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyRegistered):
		return http.StatusConflict, true
	case errors.Is(err, service.ErrInvalidCredentials):
		return http.StatusUnauthorized, true
	case errors.Is(err, service.ErrLoginRateLimited):
		return http.StatusTooManyRequests, true
	default:
		return 0, false
	}
}

func writeAuthServiceError(w http.ResponseWriter, err error) bool {
	if status, ok := statusFromAuthServiceError(err); ok {
		if status == http.StatusTooManyRequests {
			setRetryAfterHeader(w, err)
		}
		w.WriteHeader(status)
		return true
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return true
	}
	return false
}

func setRetryAfterHeader(w http.ResponseWriter, err error) {
	retryAfter, ok := service.LoginRetryAfter(err)
	if !ok {
		return
	}

	w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSeconds(retryAfter), 10))
}

func retryAfterSeconds(d time.Duration) int64 {
	seconds := int64(d / time.Second)
	if d%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}
