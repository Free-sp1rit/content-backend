package service

import (
	"context"
	"errors"
	"testing"
	"time"

	redismock "github.com/go-redis/redismock/v9"
)

func TestRedisCache_Get(t *testing.T) {
	t.Run("missing key returns empty value", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		cache := NewRedisCache(client)

		mock.ExpectGet("articles:published").RedisNil()

		got, err := cache.Get(context.Background(), "articles:published")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("got value %q, want empty", got)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("returns cached value", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		cache := NewRedisCache(client)

		mock.ExpectGet("articles:published").SetVal(`[{"id":1}]`)

		got, err := cache.Get(context.Background(), "articles:published")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != `[{"id":1}]` {
			t.Fatalf("got value %q, want cached json", got)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("returns command error", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		cache := NewRedisCache(client)
		wantErr := errors.New("redis get failed")

		mock.ExpectGet("articles:published").SetErr(wantErr)

		_, err := cache.Get(context.Background(), "articles:published")
		if !errors.Is(err, wantErr) {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
		assertRedisExpectationsMet(t, mock)
	})
}

func TestRedisCache_Set(t *testing.T) {
	client, mock := redismock.NewClientMock()
	cache := NewRedisCache(client)
	ttl := 5 * time.Minute

	mock.ExpectSet("articles:published", "[]", ttl).SetVal("OK")

	err := cache.Set(context.Background(), "articles:published", "[]", ttl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRedisExpectationsMet(t, mock)
}

func TestRedisCache_Delete(t *testing.T) {
	t.Run("deletes key", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		cache := NewRedisCache(client)

		mock.ExpectDel("articles:published").SetVal(1)

		err := cache.Delete(context.Background(), "articles:published")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("returns command error", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		cache := NewRedisCache(client)
		wantErr := errors.New("redis delete failed")

		mock.ExpectDel("articles:published").SetErr(wantErr)

		err := cache.Delete(context.Background(), "articles:published")
		if !errors.Is(err, wantErr) {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
		assertRedisExpectationsMet(t, mock)
	})
}
