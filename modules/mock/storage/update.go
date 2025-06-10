package storage

import (
	"context"
	common "j-okx-ai/common"
	"j-okx-ai/modules/mock/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mongodbStore *mongodbStore) UpdateMock(ctx context.Context, cond map[string]interface{}, dataUpdate *model.Mock) error {
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "updated_at", Value: time.Now().UTC()},
		}},
	}

	if idStr, ok := cond["_id"].(string); ok {
		objectID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return common.ErrorSimpleMessage("invalid novel ID format")
		}
		cond["_id"] = objectID
	}

	filter := bson.D{}
	for key, value := range cond {
		filter = append(filter, bson.E{Key: key, Value: value})
	}

	collection := mongodbStore.db.Collection(dataUpdate.CollectionName())

	result := collection.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return common.ErrEntityNotFoundEntity(model.EntityName, result.Err())
		}
		return result.Err()
	}

	var updatedNovel model.Mock
	err := result.Decode(&updatedNovel)
	if err != nil {
		return err
	}

	return nil
}
