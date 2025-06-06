package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RedisRateLimiter Redis全局限流器
type RedisRateLimiter struct {
	client      *redis.Client
	globalLimit int           // 全局QPS限制
	userLimit   int           // 单用户QPS限制
	windowSize  time.Duration // 时间窗口大小
	keyPrefix   string        // Redis key前缀
}

// NewRedisRateLimiter 创建Redis限流器
func NewRedisRateLimiter(redisClient *redis.Client, globalLimit, userLimit int, windowSize time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:      redisClient,
		globalLimit: globalLimit,
		userLimit:   userLimit,
		windowSize:  windowSize,
		keyPrefix:   "rate_limit:",
	}
}

// RedisRateLimit Redis全局限流中间件
func RedisRateLimit(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		clientIP := c.ClientIP()
		userKey := c.GetHeader("Authorization") // 使用API Key作为用户标识
		if userKey == "" {
			userKey = clientIP
		}

		// 检查全局限流
		globalAllowed, globalRemaining, err := limiter.checkLimit(ctx, "global", limiter.globalLimit)
		if err != nil {
			logrus.WithError(err).Error("Redis rate limit check failed")
			// 如果Redis出错，降级到内存限流
			c.Next()
			return
		}

		if !globalAllowed {
			RecordRateLimitHit("global")
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.globalLimit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(globalRemaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limiter.windowSize).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "Global rate limit exceeded",
					"type":    "rate_limit_error",
					"code":    "global_rate_limit_exceeded",
					"details": map[string]interface{}{
						"limit":     limiter.globalLimit,
						"remaining": globalRemaining,
						"reset_at":  time.Now().Add(limiter.windowSize).Unix(),
					},
				},
			})
			c.Abort()
			return
		}

		// 检查用户限流
		userAllowed, userRemaining, err := limiter.checkLimit(ctx, fmt.Sprintf("user:%s", userKey), limiter.userLimit)
		if err != nil {
			logrus.WithError(err).Error("Redis user rate limit check failed")
			c.Next()
			return
		}

		if !userAllowed {
			RecordRateLimitHit(clientIP)
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.userLimit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(userRemaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limiter.windowSize).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "User rate limit exceeded",
					"type":    "rate_limit_error",
					"code":    "user_rate_limit_exceeded",
					"details": map[string]interface{}{
						"limit":     limiter.userLimit,
						"remaining": userRemaining,
						"reset_at":  time.Now().Add(limiter.windowSize).Unix(),
					},
				},
			})
			c.Abort()
			return
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.userLimit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(userRemaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limiter.windowSize).Unix(), 10))

		c.Next()
	}
}

// checkLimit 检查限流，使用滑动窗口算法
func (r *RedisRateLimiter) checkLimit(ctx context.Context, key string, limit int) (bool, int, error) {
	now := time.Now()
	windowStart := now.Add(-r.windowSize)
	redisKey := r.keyPrefix + key

	pipe := r.client.TxPipeline()

	// 移除过期的记录
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.UnixNano(), 10))

	// 计算当前窗口内的请求数
	countCmd := pipe.ZCard(ctx, redisKey)
	// 添加当前请求
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// 设置过期时间
	pipe.Expire(ctx, redisKey, r.windowSize*2)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	currentCount := int(countCmd.Val())
	remaining := limit - currentCount - 1
	if remaining < 0 {
		remaining = 0
	}

	allowed := currentCount < limit
	return allowed, remaining, nil
}

// GetRateLimitStats 获取限流统计信息
func (r *RedisRateLimiter) GetRateLimitStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取全局统计
	globalKey := r.keyPrefix + "global"
	globalCount, err := r.client.ZCard(ctx, globalKey).Result()
	if err != nil {
		return nil, err
	}

	stats["global_current_requests"] = globalCount
	stats["global_limit"] = r.globalLimit
	stats["global_remaining"] = r.globalLimit - int(globalCount)

	// 获取活跃用户数
	pattern := r.keyPrefix + "user:*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	stats["active_users"] = len(keys)
	stats["window_size_seconds"] = r.windowSize.Seconds()

	return stats, nil
}
