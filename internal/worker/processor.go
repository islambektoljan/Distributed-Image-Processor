package worker

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"log"

	"image-processor/internal/models"
	minioclient "image-processor/internal/storage/minio"
	redisclient "image-processor/pkg/database/redis"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Processor struct {
	pgPool      *pgxpool.Pool
	minioClient *minioclient.Client
	redisClient *redisclient.Client
}

func NewProcessor(pg *pgxpool.Pool, minio *minioclient.Client, redis *redisclient.Client) *Processor {
	return &Processor{
		pgPool:      pg,
		minioClient: minio,
		redisClient: redis,
	}
}

func (p *Processor) ProcessImage(ctx context.Context, imageID uuid.UUID, bucketName, objectName string) error {
	log.Printf("Starting processing for image %s", imageID)

	// Update status to processing
	if err := p.updateStatus(ctx, imageID, models.ImageStatusProcessing); err != nil {
		log.Printf("Failed to update status to processing: %v", err)
		return err
	}

	// Download image from Minio
	log.Printf("Downloading image from Minio: %s/%s", bucketName, objectName)
	obj, err := p.minioClient.DownloadFile(ctx, bucketName, objectName)
	if err != nil {
		p.updateStatus(ctx, imageID, models.ImageStatusFailed)
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer obj.Close()

	// Decode image
	img, err := imaging.Decode(obj)
	if err != nil {
		p.updateStatus(ctx, imageID, models.ImageStatusFailed)
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to 800px width (maintain aspect ratio)
	log.Printf("Resizing image to 800px width")
	img = imaging.Resize(img, 800, 0, imaging.Lanczos)

	// Apply grayscale filter
	log.Printf("Applying grayscale filter")
	img = imaging.Grayscale(img)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		p.updateStatus(ctx, imageID, models.ImageStatusFailed)
		return fmt.Errorf("failed to encode image: %w", err)
	}

	// Upload to processed-images bucket
	processedObjectName := fmt.Sprintf("%s.png", imageID.String())
	log.Printf("Uploading processed image to Minio: processed-images/%s", processedObjectName)
	_, err = p.minioClient.UploadFile(ctx, "processed-images", processedObjectName, &buf, int64(buf.Len()), "image/png")
	if err != nil {
		p.updateStatus(ctx, imageID, models.ImageStatusFailed)
		return fmt.Errorf("failed to upload processed image: %w", err)
	}

	// Update status to completed
	if err := p.updateStatus(ctx, imageID, models.ImageStatusCompleted); err != nil {
		log.Printf("Failed to update status to completed: %v", err)
		return err
	}

	// Invalidate Redis cache
	cacheKey := fmt.Sprintf("image:%s", imageID.String())
	if err := p.redisClient.Delete(ctx, cacheKey); err != nil {
		log.Printf("Warning: failed to invalidate cache for %s: %v", cacheKey, err)
		// Don't fail the entire operation if cache invalidation fails
	}

	log.Printf("Successfully processed image %s", imageID)
	return nil
}

func (p *Processor) updateStatus(ctx context.Context, imageID uuid.UUID, status models.ImageStatus) error {
	query := `UPDATE images SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := p.pgPool.Exec(ctx, query, status, imageID)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	log.Printf("Updated image %s status to: %s", imageID, status)
	return nil
}
