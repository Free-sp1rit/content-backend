package service

import (
	"context"
	"testing"
	"time"

	redismock "github.com/go-redis/redismock/v9"
)

func TestRedisArticleViewCounter_Increment(t *testing.T) {
	client, mock := redismock.NewClientMock()
	counter := NewRedisArticleViewCounter(client)

	mock.ExpectIncr("article:views:42").SetVal(1)

	err := counter.Increment(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertRedisExpectationsMet(t, mock)
}

func TestRedisArticleViewCounter_IncrementAuthenticated(t *testing.T) {
	t.Run("first user view increments user view count", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		counter := NewRedisArticleViewCounterWithOptions(client, time.Hour)

		mock.ExpectSetNX("article:viewed:42:user:7", "1", time.Hour).SetVal(true)
		mock.ExpectIncr("article:user_views:42").SetVal(1)

		err := counter.IncrementAuthenticated(context.Background(), 42, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("duplicate user view does not increment user view count", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		counter := NewRedisArticleViewCounterWithOptions(client, time.Hour)

		mock.ExpectSetNX("article:viewed:42:user:7", "1", time.Hour).SetVal(false)

		err := counter.IncrementAuthenticated(context.Background(), 42, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertRedisExpectationsMet(t, mock)
	})
}

func TestArticleViewCountKey(t *testing.T) {
	got := articleViewCountKey(42)
	want := "article:views:42"
	if got != want {
		t.Fatalf("got key %q, want %q", got, want)
	}
}

func TestArticleUserViewCountKey(t *testing.T) {
	got := articleUserViewCountKey(42)
	want := "article:user_views:42"
	if got != want {
		t.Fatalf("got key %q, want %q", got, want)
	}
}

func TestArticleViewedKey(t *testing.T) {
	got := articleViewedKey(42, 7)
	want := "article:viewed:42:user:7"
	if got != want {
		t.Fatalf("got key %q, want %q", got, want)
	}
}
