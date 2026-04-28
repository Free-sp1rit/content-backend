package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultLoginEmailMaxFailures int64 = 5

const defaultLoginIPMaxFailures int64 = 20

const defaultLoginFailureWindow = 10 * time.Minute

var recordLoginFailureScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`)

type RedisLoginLimiter struct {
	client      *redis.Client
	maxFailures int64
	window      time.Duration
}

func NewRedisLoginLimiter(client *redis.Client) *RedisLoginLimiter {
	return NewRedisLoginLimiterWithOptions(client, defaultLoginEmailMaxFailures, defaultLoginFailureWindow)
}

func NewRedisLoginIPLimiter(client *redis.Client) *RedisLoginLimiter {
	return NewRedisLoginLimiterWithOptions(client, defaultLoginIPMaxFailures, defaultLoginFailureWindow)
}

func NewRedisLoginLimiterWithOptions(client *redis.Client, maxFailures int64, window time.Duration) *RedisLoginLimiter {
	return &RedisLoginLimiter{
		client:      client,
		maxFailures: maxFailures,
		window:      window,
	}
}

func (l *RedisLoginLimiter) TooManyAttempts(ctx context.Context, key string) (bool, error) {
	count, err := l.client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return count >= l.maxFailures, nil
}

func (l *RedisLoginLimiter) RecordFailure(ctx context.Context, key string) error {
	windowSeconds := strconv.FormatInt(int64(l.window/time.Second), 10)
	return recordLoginFailureScript.Run(ctx, l.client, []string{key}, windowSeconds).Err()
}

func (l *RedisLoginLimiter) Reset(ctx context.Context, key string) error {
	return l.client.Del(ctx, key).Err()
}
