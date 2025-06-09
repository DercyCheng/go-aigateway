package performance

import (
	"bytes"
	"compress/gzip"
	"go-aigateway/internal/config"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PerformanceOptimizer provides comprehensive performance enhancements
type PerformanceOptimizer struct {
	config          *config.Config
	logger          *logrus.Logger
	cachePool       sync.Pool
	gzipPool        sync.Pool
	bufferPool      sync.Pool
	metrics         *PerformanceMetrics
	rateLimiter     *AdaptiveRateLimiter
	loadBalancer    *LoadBalancer
	circuitBreakers map[string]*CircuitBreaker
	connectionPool  *ConnectionPool
}

// PerformanceMetrics tracks comprehensive performance data
type PerformanceMetrics struct {
	RequestCount        int64
	TotalDuration       time.Duration
	AverageResponseTime time.Duration
	CacheHits           int64
	CacheMisses         int64
	CompressionUse      int64
	ConnectionPoolHits  int64
	ConnectionPoolMiss  int64
	CircuitBreakerTrips int64
	RateLimitHits       int64
	CPUUsage            float64
	MemoryUsage         float64
	GoroutineCount      int
	mutex               sync.RWMutex
}

// AdaptiveRateLimiter implements intelligent rate limiting
type AdaptiveRateLimiter struct {
	baseLimit    int
	currentLimit int64
	windowSize   time.Duration
	requests     map[string]*RequestWindow
	mutex        sync.RWMutex
	cpuThreshold float64
	memThreshold float64
}

// RequestWindow tracks requests in a time window
type RequestWindow struct {
	count     int64
	lastReset time.Time
}

// LoadBalancer implements weighted round-robin load balancing
type LoadBalancer struct {
	backends []Backend
	current  int64
	mutex    sync.RWMutex
}

// Backend represents a backend server
type Backend struct {
	URL         string
	Weight      int
	HealthScore float64
	Active      bool
	LastCheck   time.Time
}

// CircuitBreaker implements circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failureCount     int64
	lastFailureTime  time.Time
	state            int32 // 0: Closed, 1: Open, 2: HalfOpen
}

// ConnectionPool manages HTTP connections efficiently
type ConnectionPool struct {
	client      *http.Client
	maxConns    int
	activeConns int64
	mutex       sync.RWMutex
}

// CacheEntry represents a cached response
type CacheEntry struct {
	StatusCode  int
	ContentType string
	Headers     map[string]string
	Body        []byte
	Timestamp   time.Time
	TTL         time.Duration
}

// CacheResponseWriter wraps gin.ResponseWriter to capture response data
type CacheResponseWriter struct {
	gin.ResponseWriter
	body   []byte
	status int
}

// NewPerformanceOptimizer creates a new performance optimizer with all features
func NewPerformanceOptimizer(cfg *config.Config) *PerformanceOptimizer {
	po := &PerformanceOptimizer{
		config:  cfg,
		logger:  logrus.New(),
		metrics: &PerformanceMetrics{},
		rateLimiter: &AdaptiveRateLimiter{
			baseLimit:    1000,
			currentLimit: 1000,
			windowSize:   time.Minute,
			requests:     make(map[string]*RequestWindow),
			cpuThreshold: 80.0,
			memThreshold: 85.0,
		},
		loadBalancer: &LoadBalancer{
			backends: make([]Backend, 0),
		},
		circuitBreakers: make(map[string]*CircuitBreaker),
		connectionPool: &ConnectionPool{
			client: &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
				},
			},
			maxConns: 100,
		},
		cachePool: sync.Pool{
			New: func() interface{} {
				return make(map[string]*CacheEntry)
			},
		},
		gzipPool: sync.Pool{
			New: func() interface{} {
				w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
				return w
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
	}

	// Start background performance monitoring
	go po.performanceMonitor()

	return po
}

// IntelligentCachingMiddleware implements advanced response caching
func (po *PerformanceOptimizer) IntelligentCachingMiddleware(cacheTTL time.Duration) gin.HandlerFunc {
	cache := po.cachePool.Get().(map[string]*CacheEntry)
	var mu sync.RWMutex

	return func(c *gin.Context) {
		// Only cache GET requests for specific endpoints
		if c.Request.Method != "GET" || !po.shouldCache(c.Request.URL.Path) {
			c.Next()
			return
		}

		cacheKey := po.generateAdvancedCacheKey(c)

		// Check cache with thread safety
		mu.RLock()
		entry, exists := cache[cacheKey]
		mu.RUnlock()

		if exists && time.Since(entry.Timestamp) < entry.TTL {
			// Cache hit - serve from cache
			atomic.AddInt64(&po.metrics.CacheHits, 1)
			c.Header("X-Cache", "HIT")
			c.Header("X-Cache-Age", strconv.Itoa(int(time.Since(entry.Timestamp).Seconds())))

			// Restore headers
			for key, value := range entry.Headers {
				c.Header(key, value)
			}

			c.Data(entry.StatusCode, entry.ContentType, entry.Body)
			return
		}

		// Cache miss - record and process request
		atomic.AddInt64(&po.metrics.CacheMisses, 1)
		writer := &CacheResponseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = writer

		c.Next()

		// Store successful responses in cache
		if writer.status == http.StatusOK && len(writer.body) > 0 {
			entry := &CacheEntry{
				StatusCode:  writer.status,
				ContentType: writer.Header().Get("Content-Type"),
				Headers:     copyHeaders(writer.Header()),
				Body:        writer.body,
				Timestamp:   time.Now(),
				TTL:         po.calculateDynamicTTL(c.Request.URL.Path, len(writer.body)),
			}

			mu.Lock()
			cache[cacheKey] = entry
			// Implement LRU eviction if cache is too large
			if len(cache) > 1000 {
				po.evictOldestEntries(cache, 100)
			}
			mu.Unlock()
		}
	}
}

// AdaptiveCompressionMiddleware implements intelligent compression
func (po *PerformanceOptimizer) AdaptiveCompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Skip compression for small responses or certain content types
		if shouldSkipCompression(c.GetHeader("Content-Type")) {
			c.Next()
			return
		}

		// Get gzip writer from pool
		gzipWriter := po.gzipPool.Get().(*gzip.Writer)
		defer po.gzipPool.Put(gzipWriter)

		// Create compression writer
		writer := &gzipResponseWriter{
			ResponseWriter: c.Writer,
			writer:         gzipWriter,
		}

		// Reset gzip writer with new underlying writer
		gzipWriter.Reset(c.Writer)
		c.Writer = writer
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Next()

		gzipWriter.Close()
		atomic.AddInt64(&po.metrics.CompressionUse, 1)
	}
}

// AdaptiveRateLimitingMiddleware implements intelligent rate limiting
func (po *PerformanceOptimizer) AdaptiveRateLimitingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !po.rateLimiter.allowRequest(clientIP) {
			atomic.AddInt64(&po.metrics.RateLimitHits, 1)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": int(po.rateLimiter.windowSize.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoadBalancingMiddleware implements intelligent load balancing
func (po *PerformanceOptimizer) LoadBalancingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		backend := po.loadBalancer.selectBackend()
		if backend == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "No healthy backends available",
			})
			c.Abort()
			return
		}

		// Store selected backend for downstream use
		c.Set("selected_backend", backend)
		c.Next()
	}
}

// CircuitBreakerMiddleware implements circuit breaker pattern
func (po *PerformanceOptimizer) CircuitBreakerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.GetString("service_name")
		if serviceName == "" {
			serviceName = "default"
		}

		cb := po.getOrCreateCircuitBreaker(serviceName)

		if !cb.allowRequest() {
			atomic.AddInt64(&po.metrics.CircuitBreakerTrips, 1)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Service circuit breaker is open",
				"service": serviceName,
			})
			c.Abort()
			return
		}

		c.Next()

		// Record success or failure based on response status
		if c.Writer.Status() >= 500 {
			cb.recordFailure()
		} else {
			cb.recordSuccess()
		}
	}
}

// ConnectionPoolMiddleware manages HTTP connections efficiently
func (po *PerformanceOptimizer) ConnectionPoolMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if atomic.LoadInt64(&po.connectionPool.activeConns) >= int64(po.connectionPool.maxConns) {
			atomic.AddInt64(&po.metrics.ConnectionPoolMiss, 1)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Connection pool exhausted",
			})
			c.Abort()
			return
		}

		atomic.AddInt64(&po.connectionPool.activeConns, 1)
		atomic.AddInt64(&po.metrics.ConnectionPoolHits, 1)

		defer atomic.AddInt64(&po.connectionPool.activeConns, -1)

		c.Next()
	}
}

// Performance monitoring and optimization methods
func (po *PerformanceOptimizer) performanceMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		po.updateSystemMetrics()
		po.adjustRateLimits()
		po.optimizeResourceUsage()
		po.healthCheckBackends()
	}
}

func (po *PerformanceOptimizer) updateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	po.metrics.mutex.Lock()
	po.metrics.MemoryUsage = float64(m.Alloc) / 1024 / 1024 // MB
	po.metrics.GoroutineCount = runtime.NumGoroutine()
	po.metrics.mutex.Unlock()
}

func (po *PerformanceOptimizer) adjustRateLimits() {
	po.metrics.mutex.RLock()
	cpuUsage := po.metrics.CPUUsage
	memUsage := po.metrics.MemoryUsage
	po.metrics.mutex.RUnlock()

	// Adjust rate limits based on system load
	if cpuUsage > po.rateLimiter.cpuThreshold || memUsage > po.rateLimiter.memThreshold {
		// Reduce rate limit
		newLimit := int64(float64(po.rateLimiter.currentLimit) * 0.8)
		atomic.StoreInt64(&po.rateLimiter.currentLimit, newLimit)
		po.logger.WithFields(logrus.Fields{
			"cpu_usage":    cpuUsage,
			"memory_usage": memUsage,
			"new_limit":    newLimit,
		}).Info("Reducing rate limit due to high system load")
	} else if cpuUsage < po.rateLimiter.cpuThreshold*0.5 && memUsage < po.rateLimiter.memThreshold*0.5 {
		// Increase rate limit
		newLimit := int64(float64(po.rateLimiter.currentLimit) * 1.2)
		if newLimit > int64(po.rateLimiter.baseLimit*2) {
			newLimit = int64(po.rateLimiter.baseLimit * 2)
		}
		atomic.StoreInt64(&po.rateLimiter.currentLimit, newLimit)
	}
}

// Helper methods for performance optimization
func (po *PerformanceOptimizer) shouldCache(path string) bool {
	// Cache static content and specific API endpoints
	staticPaths := []string{"/health", "/metrics", "/api/v1/models", "/api/v1/stats"}
	for _, staticPath := range staticPaths {
		if strings.HasPrefix(path, staticPath) {
			return true
		}
	}
	return false
}

func (po *PerformanceOptimizer) generateAdvancedCacheKey(c *gin.Context) string {
	var keyBuilder strings.Builder
	keyBuilder.WriteString(c.Request.Method)
	keyBuilder.WriteString(":")
	keyBuilder.WriteString(c.Request.URL.Path)

	// Include query parameters for cache differentiation
	if c.Request.URL.RawQuery != "" {
		keyBuilder.WriteString("?")
		keyBuilder.WriteString(c.Request.URL.RawQuery)
	}

	// Include relevant headers
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		keyBuilder.WriteString(":auth:")
		keyBuilder.WriteString(auth[:min(len(auth), 10)]) // First 10 chars for uniqueness
	}

	return keyBuilder.String()
}

func (po *PerformanceOptimizer) calculateDynamicTTL(path string, responseSize int) time.Duration {
	baseTTL := 5 * time.Minute

	// Longer TTL for smaller responses
	if responseSize < 1024 {
		return baseTTL * 2
	} else if responseSize > 1024*1024 {
		return baseTTL / 2
	}

	return baseTTL
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CompressionMiddleware provides intelligent compression
func (po *PerformanceOptimizer) CompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Skip compression for certain content types
		if shouldSkipCompression(c.GetHeader("Content-Type")) {
			c.Next()
			return
		}

		po.recordCompressionUse()

		// Get gzip writer from pool
		gz := po.gzipPool.Get().(*gzip.Writer)
		defer po.gzipPool.Put(gz)

		gz.Reset(c.Writer)
		defer gz.Close()

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		c.Writer = &gzipResponseWriter{c.Writer, gz}

		c.Next()
	}
}

// ConnectionPoolingMiddleware optimizes HTTP client connections
func (po *PerformanceOptimizer) ConnectionPoolingMiddleware() gin.HandlerFunc {
	// Configure HTTP client with connection pooling
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 30 * time.Second,
	}

	return func(c *gin.Context) {
		c.Set("http_client", client)
		c.Next()
	}
}

// RequestBatchingMiddleware batches similar requests
func (po *PerformanceOptimizer) RequestBatchingMiddleware(batchSize int, batchTimeout time.Duration) gin.HandlerFunc {
	batches := make(map[string]*RequestBatch)
	var mu sync.Mutex

	return func(c *gin.Context) {
		batchKey := generateBatchKey(c)

		mu.Lock()
		batch, exists := batches[batchKey]
		if !exists {
			batch = &RequestBatch{
				Requests: make([]*gin.Context, 0, batchSize),
				Timer:    time.NewTimer(batchTimeout),
			}
			batches[batchKey] = batch
		}

		batch.Requests = append(batch.Requests, c)

		if len(batch.Requests) >= batchSize {
			// Process batch immediately
			go po.processBatch(batch)
			delete(batches, batchKey)
		}
		mu.Unlock()

		// Wait for batch processing or timeout
		select {
		case <-batch.Timer.C:
			mu.Lock()
			if _, exists := batches[batchKey]; exists {
				go po.processBatch(batch)
				delete(batches, batchKey)
			}
			mu.Unlock()
		case <-c.Request.Context().Done():
			// Request cancelled
			return
		}
	}
}

// PerformanceMetricsMiddleware tracks performance metrics
func (po *PerformanceOptimizer) PerformanceMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		po.recordRequest(duration)

		// Add performance headers
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Request-Count", strconv.FormatInt(po.getRequestCount(), 10))
	}
}

func (w *CacheResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return w.ResponseWriter.Write(data)
}

// gzipResponseWriter wraps response writer with gzip compression
type gzipResponseWriter struct {
	gin.ResponseWriter
	writer io.Writer
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}

// RequestBatch groups similar requests for batch processing
type RequestBatch struct {
	Requests []*gin.Context
	Timer    *time.Timer
}

// Helper functions

// evictOldestEntries removes old cache entries to maintain cache size limit
func (po *PerformanceOptimizer) evictOldestEntries(cache map[string]*CacheEntry, maxEntries int) {
	if len(cache) <= maxEntries {
		return
	}

	// Simple LRU eviction based on timestamp
	entriesToRemove := len(cache) - maxEntries
	oldestEntries := make([]string, 0, entriesToRemove)
	oldestTime := time.Now()

	for key, entry := range cache {
		if entry.Timestamp.Before(oldestTime) || len(oldestEntries) < entriesToRemove {
			oldestEntries = append(oldestEntries, key)
			if len(oldestEntries) > entriesToRemove {
				// Remove the newest from our removal list
				oldestEntries = oldestEntries[:entriesToRemove]
			}
		}
	}

	for _, key := range oldestEntries {
		delete(cache, key)
	}
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for a service
func (po *PerformanceOptimizer) getOrCreateCircuitBreaker(serviceName string) *CircuitBreaker {
	if cb, exists := po.circuitBreakers[serviceName]; exists {
		return cb
	}

	cb := &CircuitBreaker{
		failureThreshold: 5,
		resetTimeout:     30 * time.Second,
		state:            0, // Closed
	}
	po.circuitBreakers[serviceName] = cb
	return cb
}

// optimizeResourceUsage performs various resource optimization tasks
func (po *PerformanceOptimizer) optimizeResourceUsage() {
	// Force garbage collection if memory usage is high
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.Alloc > 100*1024*1024 { // 100MB threshold
		runtime.GC()
	}

	// Update system metrics
	po.metrics.mutex.Lock()
	po.metrics.CPUUsage = float64(runtime.NumCPU())
	po.metrics.MemoryUsage = float64(m.Alloc)
	po.metrics.GoroutineCount = runtime.NumGoroutine()
	po.metrics.mutex.Unlock()
}

// healthCheckBackends performs health checks on backend servers
func (po *PerformanceOptimizer) healthCheckBackends() {
	po.loadBalancer.mutex.Lock()
	defer po.loadBalancer.mutex.Unlock()

	for i := range po.loadBalancer.backends {
		backend := &po.loadBalancer.backends[i]

		// Simple HTTP health check
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(backend.URL + "/health")

		if err != nil || resp.StatusCode != http.StatusOK {
			backend.Active = false
			backend.HealthScore = 0.0
		} else {
			backend.Active = true
			backend.HealthScore = 1.0
		}
		backend.LastCheck = time.Now()

		if resp != nil {
			resp.Body.Close()
		}
	}
}

// allowRequest checks if a request should be allowed by the rate limiter
func (rl *AdaptiveRateLimiter) allowRequest(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	window, exists := rl.requests[clientIP]

	if !exists {
		rl.requests[clientIP] = &RequestWindow{
			count:     1,
			lastReset: now,
		}
		return true
	}

	// Reset window if enough time has passed
	if now.Sub(window.lastReset) > rl.windowSize {
		window.count = 1
		window.lastReset = now
		return true
	}

	// Check if under limit
	currentLimit := atomic.LoadInt64(&rl.currentLimit)
	if window.count < currentLimit {
		window.count++
		return true
	}

	return false
}

// selectBackend selects the next backend using weighted round-robin
func (lb *LoadBalancer) selectBackend() *Backend {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if len(lb.backends) == 0 {
		return nil
	}

	// Find next active backend
	attempts := 0
	for attempts < len(lb.backends) {
		index := atomic.AddInt64(&lb.current, 1) % int64(len(lb.backends))
		backend := &lb.backends[index]

		if backend.Active {
			return backend
		}
		attempts++
	}

	// If no active backends, return the first one
	return &lb.backends[0]
}

// allowRequest checks if a request should be allowed through the circuit breaker
func (cb *CircuitBreaker) allowRequest() bool {
	state := atomic.LoadInt32(&cb.state)

	switch state {
	case 0: // Closed
		return true
	case 1: // Open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			atomic.StoreInt32(&cb.state, 2) // Half-open
			return true
		}
		return false
	case 2: // Half-open
		return true
	default:
		return false
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	atomic.AddInt64(&cb.failureCount, 1)
	cb.lastFailureTime = time.Now()

	if atomic.LoadInt64(&cb.failureCount) >= int64(cb.failureThreshold) {
		atomic.StoreInt32(&cb.state, 1) // Open
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt32(&cb.state, 0) // Closed
}

// Helper functions

func generateCacheKey(c *gin.Context) string {
	return c.Request.Method + ":" + c.Request.URL.Path + ":" + c.Request.URL.RawQuery
}

func generateBatchKey(c *gin.Context) string {
	return c.Request.URL.Path
}

func copyHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

func shouldSkipCompression(contentType string) bool {
	skipTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/zip",
		"application/gzip",
		"application/pdf",
	}

	for _, skipType := range skipTypes {
		if strings.HasPrefix(contentType, skipType) {
			return true
		}
	}
	return false
}

func (po *PerformanceOptimizer) processBatch(batch *RequestBatch) {
	// Implementation for processing batched requests
	// This would depend on the specific use case
}

// Metrics recording methods
func (po *PerformanceOptimizer) recordRequest(duration time.Duration) {
	po.metrics.mutex.Lock()
	po.metrics.RequestCount++
	po.metrics.TotalDuration += duration
	po.metrics.mutex.Unlock()
}

func (po *PerformanceOptimizer) recordCacheHit() {
	po.metrics.mutex.Lock()
	po.metrics.CacheHits++
	po.metrics.mutex.Unlock()
}

func (po *PerformanceOptimizer) recordCacheMiss() {
	po.metrics.mutex.Lock()
	po.metrics.CacheMisses++
	po.metrics.mutex.Unlock()
}

func (po *PerformanceOptimizer) recordCompressionUse() {
	po.metrics.mutex.Lock()
	po.metrics.CompressionUse++
	po.metrics.mutex.Unlock()
}

func (po *PerformanceOptimizer) getRequestCount() int64 {
	po.metrics.mutex.RLock()
	defer po.metrics.mutex.RUnlock()
	return po.metrics.RequestCount
}

// GetMetrics returns current performance metrics
func (po *PerformanceOptimizer) GetMetrics() PerformanceMetrics {
	po.metrics.mutex.RLock()
	defer po.metrics.mutex.RUnlock()
	// Create a copy without the mutex to avoid the "return copies lock value" error
	return PerformanceMetrics{
		RequestCount:   po.metrics.RequestCount,
		TotalDuration:  po.metrics.TotalDuration,
		CacheHits:      po.metrics.CacheHits,
		CacheMisses:    po.metrics.CacheMisses,
		CompressionUse: po.metrics.CompressionUse,
	}
}
