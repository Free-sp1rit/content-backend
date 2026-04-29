package service

import (
	"context"
	"testing"
	"time"

	redismock "github.com/go-redis/redismock/v9"
)

func TestRedisLoginLimiter_TooManyAttempts(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectGet("login:failures:email:user@example.com").RedisNil()

		got, retryAfter, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("expected too many attempts to be false")
		}
		if retryAfter != 0 {
			t.Fatalf("got retry after %v, want zero", retryAfter)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("below limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectGet("login:failures:email:user@example.com").SetVal("4")

		got, retryAfter, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("expected too many attempts to be false")
		}
		if retryAfter != 0 {
			t.Fatalf("got retry after %v, want zero", retryAfter)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("at limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiter(client)

		mock.ExpectGet("login:failures:email:user@example.com").SetVal("5")
		mock.ExpectTTL("login:failures:email:user@example.com").SetVal(3 * time.Minute)

		got, retryAfter, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("expected too many attempts to be true")
		}
		if retryAfter != 3*time.Minute {
			t.Fatalf("got retry after %v, want %v", retryAfter, 3*time.Minute)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("ip limiter has higher default limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginIPLimiter(client)

		mock.ExpectGet("login:failures:ip:192.0.2.1").SetVal("19")

		got, retryAfter, err := limiter.TooManyAttempts(context.Background(), "login:failures:ip:192.0.2.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("expected too many attempts to be false")
		}
		if retryAfter != 0 {
			t.Fatalf("got retry after %v, want zero", retryAfter)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("ip limiter at limit", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginIPLimiter(client)

		mock.ExpectGet("login:failures:ip:192.0.2.1").SetVal("20")
		mock.ExpectTTL("login:failures:ip:192.0.2.1").SetVal(90 * time.Second)

		got, retryAfter, err := limiter.TooManyAttempts(context.Background(), "login:failures:ip:192.0.2.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("expected too many attempts to be true")
		}
		if retryAfter != 90*time.Second {
			t.Fatalf("got retry after %v, want %v", retryAfter, 90*time.Second)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("at limit falls back to window when ttl is not positive", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		limiter := NewRedisLoginLimiterWithOptions(client, 5, 10*time.Minute)

		mock.ExpectGet("login:failures:email:user@example.com").SetVal("5")
		mock.ExpectTTL("login:failures:email:user@example.com").SetVal(-1)

		got, retryAfter, err := limiter.TooManyAttempts(context.Background(), "login:failures:email:user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("expected too many attempts to be true")
		}
		if retryAfter != 10*time.Minute {
			t.Fatalf("got retry after %v, want %v", retryAfter, 10*time.Minute)
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
