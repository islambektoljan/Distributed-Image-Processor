package handler

import (
	"image-processor/internal/queue/rabbitmq"
	minioclient "image-processor/internal/storage/minio"
	redisclient "image-processor/pkg/database/redis"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pgPool       *pgxpool.Pool
	minioClient  *minioclient.Client
	rabbitClient *rabbitmq.Client
	redisClient  *redisclient.Client
}

func NewHandler(pg *pgxpool.Pool, minio *minioclient.Client, rabbit *rabbitmq.Client, redis *redisclient.Client) *Handler {
	return &Handler{
		pgPool:       pg,
		minioClient:  minio,
		rabbitClient: rabbit,
		redisClient:  redis,
	}
}
