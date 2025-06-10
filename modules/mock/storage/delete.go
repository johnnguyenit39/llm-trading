package storage

import (
	"context"
	common "j-okx-ai/common"
	"j-okx-ai/modules/mock/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (mongodbStore *mongodbStore) DeleteMock(ctx context.Context, cond map[string]interface{}) (bool, error) {
	// Convert the string id to ObjectID
	objectID, err := primitive.ObjectIDFromHex(cond["_id"].(string))
	if err != nil {
		return false, common.ErrorSimpleMessage("invalid novel ID format")
	}

	// Prepare the delete filter
	filter := bson.D{{Key: "_id", Value: objectID}}
	novel := model.Mock{}
	// Perform the delete operation
	collection := mongodbStore.db.Collection(novel.CollectionName())
	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return false, err
	}

	// Check if any document was deleted
	if result.DeletedCount == 0 {
		return false, common.ErrEntityNotFoundEntity(model.EntityName, common.ErrorSimpleMessage("entity not found"))
	}

	return false, nil
}
