package storage

import (
	"context"
	common "j-okx-ai/common"
	"j-okx-ai/modules/user/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (mongodbStore *mongodbStore) GetUserByPhoneNumber(ctx context.Context, cond map[string]interface{}) (*model.User, error) {

	var user model.User
	collection := mongodbStore.db.Collection(user.CollectionName())

	err := collection.FindOne(ctx, bson.D{{Key: "phone_number", Value: cond["phone_number"]}}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, common.ErrEntityNotFoundEntity(model.EntityName, err)
		}
		return nil, err
	}

	return &user, nil
}
