package dao

import (
	"context"
	"time"

	"github.com/go-telegram/bot/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Message struct {
	ID        *bson.ObjectID `bson:"_id"`
	Update    *models.Update `bson:"update"`
	ChatID    int64          `bson:"chat_id"`
	CreatedAt int64          `bson:"created_at"`
	UpdatedAt int64          `bson:"updated_at"`
}

func SaveMessage(ctx context.Context, message *Message) error {
	now := time.Now().Unix()
	message.CreatedAt = now
	message.UpdatedAt = now
	message.ChatID = message.Update.Message.Chat.ID
	_, err := messagesColl.InsertOne(ctx, message)
	return err
}

// GetMessageByChatID retrieves all messages for a specific chat ID
func GetMessageByChatID(ctx context.Context, chatID int64) ([]*Message, error) {
	cursor, err := messagesColl.Find(ctx, bson.M{"chat_id": chatID})
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
