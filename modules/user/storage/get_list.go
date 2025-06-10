package storage

import (
	"context"
	"j-okx-ai/common"
	"j-okx-ai/modules/user/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mongodbStore *mongodbStore) GetUsers(ctx context.Context, paging *common.Pagination) ([]model.User, error) {
	filter := bson.D{}
	var user model.User
	collection := mongodbStore.db.Collection(user.CollectionName())

	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	paging.Count = totalCount

	options := options.Find().
		SetSkip(int64((paging.PageNumber - 1) * paging.PageSize)).
		SetLimit(int64(paging.PageSize))

	cursor, err := collection.Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []model.User
	for cursor.Next(ctx) {
		var user model.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
