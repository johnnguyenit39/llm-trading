package storage

import (
	"context"
	"j-okx-ai/common"
	"j-okx-ai/modules/okx/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mongodbStore *mongodbStore) GetMocks(ctx context.Context, paging *common.Pagination) ([]model.Okx, error) {
	filter := bson.D{}
	var novel model.Okx
	collection := mongodbStore.db.Collection(novel.CollectionName())

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

	var novels []model.Okx
	for cursor.Next(ctx) {
		var novel model.Okx
		if err := cursor.Decode(&novel); err != nil {
			return nil, err
		}
		novels = append(novels, novel)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return novels, nil
}
