package dao

import (
	"context"
	"time"

	"github.com/go-telegram/bot/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Poll struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Type      string
	Date      string
	ChatID    int64
	MessageID int64
	CreatedAt int64
	UpdatedAt int64
	PollID    string `bson:"poll_id"`
	Poll      *models.Poll
}

func SavePoll(ctx context.Context, Poll Poll) error {
	now := time.Now().Unix()
	Poll.CreatedAt = now
	Poll.UpdatedAt = now
	result, err := pollColl.InsertOne(ctx, Poll)
	if nil != err {
		return err
	}
	Poll.ID = result.InsertedID.(bson.ObjectID)
	return nil
}

func GetPollByTypeAndDate(ctx context.Context, PollType string, date string) (*Poll, bool, error) {
	var Poll Poll
	err := pollColl.FindOne(ctx, bson.M{"type": PollType, "date": date}).Decode(&Poll)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &Poll, true, nil
}

func GetPollByID(ctx context.Context, pollID string) (*Poll, error) {
	objectID, err := bson.ObjectIDFromHex(pollID)
	if err != nil {
		return nil, err
	}

	var poll Poll
	err = pollColl.FindOne(ctx, bson.M{"poll_id": objectID}).Decode(&poll)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &poll, nil
}
