package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"image-processor/internal/config"
	"image-processor/internal/queue/rabbitmq"
	minioclient "image-processor/internal/storage/minio"
	"image-processor/pkg/database/postgres"
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
	_, err = minioclient.NewClient(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, false)
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

	log.Println("✓ Successfully connected to Minio and RabbitMQ")
	log.Println("API Gateway is running. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
}
