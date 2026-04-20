package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"followupmedium-newsroom/internal/api"
	"followupmedium-newsroom/internal/config"
	"followupmedium-newsroom/internal/database"
	"followupmedium-newsroom/internal/mcp"
	"followupmedium-newsroom/internal/services"
	"followupmedium-newsroom/internal/workers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	// Initialize configuration
	cfg := config.Load()

	// Setup logging
	setupLogging(cfg.LogLevel)

	// Initialize database connections
	db, err := database.NewMongoDB(cfg.MongoURI)
	if err != nil {
		logrus.Fatal("Failed to connect to MongoDB: ", err)
	}
	defer db.Close()

	redis, err := database.NewRedis(cfg.RedisURI)
	if err != nil {
		logrus.Fatal("Failed to connect to Redis: ", err)
	}
	defer redis.Close()

	// Initialize services
	storyService := services.NewStoryService(db, redis)
	diffEngine := services.NewDiffEngine(redis)
	aiService := services.NewAIService(cfg.GeminiAPIKey, cfg.NewsAPIKey)
	
	// Configure D-ID if API key is provided
	if didAPIKey := os.Getenv("DID_API_KEY"); didAPIKey != "" {
		aiService.SetDIDService(didAPIKey)
	}

	// Configure HeyGen (preferred video generator)
	if heygenKey := os.Getenv("HEYGEN_API_KEY"); heygenKey != "" {
		aiService.SetHeyGenService(heygenKey, os.Getenv("HEYGEN_AVATAR_ID"), os.Getenv("HEYGEN_VOICE_ID"))
	}

	// Configure ElevenLabs if API key is provided (for voice cloning)
	if elKey := os.Getenv("ELEVENLABS_API_KEY"); elKey != "" {
		aiService.SetElevenLabsService(elKey)
	}
	
	// Initialize Google Imagen if project ID is configured
	if cfg.GoogleCloudProjectID != "" {
		imagenService, err := services.NewGoogleImagenService(cfg.GoogleCloudProjectID)
		if err != nil {
			logrus.Warn("Google Imagen service not configured: ", err)
		} else {
			aiService.SetImagenService(imagenService)
			logrus.Info("Google Imagen service initialized")
			defer imagenService.Close()
		}
	}
	
	// Initialize S3 service if credentials are provided
	if cfg.AWSAccessKey != "" && cfg.AWSSecretKey != "" && cfg.AWSBucket != "" {
		s3Service, err := services.NewS3Service(cfg.AWSAccessKey, cfg.AWSSecretKey, cfg.AWSBucket, cfg.AWSRegion)
		if err != nil {
			logrus.Warn("S3 service not configured: ", err)
		} else {
			aiService.SetS3Service(s3Service)
			logrus.Info("S3 service initialized")
		}
	}
	
	rssService := services.NewRSSService(db, redis, cfg.RSSFeeds)
	authService := services.NewAuthService(db.Database) // Pass the mongo.Database field

	// Initialize worker pool
	workerPool := workers.NewWorkerPool(cfg.WorkerPoolSize, storyService, diffEngine)
	workerPool.Start()
	defer workerPool.Stop()

	// Initialize MCP server
	mcpServer := mcp.NewServer(storyService, aiService)
	go mcpServer.Start(cfg.MCPPort)

	// Initialize HTTP server
	router := setupRouter(cfg, storyService, diffEngine, rssService, aiService, authService)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logrus.Infof("HTTP Server is now listening on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown: ", err)
	}

	logrus.Info("Server exited")
}

func setupLogging(level string) {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	
	switch level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func setupRouter(cfg *config.Config, storyService *services.StoryService, diffEngine *services.DiffEngine, rssService *services.RSSService, aiService *services.AIService, authService *services.AuthService) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(api.RateLimitMiddleware(cfg.RateLimit))
	router.Use(api.CORSMiddleware())

	// Health check
	router.GET("/health", api.HealthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	api.SetupRoutes(v1, storyService, diffEngine)
	api.SetupRSSRoutes(v1, rssService, aiService, authService)
	api.SetupAuthRoutes(v1, authService)

	return router
}