package auth

import (
	"errors"
	"testing"
	"time"
)

const (
	testTokenSecret = "test-secret"
	testTokenIssuer = "test-issuer"
)

func assertErrIs(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got err %v, want %v", got, want)
	}
}

func mustGenerateToken(t *testing.T, tokenManager *TokenManager, userID int64) string {
	t.Helper()

	token, err := tokenManager.Generate(userID)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	return token
}

func TestTokenManager_GenerateAndValidate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ttl := 24 * time.Hour
		tokenManager := NewTokenManager(testTokenSecret, testTokenIssuer, ttl)

		userID := int64(1024)

		token := mustGenerateToken(t, tokenManager, userID)
		claims, err := tokenManager.ValidateAndParse(token)
		if err != nil {
			t.Fatalf("ValidateAndParse returned error: %v", err)
		}

		if claims.UserID != userID {
			t.Fatalf("got user id %d, want %d", claims.UserID, userID)
		}

		if claims.Issuer != testTokenIssuer {
			t.Fatalf("got issuer %q, want %q", claims.Issuer, testTokenIssuer)
		}
	})
}

func TestTokenManager_Generate(t *testing.T) {
	t.Run("empty secret", func(t *testing.T) {
		ttl := 24 * time.Hour
		tokenManager := NewTokenManager("", testTokenIssuer, ttl)

		userID := int64(1024)

		_, err := tokenManager.Generate(userID)
		assertErrIs(t, err, ErrInvalidTokenConfig)
	})
}

func TestTokenManager_ValidateAndParse(t *testing.T) {
	t.Run("invalid token format", func(t *testing.T) {
		tokenManager := NewTokenManager(testTokenSecret, testTokenIssuer, time.Hour)

		_, err := tokenManager.ValidateAndParse("bad-token")

		assertErrIs(t, err, ErrInvalidToken)
	})

	t.Run("issuer mismatch", func(t *testing.T) {
		generateManager := NewTokenManager(testTokenSecret, "issuer-a", time.Hour)
		validateManager := NewTokenManager(testTokenSecret, "issuer-b", time.Hour)

		token := mustGenerateToken(t, generateManager, 1024)

		_, err := validateManager.ValidateAndParse(token)
		assertErrIs(t, err, ErrInvalidToken)
	})

	t.Run("empty secret", func(t *testing.T) {
		tokenManager := NewTokenManager("", testTokenIssuer, time.Hour)

		_, err := tokenManager.ValidateAndParse("bad-token")

		assertErrIs(t, err, ErrInvalidTokenConfig)
	})

	t.Run("expired token", func(t *testing.T) {
		issuedAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		generateManager := NewTokenManager(testTokenSecret, testTokenIssuer, time.Hour)
		generateManager.now = func() time.Time {
			return issuedAt
		}

		token := mustGenerateToken(t, generateManager, 1024)

		validateManager := NewTokenManager(testTokenSecret, testTokenIssuer, time.Hour)
		validateManager.now = func() time.Time {
			return issuedAt.Add(time.Hour + time.Second)
		}

		_, err := validateManager.ValidateAndParse(token)

		assertErrIs(t, err, ErrExpiredToken)
	})
}
