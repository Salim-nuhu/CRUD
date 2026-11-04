package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// User is what we store in MongoDB and send back to clients.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	Name         string        `bson:"name"           json:"name"`
	Email        string        `bson:"email"          json:"email"`
	PasswordHash string        `bson:"password_hash"  json:"-"`
	CreatedAt    time.Time     `bson:"created_at"     json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"     json:"updated_at"`
}

// CreateUserRequest is what a client is allowed to send when creating a user.
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateUserRequest is what a client may send when updating a user.
// It has no password: changing passwords will be its own endpoint.
type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
