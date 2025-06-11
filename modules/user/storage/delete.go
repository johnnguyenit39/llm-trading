package storage

import (
	"context"
	common "j-ai-trade/common"
	"j-ai-trade/modules/user/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (mongodbStore *mongodbStore) DeleteUser(ctx context.Context, cond map[string]interface{}) (bool, error) {
	// Convert the string id to ObjectID
	objectID, err := primitive.ObjectIDFromHex(cond["_id"].(string))
	if err != nil {
		return false, common.ErrorSimpleMessage("invalid user ID format")
	}

	// Prepare the delete filter
	filter := bson.D{{Key: "_id", Value: objectID}}
	user := model.User{}
	// Perform the delete operation
	collection := mongodbStore.db.Collection(user.CollectionName())
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
