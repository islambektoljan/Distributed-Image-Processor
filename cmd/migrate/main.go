package main

import (
	"context"
	"log"
	"time"

	"image-processor/internal/config"
	"image-processor/pkg/database/postgres"
)

func main() {
	log.Println("Starting migration runner...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Loaded URL: %s", cfg.PostgresURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("Connecting to Postgres at %s", cfg.PostgresURL)
	pool, err := postgres.NewClient(ctx, cfg.PostgresURL)
	if err != nil {
		// Fallback for docker internal networking vs localhost
		// In docker-compose, internal service sees 'postgres', but localhost sees 'localhost'.
		// The default config uses localhost, which should work for this runner running on host.
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Connected to database. Running migrations...")
	if err := postgres.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migration runner finished successfully.")
}
