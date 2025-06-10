package storage

import (
	"context"
	common "j-okx-ai/common"
	"j-okx-ai/modules/user/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (mongodbStore *mongodbStore) GetUserById(ctx context.Context, cond map[string]interface{}) (*model.User, error) {
	objectID, err := primitive.ObjectIDFromHex(cond["_id"].(string))
	if err != nil {
		return nil, common.ErrorSimpleMessage("invalid user ID format")
	}
	var user model.User
	collection := mongodbStore.db.Collection(user.CollectionName())

	err = collection.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &user, nil
}
