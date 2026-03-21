package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"content-backend/internal/auth"
)

type contextKey string

const authClaimsContextKey contextKey = "authClaims"

type AuthMiddleware struct {
	tokenManager *auth.TokenManager
}

func NewAuthMiddleware(tokenManager *auth.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{tokenManager: tokenManager}
}

func (m *AuthMiddleware) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractBearerToken(r.Header.Get("Authorization"))
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims, err := m.tokenManager.ValidateAndParse(token)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidTokenConfig) {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrExpiredToken) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), authClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	claims, ok := ctx.Value(authClaimsContextKey).(auth.Claims)
	if !ok {
		return auth.Claims{}, false
	}

	return claims, true
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return 0, false
	}

	return claims.UserID, true
}

func extractBearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}

	return token, true
}
