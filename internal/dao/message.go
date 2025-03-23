package dao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	storageTypeMongoDB = "mongodb"
	storageTypeS3      = "s3"
)

type MessageStorage interface {
	SaveMessage(ctx context.Context, message *Message) error
	GetMessageByChatID(ctx context.Context, chatID int64) ([]*Message, error)
}

type Message struct {
	ID        bson.ObjectID  `bson:"_id,omitempty"`
	Update    *models.Update `bson:"update"`
	ChatID    int64          `bson:"chat_id"`
	CreatedAt int64          `bson:"created_at"`
	UpdatedAt int64          `bson:"updated_at"`
}

var (
	defaultMessageStorage MessageStorage
)

func GetMessageStorage() MessageStorage {
	return defaultMessageStorage
}

type MongoDBStorage struct {
	messagesColl *mongo.Collection
}

func (s *MongoDBStorage) SaveMessage(ctx context.Context, message *Message) error {
	now := time.Now().Unix()
	message.CreatedAt = now
	message.UpdatedAt = now
	// Handle potential nil values to avoid panic
	if message.Update != nil && message.Update.Message != nil {
		message.ChatID = message.Update.Message.Chat.ID
	}
	result, err := s.messagesColl.InsertOne(ctx, message)
	if nil != err {
		return err
	}
	message.ID = result.InsertedID.(bson.ObjectID)
	return nil
}

// GetMessageByChatID retrieves all messages for a specific chat ID
func (s *MongoDBStorage) GetMessageByChatID(ctx context.Context, chatID int64) ([]*Message, error) {
	cursor, err := s.messagesColl.Find(ctx, bson.M{"chat_id": chatID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*Message
	for cursor.Next(ctx) {
		var message Message
		if err := cursor.Decode(&message); err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

type S3MessageStorage struct {
	client *minio.Client
	bucket string
}

// NewS3MessageStorage creates a new S3MessageStorage
func NewS3MessageStorage(client *minio.Client, bucket string) *S3MessageStorage {
	return &S3MessageStorage{
		client: client,
		bucket: bucket,
	}
}

// generateKey creates a key in the format "chatID/year/month/day/messageID.json"
func (s *S3MessageStorage) generateKey(chatID int64, messageID string, t time.Time) string {
	return fmt.Sprintf("%d/%04d/%02d/%02d/%s.json",
		chatID,
		t.Year(),
		t.Month(),
		t.Day(),
		messageID)
}

// SaveMessage saves a message to S3 storage
func (s *S3MessageStorage) SaveMessage(ctx context.Context, message *Message) error {
	// Set the timestamp
	now := time.Now()
	message.CreatedAt = now.Unix()
	message.UpdatedAt = now.Unix()

	// Ensure chat ID is set
	if message.Update != nil && message.Update.Message != nil {
		message.ChatID = message.Update.Message.Chat.ID
	}

	// Generate a message ID if not exists
	if message.ID.IsZero() {
		message.ID = bson.NewObjectID()
	}

	// Convert message to JSON
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Generate key for S3
	key := s.generateKey(message.ChatID, message.ID.Hex(), now)

	// Upload to S3
	_, err = s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(messageJSON), int64(len(messageJSON)),
		minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		return fmt.Errorf("failed to store message in S3: %w", err)
	}

	return nil
}

// GetMessageByChatID retrieves messages from the last 7 days for a specific chat ID
func (s *S3MessageStorage) GetMessageByChatID(ctx context.Context, chatID int64) ([]*Message, error) {
	// Get the date range (today and 6 days before)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -6) // 7 days including today

	var messages []*Message

	// Iterate through the last 7 days
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		// Create prefix for this day
		dayPrefix := fmt.Sprintf("%d/%04d/%02d/%02d/",
			chatID,
			date.Year(),
			date.Month(),
			date.Day())

		// List objects with this prefix
		objects := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
			Prefix:    dayPrefix,
			Recursive: true,
		})

		// Process each object
		for object := range objects {
			if object.Err != nil {
				return nil, fmt.Errorf("error listing objects: %w", object.Err)
			}

			// Get the object
			obj, err := s.client.GetObject(ctx, s.bucket, object.Key, minio.GetObjectOptions{})
			if err != nil {
				return nil, fmt.Errorf("error getting object %s: %w", object.Key, err)
			}

			// Read the object
			var buffer bytes.Buffer
			if _, err := buffer.ReadFrom(obj); err != nil {
				obj.Close()
				return nil, fmt.Errorf("error reading object %s: %w", object.Key, err)
			}
			obj.Close()

			// Unmarshal the JSON
			var message Message
			if err := json.Unmarshal(buffer.Bytes(), &message); err != nil {
				return nil, fmt.Errorf("error unmarshaling message from %s: %w", object.Key, err)
			}

			// Add to messages
			messages = append(messages, &message)
		}
	}

	// Sort messages by creation time (oldest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	return messages, nil
}
