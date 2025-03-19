package dao

import (
	"context"
	"time"

	"github.com/go-telegram/bot/models"
)

type Message struct {
	Update    *models.Update `bson:"update"`
	CreatedAt int64          `bson:"created_at"`
	UpdatedAt int64          `bson:"updated_at"`
}

func SaveMessage(ctx context.Context, message *Message) error {
	now := time.Now().Unix()
	message.CreatedAt = now
	message.UpdatedAt = now
	_, err := messagesColl.InsertOne(ctx, message)
	return err
}
