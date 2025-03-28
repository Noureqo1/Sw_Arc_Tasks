package limiter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitMiddleware struct {
	redisClient *redis.Client
	algorithm   string
	limit       int
	window      time.Duration
}

func NewRateLimitMiddleware(rdb *redis.Client, algo string, limit int, windowStr string) (gin.HandlerFunc, error) {
	dur, err := time.ParseDuration(windowStr)
	if err != nil {
		return nil, fmt.Errorf("invalid window duration: %w", err)
	}

	mw := &RateLimitMiddleware{
		redisClient: rdb,
		algorithm:   strings.ToLower(algo),
		limit:       limit,
		window:      dur,
	}

	return mw.handle, nil
}

func (rl *RateLimitMiddleware) handle(c *gin.Context) {
	userKey := c.ClientIP()

	var allowed bool
	var err error

	switch rl.algorithm {
	case "token-bucket":
		allowed, err = rl.tokenBucketCheck(c, userKey)
	case "sliding-window":
		allowed, err = rl.slidingWindowCheck(c, userKey)
	default:
		allowed, err = rl.tokenBucketCheck(c, userKey)
	}

	if err != nil {
		c.Next()
		return
	}

	if !allowed {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	c.Next()
}

func (rl *RateLimitMiddleware) tokenBucketCheck(c *gin.Context, userKey string) (bool, error) {
	ctx := context.Background()

	tbKey := fmt.Sprintf("tokenbucket:%s", userKey)
	windowKey := fmt.Sprintf("%s:window", tbKey)

	pipe := rl.redisClient.TxPipeline()
	valCmd := pipe.Get(ctx, windowKey)
	ttlCmd := pipe.TTL(ctx, windowKey)
	_, err := pipe.Exec(ctx)

	if err != nil && err != redis.Nil {
		return false, err
	}

	count := 0
	countStr, err2 := valCmd.Result()
	if err2 == nil {
		fmt.Sscanf(countStr, "%d", &count)
	}

	ttlVal, _ := ttlCmd.Result()
	if ttlVal <= 0 {
		count = 0
	}

	count++
	if count > rl.limit {
		return false, nil
	}

	pipe2 := rl.redisClient.TxPipeline()
	pipe2.Set(ctx, windowKey, fmt.Sprintf("%d", count), rl.window)
	_, errSet := pipe2.Exec(ctx)
	if errSet != nil {
		return false, errSet
	}
	return true, nil
}

func (rl *RateLimitMiddleware) slidingWindowCheck(c *gin.Context, userKey string) (bool, error) {
	ctx := context.Background()
	swKey := fmt.Sprintf("sliding:%s", userKey)
	now := time.Now().Unix()

	cutoff := now - int64(rl.window.Seconds())

	if err := rl.redisClient.ZRemRangeByScore(ctx, swKey, "0", fmt.Sprintf("%d", cutoff)).Err(); err != nil {
		return false, err
	}

	count, err := rl.redisClient.ZCard(ctx, swKey).Result()
	if err != nil {
		return false, err
	}

	if count > int64(rl.limit) {
		return false, nil
	}

	if err := rl.redisClient.ZAdd(ctx, swKey, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d-%d", now, time.Now().UnixNano()),
	}).Err(); err != nil {
		return false, err
	}

	rl.redisClient.Expire(ctx, swKey, rl.window)

	return true, nil
}
