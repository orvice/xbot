package dao

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Pull struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Type      string
	Date      string
	ChatID    int64
	MessageID int64
	CreatedAt int64
	UpdatedAt int64
}

func SavePull(ctx context.Context, pull Pull) error {
	now := time.Now().Unix()
	pull.CreatedAt = now
	pull.UpdatedAt = now
	result, err := pullColl.InsertOne(ctx, pull)
	if nil != err {
		return err
	}
	pull.ID = result.InsertedID.(bson.ObjectID)
	return nil
}

func GetPullByTypeAndDate(ctx context.Context, pullType string, date string) (*Pull, bool, error) {
	var pull Pull
	err := pullColl.FindOne(ctx, bson.M{"type": pullType, "date": date}).Decode(&pull)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &pull, true, nil
}
