package dao

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.orx.me/xbot/internal/conf"
)

var (
	minioClient *minio.Client
)

func NewMinioClient() (*minio.Client, error) {
	config := conf.Conf.S3
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

// InitMinio initializes the MinIO client and sets up S3MessageStorage
func InitMinio() error {
	// Skip if S3 config is not provided
	if conf.Conf.S3.Endpoint == "" {
		log.Println("S3 endpoint not configured, skipping MinIO initialization")
		return nil
	}

	// Initialize MinIO client
	var err error
	minioClient, err = NewMinioClient()
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Check if bucket exists and create it if it doesn't
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, conf.Conf.S3.Bucket)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		log.Printf("Creating bucket %s...", conf.Conf.S3.Bucket)
		err = minioClient.MakeBucket(ctx, conf.Conf.S3.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Bucket %s created successfully", conf.Conf.S3.Bucket)
	}

	// Create S3MessageStorage instance and set it as the default storage
	s3Storage := NewS3MessageStorage(minioClient, conf.Conf.S3.Bucket)
	defaultMessageStorage = s3Storage
	log.Println("S3MessageStorage initialized and set as default message storage")

	return nil
}
