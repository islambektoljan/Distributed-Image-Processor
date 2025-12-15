package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"image-processor/internal/config"
	"image-processor/internal/queue/rabbitmq"
	minioclient "image-processor/internal/storage/minio"
	"image-processor/internal/worker"
	"image-processor/pkg/database/postgres"
	redisclient "image-processor/pkg/database/redis"

	"github.com/google/uuid"
)

const WorkerPoolSize = 5

type TaskMessage struct {
	ImageID    string `json:"image_id"`
	BucketName string `json:"bucket_name"`
	ObjectName string `json:"object_name"`
}

func main() {
	log.Println("Starting Worker Service...")

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

	// Create processor
	processor := worker.NewProcessor(pgPool, minioClient, redisClient)

	// Start consuming messages
	msgs, err := rabbitClient.Consume()
	if err != nil {
		log.Fatalf("Failed to start consuming: %v", err)
	}

	// Create worker pool
	var wg sync.WaitGroup
	taskChan := make(chan TaskMessage, WorkerPoolSize)

	// Start worker goroutines
	for i := 0; i < WorkerPoolSize; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			log.Printf("Worker %d started", workerID)

			for task := range taskChan {
				log.Printf("Worker %d processing image %s", workerID, task.ImageID)

				imageID, err := uuid.Parse(task.ImageID)
				if err != nil {
					log.Printf("Worker %d: invalid image ID %s: %v", workerID, task.ImageID, err)
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				err = processor.ProcessImage(ctx, imageID, task.BucketName, task.ObjectName)
				cancel()

				if err != nil {
					log.Printf("Worker %d: failed to process image %s: %v", workerID, task.ImageID, err)
				} else {
					log.Printf("Worker %d: successfully processed image %s", workerID, task.ImageID)
				}
			}

			log.Printf("Worker %d stopped", workerID)
		}(i + 1)
	}

	// Shutdown channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Worker Service is running. Press Ctrl+C to exit.")

	// Message consumer loop
	go func() {
		for msg := range msgs {
			var task TaskMessage
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				msg.Nack(false, false) // discard invalid message
				continue
			}

			log.Printf("Received task for image %s", task.ImageID)

			// Send to worker pool
			taskChan <- task

			// Acknowledge message
			msg.Ack(false)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down gracefully...")

	// Close task channel to stop workers
	close(taskChan)

	// Wait for all workers to finish
	wg.Wait()

	log.Println("Worker Service stopped")
}
