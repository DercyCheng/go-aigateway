// Package context provides enhanced context utilities
package context

import (
	"context"
	"time"
)

// TimeoutConfig defines timeout configurations for different operations
type TimeoutConfig struct {
	Default    time.Duration `yaml:"default" json:"default"`
	Database   time.Duration `yaml:"database" json:"database"`
	Network    time.Duration `yaml:"network" json:"network"`
	Processing time.Duration `yaml:"processing" json:"processing"`
}

// DefaultTimeouts provides sensible default timeout values
var DefaultTimeouts = TimeoutConfig{
	Default:    30 * time.Second,
	Database:   10 * time.Second,
	Network:    15 * time.Second,
	Processing: 60 * time.Second,
}

// WithTimeout creates a context with the specified timeout
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithDatabaseTimeout creates a context with database timeout
func WithDatabaseTimeout(parent context.Context, config *TimeoutConfig) (context.Context, context.CancelFunc) {
	timeout := DefaultTimeouts.Database
	if config != nil && config.Database > 0 {
		timeout = config.Database
	}
	return context.WithTimeout(parent, timeout)
}

// WithNetworkTimeout creates a context with network timeout
func WithNetworkTimeout(parent context.Context, config *TimeoutConfig) (context.Context, context.CancelFunc) {
	timeout := DefaultTimeouts.Network
	if config != nil && config.Network > 0 {
		timeout = config.Network
	}
	return context.WithTimeout(parent, timeout)
}

// WithProcessingTimeout creates a context with processing timeout
func WithProcessingTimeout(parent context.Context, config *TimeoutConfig) (context.Context, context.CancelFunc) {
	timeout := DefaultTimeouts.Processing
	if config != nil && config.Processing > 0 {
		timeout = config.Processing
	}
	return context.WithTimeout(parent, timeout)
}

// SafeCancel safely calls cancel function if not nil
func SafeCancel(cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
}
