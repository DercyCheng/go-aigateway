package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-aigateway/internal/config"
	"go-aigateway/internal/middleware"
	"go-aigateway/internal/router"
	"go-aigateway/internal/discovery"
	"go-aigateway/internal/protocol"
	"go-aigateway/internal/ram"
	"go-aigateway/internal/cloud"

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

	// Setup logging
	setupLogging(cfg)

	// Initialize services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize service discovery
	serviceDiscovery, err := discovery.NewManager(&cfg.ServiceDiscovery)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize service discovery")
	}
	defer serviceDiscovery.Close()

	// Initialize protocol converter
	protocolConverter := protocol.NewProtocolConverter(&cfg.ProtocolConversion)

	// Initialize RAM authenticator
	ramAuth := ram.NewRAMAuthenticator(&cfg.RAMAuth)

	// Initialize cloud integrator
	cloudIntegrator, err := cloud.NewCloudIntegrator(&cfg.CloudIntegration)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize cloud integrator")
	}

	// Setup Gin mode
	gin.SetMode(cfg.GinMode)

	// Initialize router
	r := gin.New()

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.PrometheusMetrics())
	r.Use(middleware.RateLimiter(cfg.RateLimit))

	// Add protocol conversion middleware if enabled
	if cfg.ProtocolConversion.Enabled {
		r.Use(func(c *gin.Context) {
			// Add protocol converter to context for handlers to use
			c.Set("protocol_converter", protocolConverter)
			c.Next()
		})
	}

	// Add RAM authentication middleware if enabled
	if cfg.RAMAuth.Enabled {
		r.Use(middleware.RAMAuth(ramAuth))
	}

	// Setup routes
	router.SetupRoutes(r, cfg)
	
	// Setup cloud management routes
	router.SetupCloudRoutes(r, cloudIntegrator)

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
