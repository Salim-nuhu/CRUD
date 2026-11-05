package models

import (
	// time stores the user creation and update timestamps.
	"time"

	// bson provides MongoDB document field types and ObjectIDs.
	"go.mongodb.org/mongo-driver/v2/bson"
)

// User is what we store in MongoDB and send back to clients.
type User struct {
	// ID is MongoDB's identifier and is omitted from inserts when it is empty.
	ID bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	// Name is the user's display name in both MongoDB and API JSON.
	Name string `bson:"name"           json:"name"`
	// Email is the user's normalized email address.
	Email string `bson:"email"          json:"email"`
	// PasswordHash is stored in MongoDB but deliberately omitted from API JSON.
	PasswordHash string `bson:"password_hash"  json:"-"`
	// CreatedAt records when the user record was first created.
	CreatedAt time.Time `bson:"created_at"     json:"created_at"`
	// UpdatedAt records when the user record was last modified.
	UpdatedAt time.Time `bson:"updated_at"     json:"updated_at"`
}

// CreateUserRequest is what a client is allowed to send when creating a user.
type CreateUserRequest struct {
	// Name is the display name requested by the client.
	Name string `json:"name"`
	// Email is the account email address requested by the client.
	Email string `json:"email"`
	// Password is supplied only for account creation and is hashed before storage.
	Password string `json:"password"`
}

// UpdateUserRequest is what a client may send when updating a user.
// It has no password: changing passwords will be its own endpoint.
type UpdateUserRequest struct {
	// Name is the replacement display name.
	Name string `json:"name"`
	// Email is the replacement account email address.
	Email string `json:"email"`
}

// LoginRequest is what a client sends to log in.
type LoginRequest struct {
	// Email identifies the account attempting to authenticate.
	Email string `json:"email"`
	// Password is checked against the account's stored password hash.
	Password string `json:"password"`
}
