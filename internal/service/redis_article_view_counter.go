package service

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const articleViewCountKeyPrefix = "article:views:"
const articleUserViewCountKeyPrefix = "article:user_views:"
const articleViewedKeyPrefix = "article:viewed:"

const defaultArticleUserViewDedupWindow = 24 * time.Hour

type RedisArticleViewCounter struct {
	client              *redis.Client
	userViewDedupWindow time.Duration
}

func NewRedisArticleViewCounter(client *redis.Client) *RedisArticleViewCounter {
	return NewRedisArticleViewCounterWithOptions(client, defaultArticleUserViewDedupWindow)
}

func NewRedisArticleViewCounterWithOptions(client *redis.Client, userViewDedupWindow time.Duration) *RedisArticleViewCounter {
	if userViewDedupWindow <= 0 {
		userViewDedupWindow = defaultArticleUserViewDedupWindow
	}

	return &RedisArticleViewCounter{
		client:              client,
		userViewDedupWindow: userViewDedupWindow,
	}
}

func (c *RedisArticleViewCounter) Increment(ctx context.Context, articleID int64) error {
	return c.client.Incr(ctx, articleViewCountKey(articleID)).Err()
}

func (c *RedisArticleViewCounter) IncrementAuthenticated(ctx context.Context, articleID, userID int64) error {
	firstView, err := c.client.SetNX(
		ctx,
		articleViewedKey(articleID, userID),
		"1",
		c.userViewDedupWindow,
	).Result()
	if err != nil {
		return err
	}
	if !firstView {
		return nil
	}

	return c.client.Incr(ctx, articleUserViewCountKey(articleID)).Err()
}

func articleViewCountKey(articleID int64) string {
	return articleViewCountKeyPrefix + strconv.FormatInt(articleID, 10)
}

func articleUserViewCountKey(articleID int64) string {
	return articleUserViewCountKeyPrefix + strconv.FormatInt(articleID, 10)
}

func articleViewedKey(articleID, userID int64) string {
	return articleViewedKeyPrefix +
		strconv.FormatInt(articleID, 10) +
		":user:" +
		strconv.FormatInt(userID, 10)
}
