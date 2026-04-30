package service

import (
	"context"
	"errors"
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

		mock.ExpectEvalSha(
			incrementAuthenticatedArticleViewScript.Hash(),
			[]string{"article:viewed:42:user:7", "article:user_views:42"},
			"3600",
		).SetVal(int64(1))

		err := counter.IncrementAuthenticated(context.Background(), 42, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("duplicate user view does not increment user view count", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		counter := NewRedisArticleViewCounterWithOptions(client, time.Hour)

		mock.ExpectEvalSha(
			incrementAuthenticatedArticleViewScript.Hash(),
			[]string{"article:viewed:42:user:7", "article:user_views:42"},
			"3600",
		).SetVal(int64(0))

		err := counter.IncrementAuthenticated(context.Background(), 42, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertRedisExpectationsMet(t, mock)
	})

	t.Run("script error is returned to service", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		counter := NewRedisArticleViewCounterWithOptions(client, time.Hour)
		wantErr := errors.New("redis script failed")

		mock.ExpectEvalSha(
			incrementAuthenticatedArticleViewScript.Hash(),
			[]string{"article:viewed:42:user:7", "article:user_views:42"},
			"3600",
		).SetErr(wantErr)

		err := counter.IncrementAuthenticated(context.Background(), 42, 7)
		if !errors.Is(err, wantErr) {
			t.Fatalf("got error %v, want %v", err, wantErr)
		}
		assertRedisExpectationsMet(t, mock)
	})
}

func TestRedisArticleViewCounter_UserViewDedupWindowSeconds(t *testing.T) {
	t.Run("uses configured whole seconds", func(t *testing.T) {
		counter := NewRedisArticleViewCounterWithOptions(nil, time.Hour)

		got := counter.userViewDedupWindowSeconds()
		want := "3600"
		if got != want {
			t.Fatalf("got window seconds %q, want %q", got, want)
		}
	})

	t.Run("keeps positive subsecond windows valid for Redis EX", func(t *testing.T) {
		counter := NewRedisArticleViewCounterWithOptions(nil, time.Millisecond)

		got := counter.userViewDedupWindowSeconds()
		want := "1"
		if got != want {
			t.Fatalf("got window seconds %q, want %q", got, want)
		}
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
