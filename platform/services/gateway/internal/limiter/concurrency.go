package limiter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ConcurrencyMiddleware struct {
	redisClient   *redis.Client
	maxConcurrent int
	expire        time.Duration
}

func NewConcurrencyMiddleware(rdb *redis.Client, maxConcurrent int, expire time.Duration) gin.HandlerFunc {
	cm := &ConcurrencyMiddleware{
		redisClient:   rdb,
		maxConcurrent: maxConcurrent,
		expire:        expire,
	}
	return cm.handle
}

func (c *ConcurrencyMiddleware) handle(ctx *gin.Context) {
	userID := ctx.ClientIP()

	allowed, key, err := c.increment(userID)

	if err != nil {
		ctx.Next()
		return
	}

	if !allowed {
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many concurrent requests"})
		return
	}

	defer func() {
		if err := c.decrement(userID, key); err != nil {
		}
	}()

	ctx.Next()
}

func (c *ConcurrencyMiddleware) increment(userID string) (bool, string, error) {
	rdb := c.redisClient
	ctx := context.Background()

	key := fmt.Sprintf("concurrency:%s", userID)

	val, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, key, err
	}

	if val == 1 {
		rdb.Expire(ctx, key, c.expire)
	}

	if val > int64(c.maxConcurrent) {
		rdb.Decr(ctx, key)
		return false, key, nil
	}

	return true, key, nil
}

func (c *ConcurrencyMiddleware) decrement(userID, key string) error {
	ctx := context.Background()

	val, err := c.redisClient.Decr(ctx, key).Result()
	if err != nil {
		return err
	}

	if val <= 0 {
		c.redisClient.Del(ctx, key)
	}
	return nil
}
