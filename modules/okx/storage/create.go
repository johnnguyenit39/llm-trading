package storage

import (
	"context"
	"j-okx-ai/modules/okx/model"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (mongodbStore *mongodbStore) CreateMock(ctx context.Context, data *model.Okx) error {
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
