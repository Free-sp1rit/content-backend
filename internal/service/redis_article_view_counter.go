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

var incrementAuthenticatedArticleViewScript = redis.NewScript(`
if redis.call("SET", KEYS[1], "1", "NX", "EX", ARGV[1]) then
	return redis.call("INCR", KEYS[2])
end
return 0
`)

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
	return incrementAuthenticatedArticleViewScript.Run(
		ctx,
		c.client,
		[]string{
			articleViewedKey(articleID, userID),
			articleUserViewCountKey(articleID),
		},
		c.userViewDedupWindowSeconds(),
	).Err()
}

func (c *RedisArticleViewCounter) userViewDedupWindowSeconds() string {
	seconds := int64(c.userViewDedupWindow / time.Second)
	if seconds < 1 {
		seconds = 1
	}

	return strconv.FormatInt(seconds, 10)
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
