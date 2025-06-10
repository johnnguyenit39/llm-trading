package common

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseModel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	CreatedAt time.Time          `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at,omitempty" json:"-"`
	DeletedAt *time.Time         `bson:"deleted_at,omitempty" json:"-"`
}

// NewBaseModel initializes a new BaseModel with UUID (or ObjectID) and timestamps
func NewBaseModel() *BaseModel {
	now := time.Now().UTC()
	return &BaseModel{
		ID:        primitive.NewObjectID(), // GeneMock ObjectID
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetUpdatedAt updates the UpdatedAt field
func (baseModel *BaseModel) SetUpdatedAt() {
	baseModel.UpdatedAt = time.Now().UTC()
}
