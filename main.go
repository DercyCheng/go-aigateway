package main

// AI Gateway - 智能AI网关
//
// 支持本地模型和第三方模型 (阿里百炼/Alibaba DashScope)
// 第三方模型通过阿里云DashScope API提供云端AI服务
// 详见: docs/THIRD_PARTY_MODELS.md

import (
	"context"
	"go-aigateway/internal/autoscaler"
	"go-aigateway/internal/cloud"
	"go-aigateway/internal/config"
	"go-aigateway/internal/discovery"
	"go-aigateway/internal/errors"
	"go-aigateway/internal/handlers"
	"go-aigateway/internal/localmodel"
	"go-aigateway/internal/middleware"
	"go-aigateway/internal/monitoring"
	"go-aigateway/internal/performance"
	"go-aigateway/internal/protocol"
	"go-aigateway/internal/ram"
	redisClient "go-aigateway/internal/redis"
	"go-aigateway/internal/router"
	"go-aigateway/internal/security"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Info("No .env file found, using system environment variables")
	}
	// Initialize configuration
	cfg := config.New()

	// Validate configuration
	if err := cfg.ValidateConfig(); err != nil {
		logrus.WithError(err).Fatal("Configuration validation failed")
	}

	// Setup logging
	setupLogging(cfg)

	// Initialize services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis client
	var redisClientInstance *redisClient.Client
	var err error
	if cfg.Redis.Enabled {
		redisConfig := &redisClient.Config{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
			PoolSize: cfg.Redis.PoolSize,
		}
		redisClientInstance, err = redisClient.NewClient(redisConfig)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to initialize Redis client")
		}

		// Start Redis health check
		go redisClientInstance.StartHealthCheck(ctx)

		logrus.Info("Redis client initialized")
	} else {
		logrus.Info("Redis is disabled")
	}

	// Initialize enhanced error handling system
	errorHandler := errors.NewErrorHandler()
	// Use error handler as middleware (will be added to Gin router later)

	// Initialize performance optimization system
	performanceOptimizer := performance.NewPerformanceOptimizer(cfg)
	// Performance optimizer will be used in middleware (added to Gin router later)

	// Initialize monitoring system with enhanced features
	var monitoringSystem *monitoring.MonitoringSystem
	if cfg.Monitoring.Enabled && redisClientInstance != nil {
		monitoringSystem = monitoring.NewMonitoringSystem(&cfg.Monitoring, redisClientInstance.Client)
		if monitoringSystem != nil {
			logrus.Info("Enhanced monitoring system initialized")
		}
	}

	// Initialize service discovery with real implementations
	serviceDiscovery, err := discovery.NewManager(&cfg.ServiceDiscovery)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize service discovery")
	}
	if serviceDiscovery != nil {
		defer serviceDiscovery.Close()
		logrus.Info("Service discovery initialized")
	}

	// Initialize protocol converter
	protocolConverter := protocol.NewProtocolConverter(&cfg.ProtocolConversion)

	// Initialize authentication systems
	localAuth := security.NewLocalAuthenticator(&cfg.Security)

	// Initialize RAM authentication if enabled
	var ramAuth *ram.RAMAuthenticator
	if cfg.RAMAuth.Enabled {
		ramAuth = ram.NewRAMAuthenticator(&cfg.RAMAuth)
		logrus.Info("RAM authentication initialized")
		// RAM auth will be used in middleware
		_ = ramAuth // Use ramAuth to avoid unused variable warning
	}

	// Initialize cloud integrator with real implementations
	cloudIntegrator, err := cloud.NewCloudIntegrator(&cfg.CloudIntegration)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize cloud integrator")
	} else if cloudIntegrator != nil {
		defer cloudIntegrator.Close()
		logrus.Info("Cloud integration initialized")
	}

	// Initialize local model server and manager if enabled
	var localModelManager *localmodel.Manager
	if cfg.LocalModel.Enabled {
		// Create Python model server
		server := localmodel.NewPythonModelServer(&cfg.LocalModel)
		// Create manager
		localModelManager = localmodel.NewManager(server)

		// Start local model server
		go func() {
			if err := localModelManager.Start(context.Background()); err != nil {
				logrus.WithError(err).Error("Failed to start local model server")
			} else {
				logrus.Info("Local model server started successfully")
			}
		}()
	}

	// Initialize advanced monitoring and scaling components
	var metricsCollector *middleware.AdvancedMetricsCollector
	var autoScaler *autoscaler.AutoScaler
	var redisRateLimiter *middleware.RedisRateLimiter
	var monitoringHandler *handlers.MonitoringHandler

	if redisClientInstance != nil {
		// Initialize advanced metrics collector
		metricsCollector = middleware.NewAdvancedMetricsCollector(redisClientInstance.Client)
		go metricsCollector.StartMetricsCollector(ctx)

		// Initialize auto scaler
		if cfg.AutoScaling.Enabled {
			autoScaler = autoscaler.NewAutoScaler(redisClientInstance.Client, "ai-gateway")
			go autoScaler.Start(ctx)
			logrus.Info("Auto scaler started")
		}

		// Initialize Redis rate limiter
		redisRateLimiter = middleware.NewRedisRateLimiter(
			redisClientInstance.Client,
			cfg.AutoScaling.TargetQPS, // Global limit
			cfg.RateLimit,             // User limit
			time.Minute,               // Window size
		)

		// Initialize monitoring handler
		monitoringHandler = handlers.NewMonitoringHandler(
			redisClientInstance.Client,
			metricsCollector,
			monitoringSystem,
			autoScaler,
			redisRateLimiter,
		)

		logrus.Info("Advanced monitoring and scaling features initialized")
	}

	// Setup Gin mode
	gin.SetMode(cfg.GinMode) // Initialize router
	r := gin.New()

	// Add basic middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Add enhanced error handling middleware
	r.Use(errorHandler.RecoveryMiddleware())

	// Add performance optimization middleware
	r.Use(performanceOptimizer.PerformanceMetricsMiddleware())
	r.Use(performanceOptimizer.IntelligentCachingMiddleware(5 * time.Minute))
	r.Use(performanceOptimizer.AdaptiveCompressionMiddleware())
	r.Use(performanceOptimizer.AdaptiveRateLimitingMiddleware())

	// Add security middleware
	r.Use(middleware.RequestTimeout(30 * time.Second))
	r.Use(middleware.RequestSizeLimit(10 * 1024 * 1024)) // 10MB limit
	r.Use(middleware.CORS(cfg))                          // Pass config to CORS middleware
	r.Use(middleware.PrometheusMetrics())

	// Use Redis rate limiter if available, otherwise use memory-based limiter
	if redisRateLimiter != nil {
		r.Use(middleware.RedisRateLimit(redisRateLimiter))
	} else {
		r.Use(middleware.RateLimiter(cfg.RateLimit))
	}

	// Add advanced metrics middleware if available
	if metricsCollector != nil {
		r.Use(middleware.AdvancedPrometheusMetrics(metricsCollector))
	}

	// Add protocol conversion middleware if enabled
	if cfg.ProtocolConversion.Enabled {
		r.Use(func(c *gin.Context) {
			// Add protocol converter to context for handlers to use
			c.Set("protocol_converter", protocolConverter)
			c.Next()
		})
	}

	// Setup routes
	router.SetupRoutes(r, cfg, localAuth)
	// Setup cloud management routes
	router.SetupCloudRoutes(r, cloudIntegrator)

	// Setup local model routes if enabled
	if cfg.LocalModel.Enabled && localModelManager != nil {
		router.SetupLocalModelRoutes(r, localModelManager, cfg)
		logrus.Info("Local model API routes registered")
	}

	// Setup monitoring routes if available
	if monitoringHandler != nil {
		handlers.RegisterMonitoringRoutes(r, monitoringHandler)
		logrus.Info("Monitoring API routes registered")
	}

	// Setup service management routes
	serviceHandler := handlers.NewServiceHandler()
	handlers.RegisterServiceRoutes(r, serviceHandler)
	logrus.Info("Service management API routes registered")

	// Setup certificate management routes
	certificateHandler := handlers.NewCertificateHandler()
	handlers.RegisterCertificateRoutes(r, certificateHandler)
	logrus.Info("Certificate management API routes registered")

	// Setup domain management routes
	domainHandler := handlers.NewDomainHandler()
	handlers.RegisterDomainRoutes(r, domainHandler)
	logrus.Info("Domain management API routes registered")

	// Start background services
	// Service discovery is automatically started in NewManager

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	logrus.WithField("port", port).Info("Starting AI Gateway server with advanced features")

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server forced to shutdown")
	}

	logrus.Info("Server exited")
}

func setupLogging(cfg *config.Config) {
	// Set log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	if cfg.LogFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Set output
	logrus.SetOutput(os.Stdout)
}
