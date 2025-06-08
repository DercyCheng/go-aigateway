package performance

import (
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PerformanceOptimizer provides performance enhancements
type PerformanceOptimizer struct {
	logger    *logrus.Logger
	cachePool sync.Pool
	gzipPool  sync.Pool
	metrics   *PerformanceMetrics
}

// PerformanceMetrics tracks performance data
type PerformanceMetrics struct {
	RequestCount   int64
	TotalDuration  time.Duration
	CacheHits      int64
	CacheMisses    int64
	CompressionUse int64
	mutex          sync.RWMutex
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer() *PerformanceOptimizer {
	return &PerformanceOptimizer{
		logger:  logrus.New(),
		metrics: &PerformanceMetrics{},
		cachePool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{})
			},
		},
		gzipPool: sync.Pool{
			New: func() interface{} {
				return gzip.NewWriter(nil)
			},
		},
	}
}

// ResponseCachingMiddleware implements intelligent response caching
func (po *PerformanceOptimizer) ResponseCachingMiddleware(cacheTTL time.Duration) gin.HandlerFunc {
	cache := make(map[string]CacheEntry)
	var mu sync.RWMutex

	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		cacheKey := generateCacheKey(c)

		// Check cache
		mu.RLock()
		entry, exists := cache[cacheKey]
		mu.RUnlock()

		if exists && time.Since(entry.Timestamp) < cacheTTL {
			// Cache hit
			po.recordCacheHit()
			c.Header("X-Cache", "HIT")
			c.Header("X-Cache-TTL", strconv.Itoa(int(cacheTTL.Seconds())))

			for key, value := range entry.Headers {
				c.Header(key, value)
			}

			c.Data(entry.StatusCode, entry.ContentType, entry.Body)
			return
		}

		// Cache miss - record response
		po.recordCacheMiss()
		writer := &CacheResponseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = writer

		c.Next()

		// Store in cache if response is cacheable
		if writer.Status() == http.StatusOK && len(writer.body) > 0 {
			entry := CacheEntry{
				StatusCode:  writer.Status(),
				ContentType: writer.Header().Get("Content-Type"),
				Headers:     copyHeaders(writer.Header()),
				Body:        writer.body,
				Timestamp:   time.Now(),
			}

			mu.Lock()
			cache[cacheKey] = entry
			mu.Unlock()

			c.Header("X-Cache", "MISS")
		}
	}
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

// CacheEntry represents a cached response
type CacheEntry struct {
	StatusCode  int
	ContentType string
	Headers     map[string]string
	Body        []byte
	Timestamp   time.Time
}

// CacheResponseWriter captures response for caching
type CacheResponseWriter struct {
	gin.ResponseWriter
	body []byte
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
	return *po.metrics
}
