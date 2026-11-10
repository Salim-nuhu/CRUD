package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// The roles a user can have. Admin exists now so we can use it later,
// but registration will never allow it.
const (
	RoleBuyer = "buyer"
	RoleOwner = "owner"
	RoleAdmin = "admin"
)

// User is what we store in MongoDB and send back to clients.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty"          json:"id"`
	Name         string        `bson:"name"                   json:"name"`
	Email        string        `bson:"email"                  json:"email"`
	Phone        string        `bson:"phone,omitempty"        json:"phone,omitempty"`
	Role         string        `bson:"role"                   json:"role"`
	PasswordHash string        `bson:"password_hash"          json:"-"`
	CreatedAt    time.Time     `bson:"created_at"             json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"             json:"updated_at"`
}

// CreateUserRequest is what a client is allowed to send when registering.
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

// UpdateUserRequest is what a client may send when updating a user.
// It deliberately has no Role and no Password.
type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// LoginRequest is what a client sends to log in.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
