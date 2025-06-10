package storage

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type mongodbStore struct {
	db *mongo.Database
}

func NewMongoDbStore(db *mongo.Database) *mongodbStore {
	return &mongodbStore{db: db}
}
