package dao

import (
	"context"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/minio/minio-go/v7"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
}
