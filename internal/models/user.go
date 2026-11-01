package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User is both the database record and the object returned to API clients.
// Struct tags are labels that tell MongoDB and JSON how to name each field.
type User struct {
	// ObjectID is MongoDB's unique ID type. "omitempty" lets MongoDB generate
	// it when creating a record; JSON clients see the simpler name "id".
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string        `bson:"name"          json:"name"`
	Email     string        `bson:"email"         json:"email"`
	CreatedAt time.Time     `bson:"created_at"    json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"    json:"updated_at"`
}

// CreateUserRequest contains only the fields clients may provide when creating
// a user. The server fills in the database ID and timestamps itself.
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
