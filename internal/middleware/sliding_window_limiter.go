package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// SlidingWindowRateLimiter implements a sliding window rate limiter
type SlidingWindowRateLimiter struct {
	client     *redis.Client
	logger     *logrus.Logger
	windowSize time.Duration
	limit      int
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(client *redis.Client, limit int, windowSize time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		client:     client,
		logger:     logrus.New(),
		windowSize: windowSize,
		limit:      limit,
	}
}

// IsAllowed checks if the request is allowed under the sliding window
func (rl *SlidingWindowRateLimiter) IsAllowed(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	pipe := rl.client.Pipeline()

	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current requests in window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiry for the key
	pipe.Expire(ctx, key, rl.windowSize+time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.WithError(err).Error("Failed to execute Redis pipeline for rate limiting")
		return false, err
	}

	count := countCmd.Val()
	return count < int64(rl.limit), nil
}

// SlidingWindowMiddleware returns a Gin middleware for sliding window rate limiting
func (rl *SlidingWindowRateLimiter) SlidingWindowMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		userID := c.GetString("user_id")

		// Create a composite key for per-user and per-IP limiting
		key := fmt.Sprintf("rate_limit:%s:%s", clientIP, userID)
		if userID == "" {
			key = fmt.Sprintf("rate_limit:ip:%s", clientIP)
		}

		allowed, err := rl.IsAllowed(c.Request.Context(), key)
		if err != nil {
			rl.logger.WithError(err).Error("Rate limiter error")
			// Fail open - allow request if Redis is down
			c.Next()
			return
		}

		if !allowed {
			rl.logger.WithFields(logrus.Fields{
				"client_ip": clientIP,
				"user_id":   userID,
				"key":       key,
			}).Warn("Rate limit exceeded")

			c.JSON(429, gin.H{
				"error": gin.H{
					"message":     "Rate limit exceeded",
					"code":        "rate_limit_exceeded",
					"retry_after": int(rl.windowSize.Seconds()),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
