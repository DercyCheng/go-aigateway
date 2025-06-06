package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Config Redis配置
type Config struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

// Client Redis客户端管理器
type Client struct {
	*redis.Client
	config *Config
}

// NewClient 创建Redis客户端
func NewClient(config *Config) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	return &Client{
		Client: rdb,
		config: config,
	}
}

// HealthCheck Redis健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.Ping(ctx).Result()
	if err != nil {
		logrus.WithError(err).Error("Redis health check failed")
		return err
	}
	return nil
}

// StartHealthCheck 启动健康检查
func (c *Client) StartHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.HealthCheck(ctx); err != nil {
				logrus.WithError(err).Error("Redis health check failed")
			}
		}
	}
}

// Close 关闭Redis连接
func (c *Client) Close() error {
	return c.Client.Close()
}
