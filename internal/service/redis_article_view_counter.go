package service

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const articleViewCountKeyPrefix = "article:views:"

type RedisArticleViewCounter struct {
	client *redis.Client
}

func NewRedisArticleViewCounter(client *redis.Client) *RedisArticleViewCounter {
	return &RedisArticleViewCounter{client: client}
}

func (c *RedisArticleViewCounter) Increment(ctx context.Context, articleID int64) error {
	return c.client.Incr(ctx, articleViewCountKey(articleID)).Err()
}

func articleViewCountKey(articleID int64) string {
	return articleViewCountKeyPrefix + strconv.FormatInt(articleID, 10)
}
