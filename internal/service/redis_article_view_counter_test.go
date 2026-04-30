package service

import (
	"context"
	"testing"

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

func TestArticleViewCountKey(t *testing.T) {
	got := articleViewCountKey(42)
	want := "article:views:42"
	if got != want {
		t.Fatalf("got key %q, want %q", got, want)
	}
}
