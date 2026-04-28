package service

import (
	"context"
	"testing"

	redismock "github.com/go-redis/redismock/v9"
)

func TestRedisLoginLimiter_TooManyAttempts(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectGet("login:failures:email:user@example.com").RedisNil()

		got, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("expected too many attempts to be false")
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("below limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectGet("login:failures:email:user@example.com").SetVal("4")

		got, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("expected too many attempts to be false")
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("at limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectGet("login:failures:email:user@example.com").SetVal("5")

		got, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("expected too many attempts to be true")
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("ip limiter has higher default limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginIPLimiter(client)

		mock.ExpectGet("login:failures:ip:192.0.2.1").SetVal("19")

		got, err := limiter.TooManyAttempts(context.Background(), "login:failures:ip:192.0.2.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("expected too many attempts to be false")
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("ip limiter at limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginIPLimiter(client)

		mock.ExpectGet("login:failures:ip:192.0.2.1").SetVal("20")

		got, err := limiter.TooManyAttempts(context.Background(), "login:failures:ip:192.0.2.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("expected too many attempts to be true")
		}
		assertRedisExpectationsMet(t, mock)
	})
}

func TestRedisLoginLimiter_RecordFailure(t *testing.T) {
	t.Run("runs atomic script", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectEvalSha(
			recordLoginFailureScript.Hash(),
			[]string{"login:failures:email:user@example.com"},
			"600",
		).SetVal(int64(1))

		err := limiter.RecordFailure(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertRedisExpectationsMet(t, mock)
	})
}

func TestRedisLoginLimiter_Reset(t *testing.T) {
	client, mock := redismock.NewClientMock()
	limiter := NewRedisLoginLimiter(client)

	mock.ExpectDel("login:failures:email:user@example.com").SetVal(1)

	err := limiter.Reset(context.Background(), "login:failures:email:user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRedisExpectationsMet(t, mock)
}

func assertRedisExpectationsMet(t *testing.T, mock redismock.ClientMock) {
	t.Helper()

	err := mock.ExpectationsWereMet()
	if err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}
