package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"image-processor/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ImageResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Status      string    `json:"status"`
	BucketName  string    `json:"bucket_name"`
	DownloadURL string    `json:"download_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h *Handler) GetImage(c *gin.Context) {
	idParam := c.Param("id")

	// Validate UUID
	imageID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("image:%s", imageID.String())

	// Check Redis cache first
	cachedData, err := h.redisClient.Get(ctx, cacheKey)
	if err == nil {
		// Cache hit
		var response ImageResponse
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			// Regenerate presigned URL (they expire)
			if response.Status == string(models.ImageStatusCompleted) {
				downloadURL, _ := h.minioClient.GetFileLink(ctx, "processed-images", response.ID+".png", 15*time.Minute)
				response.DownloadURL = downloadURL
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Cache miss - query PostgreSQL
	var image models.Image
	query := `
		SELECT id, filename, status, bucket_name, created_at, updated_at
		FROM images
		WHERE id = $1
	`
	err = h.pgPool.QueryRow(ctx, query, imageID).Scan(
		&image.ID,
		&image.Filename,
		&image.Status,
		&image.BucketName,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	response := ImageResponse{
		ID:         image.ID.String(),
		Filename:   image.Filename,
		Status:     string(image.Status),
		BucketName: image.BucketName,
		CreatedAt:  image.CreatedAt,
		UpdatedAt:  image.UpdatedAt,
	}

	// Generate presigned URL if image is completed
	if image.Status == models.ImageStatusCompleted {
		// Assume processed images are stored with .png extension
		objectName := fmt.Sprintf("%s.png", image.ID.String())
		downloadURL, err := h.minioClient.GetFileLink(ctx, "processed-images", objectName, 15*time.Minute)
		if err == nil {
			response.DownloadURL = downloadURL
		}
	}

	// Cache the result in Redis (TTL: 10 minutes)
	responseBytes, _ := json.Marshal(response)
	_ = h.redisClient.Set(ctx, cacheKey, string(responseBytes), 10*time.Minute)

	c.JSON(http.StatusOK, response)
}
