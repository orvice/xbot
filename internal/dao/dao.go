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

	// Message storage configuration
	storage := conf.Conf.MessageStorage
	log.Printf("Message storage configuration: %s", storage)

	// Initialize based on configuration or initialize both with priority
	switch storage {
	case storageTypeMongoDB:
		defaultMessageStorage = &MongoDBStorage{
			messagesColl: messagesColl,
		}

	case storageTypeS3:
		// Initialize only S3/MinIO
		errMinio := InitMinio()
		if errMinio != nil {
			log.Printf("Failed to initialize MinIO: %v", errMinio)
			return fmt.Errorf("failed to initialize configured storage S3: %w", errMinio)
		}
		log.Println("MinIO initialized and set as message storage")

	default:
	}

	return nil
}
