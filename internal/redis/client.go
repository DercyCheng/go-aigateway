package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Config Redis配置
type Config struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MaxRetries   int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	TLSConfig    *tls.Config
}

// DefaultConfig returns default Redis configuration
func DefaultConfig() *Config {
	return &Config{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MaxRetries:   3,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	}
}

// Client Redis客户端管理器
type Client struct {
	*redis.Client
	config *Config
}

// NewClient 创建Redis客户端
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MaxRetries:   config.MaxRetries,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolTimeout:  config.PoolTimeout,
		TLSConfig:    config.TLSConfig,
	})

	client := &Client{
		Client: rdb,
		config: config,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logrus.WithField("addr", config.Addr).Info("Redis client connected successfully")
	return client, nil
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

// GetStats returns Redis client statistics
func (c *Client) GetStats() *redis.PoolStats {
	return c.PoolStats()
}

// Close gracefully closes the Redis connection
func (c *Client) Close() error {
	logrus.Info("Closing Redis connection")
	return c.Client.Close()
}

// IsConnected checks if Redis is currently connected
func (c *Client) IsConnected(ctx context.Context) bool {
	return c.HealthCheck(ctx) == nil
}
