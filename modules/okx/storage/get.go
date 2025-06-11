package storage

import (
	"context"
	common "j-ai-trade/common"
	"j-ai-trade/modules/okx/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (mongodbStore *mongodbStore) GetMockById(ctx context.Context, cond map[string]interface{}) (*model.Okx, error) {
	objectID, err := primitive.ObjectIDFromHex(cond["_id"].(string))
	if err != nil {
		return nil, common.ErrorSimpleMessage("invalid novel ID format")
	}
	var novel model.Okx
	collection := mongodbStore.db.Collection(novel.CollectionName())

	err = collection.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&novel)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &novel, nil
}
