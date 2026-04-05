package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BaseModel contains common fields for all models.
type BaseModel struct {
	ID        string        `json:"id" bson:"id"`
	MongoID   bson.ObjectID `json:"-" bson:"_id,omitempty"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at"`
}
