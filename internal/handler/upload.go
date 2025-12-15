package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"image-processor/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const MaxUploadSize = 10 << 20 // 10MB

type UploadResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type TaskMessage struct {
	ImageID    string `json:"image_id"`
	BucketName string `json:"bucket_name"`
	ObjectName string `json:"object_name"`
}

func (h *Handler) UploadImage(c *gin.Context) {
	// Set max upload size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize)

	// Parse multipart form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request"})
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/jpeg") && !strings.HasPrefix(contentType, "image/png") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPEG and PNG files are allowed"})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only .jpg, .jpeg, and .png extensions are allowed"})
		return
	}

	// Generate UUID for image
	imageID := uuid.New()
	objectName := fmt.Sprintf("%s%s", imageID.String(), ext)
	bucketName := "raw-images"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Upload to Minio
	_, err = h.minioClient.UploadFile(ctx, bucketName, objectName, file, header.Size, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}

	// Insert record into PostgreSQL
	query := `
		INSERT INTO images (id, filename, status, bucket_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err = h.pgPool.Exec(ctx, query, imageID, header.Filename, models.ImageStatusPending, bucketName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save to database: %v", err)})
		return
	}

	// Publish message to RabbitMQ
	taskMsg := TaskMessage{
		ImageID:    imageID.String(),
		BucketName: bucketName,
		ObjectName: objectName,
	}
	msgBytes, err := json.Marshal(taskMsg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task message"})
		return
	}

	err = h.rabbitClient.Publish(msgBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to publish message: %v", err)})
		return
	}

	// Return success response
	c.JSON(http.StatusCreated, UploadResponse{
		ID:       imageID.String(),
		Filename: header.Filename,
		Status:   string(models.ImageStatusPending),
		Message:  "Image uploaded successfully and queued for processing",
	})
}
