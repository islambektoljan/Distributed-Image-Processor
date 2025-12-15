package minio

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client *minio.Client
}

// NewClient creates a new Minio client and ensures buckets exist
func NewClient(endpoint, accessKey, secretKey string, useSSL bool) (*Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	client := &Client{client: minioClient}

	// Create buckets if they don't exist
	buckets := []string{"raw-images", "processed-images"}
	for _, bucketName := range buckets {
		if err := client.ensureBucketExists(context.Background(), bucketName); err != nil {
			return nil, fmt.Errorf("failed to ensure bucket %s exists: %w", bucketName, err)
		}
	}

	log.Printf("Minio client initialized successfully with buckets: %v", buckets)
	return client, nil
}

// ensureBucketExists creates a bucket if it doesn't exist
func (c *Client) ensureBucketExists(ctx context.Context, bucketName string) error {
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created bucket: %s", bucketName)
	} else {
		log.Printf("Bucket already exists: %s", bucketName)
	}

	return nil
}

// UploadFile uploads a file to the specified bucket
func (c *Client) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	uploadInfo, err := c.client.PutObject(ctx, bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("Successfully uploaded %s to bucket %s", objectName, bucketName)
	return uploadInfo, nil
}

// GetFileLink generates a presigned URL for file download
func (c *Client) GetFileLink(ctx context.Context, bucketName, objectName string, expires time.Duration) (string, error) {
	presignedURL, err := c.client.PresignedGetObject(ctx, bucketName, objectName, expires, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

// DownloadFile downloads a file from the specified bucket
func (c *Client) DownloadFile(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	object, err := c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	return object, nil
}
