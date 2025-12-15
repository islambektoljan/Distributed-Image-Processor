package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"image-processor/internal/config"
	"image-processor/internal/handler"
	"image-processor/internal/queue/rabbitmq"
	minioclient "image-processor/internal/storage/minio"
	"image-processor/pkg/database/postgres"
	redisclient "image-processor/pkg/database/redis"
	"image-processor/pkg/security"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting API Gateway...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	pgPool, err := postgres.NewClient(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	// Run migrations
	if err := postgres.RunMigrations(ctx, pgPool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Minio
	log.Println("Connecting to Minio...")
	minioClient, err := minioclient.NewClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, false)
	if err != nil {
		log.Fatalf("Failed to connect to Minio: %v", err)
	}

	// Initialize RabbitMQ
	log.Println("Connecting to RabbitMQ...")
	rabbitClient, err := rabbitmq.NewClient(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitClient.Close()

	// Initialize Redis
	log.Println("Connecting to Redis...")
	redisClient, err := redisclient.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	log.Println("✓ Successfully connected to all services")

	// Initialize handler
	h := handler.NewHandler(pgPool, minioClient, rabbitClient, redisClient)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Health check endpoint (unprotected)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
		})
	})

	// Keycloak middleware setup
	jwksURL := cfg.KeycloakURL + "/realms/" + cfg.KeycloakRealm + "/protocol/openid-connect/certs"
	authMiddleware := security.AuthMiddleware(jwksURL, cfg.KeycloakClientID)

	// Protected API routes
	v1 := router.Group("/api/v1")
	v1.Use(authMiddleware)
	{
		v1.POST("/upload", h.UploadImage)
		v1.GET("/images/:id", h.GetImage)
	}

	// Start HTTP server in a goroutine
	srv := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	go func() {
		log.Println("Starting HTTP server on :3000")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("API Gateway is running. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
