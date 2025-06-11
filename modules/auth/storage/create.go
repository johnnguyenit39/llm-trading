package storage

import (
	"context"
	"j-ai-trade/modules/user/model"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (mongodbStore *mongodbStore) Register(ctx context.Context, data *model.User) error {
	if data.ID == primitive.NilObjectID {
		data.ID = primitive.NewObjectID()
	}
	if data.CreatedAt.IsZero() {
		data.CreatedAt = time.Now().UTC()
	}
	if data.UpdatedAt.IsZero() {
		data.UpdatedAt = time.Now().UTC()
	}

	_, err := mongodbStore.db.Collection(data.CollectionName()).InsertOne(ctx, data)

	return err
}
