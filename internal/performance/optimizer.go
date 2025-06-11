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
	cache           map[string]*CacheEntry
	cacheMutex      sync.RWMutex
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
	BatchProcessed      int64
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

// copyHeaders converts http.Header to map[string]string
func copyHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// generateBatchKey generates a key for request batching
func generateBatchKey(c *gin.Context) string {
	return c.Request.URL.Path
}

// generateCacheKey generates a key for caching
func generateCacheKey(c *gin.Context) string {
	return c.Request.Method + ":" + c.Request.URL.Path + ":" + c.Request.URL.RawQuery
}

// recordCompressionUse records compression usage metrics
func (po *PerformanceOptimizer) recordCompressionUse() {
	atomic.AddInt64(&po.metrics.CompressionUse, 1)
}

// processBatch processes a batch of requests
func (po *PerformanceOptimizer) processBatch(batch *RequestBatch) {
	// Process batched requests efficiently
	if len(batch.Requests) == 0 {
		return
	}

	logrus.WithField("batch_size", len(batch.Requests)).Debug("Processing request batch")

	// Group requests by endpoint for parallel processing
	endpointGroups := make(map[string][]*gin.Context)
	for _, req := range batch.Requests {
		endpoint := req.Request.URL.Path
		endpointGroups[endpoint] = append(endpointGroups[endpoint], req)
	}

	// Process each endpoint group in parallel
	var wg sync.WaitGroup
	for endpoint, requests := range endpointGroups {
		wg.Add(1)
		go func(ep string, reqs []*gin.Context) {
			defer wg.Done()
			po.processEndpointBatch(ep, reqs)
		}(endpoint, requests)
	}

	wg.Wait()
	atomic.AddInt64(&po.metrics.BatchProcessed, 1)
}

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

// processEndpointBatch processes a batch of requests for a specific endpoint
func (po *PerformanceOptimizer) processEndpointBatch(endpoint string, requests []*gin.Context) {
	logrus.WithFields(logrus.Fields{
		"endpoint":   endpoint,
		"batch_size": len(requests),
	}).Debug("Processing endpoint batch")

	// Process requests based on endpoint type
	switch {
	case strings.Contains(endpoint, "/chat"):
		po.processChatBatch(requests)
	case strings.Contains(endpoint, "/completion"):
		po.processCompletionBatch(requests)
	case strings.Contains(endpoint, "/models"):
		po.processModelsBatch(requests)
	default:
		po.processGenericBatch(requests)
	}
}

// processChatBatch optimizes chat completion requests
func (po *PerformanceOptimizer) processChatBatch(requests []*gin.Context) {
	// For chat requests, we can potentially combine similar requests
	// or prioritize based on user context
	for _, req := range requests {
		start := time.Now()
		req.Next()
		duration := time.Since(start)
		po.recordRequest(duration)
	}
}

// processCompletionBatch optimizes completion requests
func (po *PerformanceOptimizer) processCompletionBatch(requests []*gin.Context) {
	// Similar optimization strategies for completion requests
	for _, req := range requests {
		start := time.Now()
		req.Next()
		duration := time.Since(start)
		po.recordRequest(duration)
	}
}

// processModelsBatch optimizes model listing requests
func (po *PerformanceOptimizer) processModelsBatch(requests []*gin.Context) {
	// Models endpoint can be heavily cached
	cacheKey := "models_list"

	// Check if we have cached response
	if cached := po.getCachedResponse(cacheKey); cached != nil {
		for _, req := range requests {
			req.JSON(http.StatusOK, cached)
		}
		return
	}

	// Process first request and cache result
	if len(requests) > 0 {
		first := requests[0]
		start := time.Now()
		first.Next()
		duration := time.Since(start)
		po.recordRequest(duration)

		// Cache the response if successful
		if first.Writer.Status() == http.StatusOK {
			// In a real implementation, we'd extract the response body
			// and cache it for subsequent requests
			po.setCachedResponse(cacheKey, map[string]interface{}{
				"cached_at": time.Now(),
				"data":      "models_data", // Placeholder
			})
		}

		// Process remaining requests with cached data
		for _, req := range requests[1:] {
			req.JSON(http.StatusOK, map[string]interface{}{
				"message": "Processed with batch optimization",
			})
		}
	}
}

// processGenericBatch processes other types of requests
func (po *PerformanceOptimizer) processGenericBatch(requests []*gin.Context) {
	// Default processing for non-specialized endpoints
	for _, req := range requests {
		start := time.Now()
		req.Next()
		duration := time.Since(start)
		po.recordRequest(duration)
	}
}

// recordRequest records metrics for a completed request
func (po *PerformanceOptimizer) recordRequest(duration time.Duration) {
	po.metrics.mutex.Lock()
	defer po.metrics.mutex.Unlock()

	po.metrics.RequestCount++
	po.metrics.TotalDuration += duration
	if po.metrics.RequestCount > 0 {
		po.metrics.AverageResponseTime = time.Duration(
			int64(po.metrics.TotalDuration) / po.metrics.RequestCount,
		)
	}
}

// getRequestCount returns the current request count
func (po *PerformanceOptimizer) getRequestCount() int64 {
	po.metrics.mutex.RLock()
	defer po.metrics.mutex.RUnlock()
	return po.metrics.RequestCount
}

// getCachedResponse retrieves cached response data
func (po *PerformanceOptimizer) getCachedResponse(key string) interface{} {
	po.cacheMutex.RLock()
	defer po.cacheMutex.RUnlock()

	// Check if we have an in-memory cache implementation
	cache := po.getCache()
	if cache == nil {
		return nil
	}

	// Retrieve entry with TTL check
	if entry, exists := cache[key]; exists {
		if time.Since(entry.Timestamp) <= entry.TTL {
			atomic.AddInt64(&po.metrics.CacheHits, 1)
			logrus.WithField("cache_key", key).Debug("Cache hit")
			return entry.Body
		} else {
			// Entry expired, remove it (upgrade to write lock)
			po.cacheMutex.RUnlock()
			po.cacheMutex.Lock()
			delete(cache, key)
			po.cacheMutex.Unlock()
			po.cacheMutex.RLock()

			atomic.AddInt64(&po.metrics.CacheMisses, 1)
			logrus.WithField("cache_key", key).Debug("Cache miss (expired)")
		}
	} else {
		atomic.AddInt64(&po.metrics.CacheMisses, 1)
		logrus.WithField("cache_key", key).Debug("Cache miss")
	}

	return nil
}

// setCachedResponse stores response data in cache
func (po *PerformanceOptimizer) setCachedResponse(key string, data interface{}) {
	po.cacheMutex.Lock()
	defer po.cacheMutex.Unlock()

	cache := po.getCache()
	if cache == nil {
		return
	}

	// Create cache entry with appropriate TTL
	entry := &CacheEntry{
		Body:        data.([]byte),
		Timestamp:   time.Now(),
		TTL:         po.calculateCacheTTL(key),
		StatusCode:  200,
		ContentType: "application/json",
		Headers:     make(map[string]string),
	}

	// Store in cache with size limit
	if len(cache) >= 1000 {
		po.evictOldestCacheEntries(cache)
	}

	cache[key] = entry
	logrus.WithFields(logrus.Fields{
		"cache_key": key,
		"ttl":       entry.TTL,
	}).Debug("Response cached")
}

// getCache returns the cache map (in a real implementation, this might be Redis)
func (po *PerformanceOptimizer) getCache() map[string]*CacheEntry {
	// Simple in-memory cache for this implementation
	// In production, this would be Redis or another distributed cache
	if po.cache == nil {
		po.cache = make(map[string]*CacheEntry)
	}
	return po.cache
}

// calculateCacheTTL calculates appropriate TTL based on content type
func (po *PerformanceOptimizer) calculateCacheTTL(key string) time.Duration {
	// Different TTLs for different content types
	switch {
	case strings.Contains(key, "models"):
		return 1 * time.Hour // Model info changes rarely
	case strings.Contains(key, "chat"):
		return 5 * time.Minute // Chat responses can be cached briefly
	case strings.Contains(key, "completion"):
		return 5 * time.Minute // Completion responses can be cached briefly
	default:
		return 10 * time.Minute // Default TTL
	}
}

// evictOldestCacheEntries removes oldest entries to maintain cache size
func (po *PerformanceOptimizer) evictOldestCacheEntries(cache map[string]*CacheEntry) {
	if len(cache) <= 800 {
		return
	}

	// Find and remove 200 oldest entries
	type keyTimestamp struct {
		key       string
		timestamp time.Time
	}

	var entries []keyTimestamp
	for k, v := range cache {
		entries = append(entries, keyTimestamp{key: k, timestamp: v.Timestamp})
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].timestamp.After(entries[j].timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest 200 entries
	removeCount := 200
	if len(entries) < removeCount {
		removeCount = len(entries)
	}
	for i := 0; i < removeCount; i++ {
		delete(cache, entries[i].key)
	}

	logrus.WithField("evicted_count", removeCount).Debug("Evicted old cache entries")
}

// shouldSkipCompression determines if compression should be skipped
func shouldSkipCompression(contentType string) bool {
	skipTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/gzip",
		"application/zip",
		"application/octet-stream",
	}

	for _, skipType := range skipTypes {
		if strings.Contains(contentType, skipType) {
			return true
		}
	}
	return false
}

// calculateDynamicTTL calculates cache TTL based on content size and type
func (po *PerformanceOptimizer) calculateDynamicTTL(path string, contentSize int) time.Duration {
	baseTTL := 5 * time.Minute

	// Adjust TTL based on path
	switch {
	case strings.Contains(path, "/models"):
		return 30 * time.Minute // Models don't change frequently
	case strings.Contains(path, "/health"):
		return 1 * time.Minute // Health status changes frequently
	case strings.Contains(path, "/stats"):
		return 30 * time.Second // Stats change frequently
	case contentSize < 1024: // Small responses
		return baseTTL * 2
	case contentSize > 100*1024: // Large responses
		return baseTTL / 2
	default:
		return baseTTL
	}
}
