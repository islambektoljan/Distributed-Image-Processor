package models

import (
	"time"

	"github.com/google/uuid"
)

type ImageStatus string

const (
	ImageStatusPending    ImageStatus = "pending"
	ImageStatusProcessing ImageStatus = "processing"
	ImageStatusCompleted  ImageStatus = "completed"
	ImageStatusFailed     ImageStatus = "failed"
)

type Image struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	Filename   string      `json:"filename" db:"filename"`
	Status     ImageStatus `json:"status" db:"status"`
	BucketName string      `json:"bucket_name" db:"bucket_name"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
}
