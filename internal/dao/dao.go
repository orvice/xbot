package dao

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.orx.me/xbot/internal/conf"
)

// ErrNoStorage is returned when no storage is configured
var ErrNoStorage = errors.New("no message storage configured")

// Init initializes all database connections and storage components
func Init(ctx context.Context) error {
	log.Println("Initializing data access layer...")

	var errMongo, errMinio error

	// Message storage configuration
	storage := conf.Conf.MessageStorage
	log.Printf("Message storage configuration: %s", storage)

	// Initialize based on configuration or initialize both with priority
	switch storage {
	case storageTypeMongoDB:
		// Initialize only MongoDB
		errMongo = InitMongo(ctx)
		if errMongo != nil {
			log.Printf("Failed to initialize MongoDB: %v", errMongo)
			return fmt.Errorf("failed to initialize configured storage MongoDB: %w", errMongo)
		}
		log.Println("MongoDB initialized and set as message storage")

	case storageTypeS3:
		// Initialize only S3/MinIO
		errMinio = InitMinio()
		if errMinio != nil {
			log.Printf("Failed to initialize MinIO: %v", errMinio)
			return fmt.Errorf("failed to initialize configured storage S3: %w", errMinio)
		}
		log.Println("MinIO initialized and set as message storage")

	default:
		// If no specific configuration or unknown, try both with MongoDB as primary
		log.Println("No specific message storage configured, trying MongoDB first, then S3 as fallback")
		
		// Initialize MongoDB first
		errMongo = InitMongo(ctx)
		if errMongo != nil {
			log.Printf("Failed to initialize MongoDB: %v", errMongo)
		} else {
			log.Println("MongoDB initialized successfully and set as default message storage")
		}

		// Try MinIO as fallback or secondary
		errMinio = InitMinio()
		if errMinio != nil {
			log.Printf("Failed to initialize MinIO: %v", errMinio)
		} else if errMongo == nil {
			// Both initialized, but MongoDB is primary
			log.Println("MinIO initialized successfully (MongoDB will be used as default storage)")
		} else {
			// MongoDB failed, but MinIO worked - use MinIO
			log.Println("MinIO initialized and set as default message storage")
		}
		
		// Check if at least one storage is available
		if errMongo != nil && errMinio != nil {
			return ErrNoStorage
		}
	}

	return nil
}
