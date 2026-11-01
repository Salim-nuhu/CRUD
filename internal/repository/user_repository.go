package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Salim-nuhu/CRUD/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrEmailTaken gives the rest of the app a recognizable duplicate-email error.
var ErrEmailTaken = errors.New("email already in use")

// UserRepository keeps MongoDB operations for users in one place.
// The collection is like one labeled drawer in the database filing cabinet.
type UserRepository struct {
	coll *mongo.Collection
}

// NewUserRepository selects the users collection and ensures emails are unique.
func NewUserRepository(db *mongo.Database) (*UserRepository, error) {
	coll := db.Collection("users")

	// A unique index makes MongoDB reject duplicates even if two requests arrive
	// at almost the same time. It is the database's own "one copy per email" rule.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}
	return &UserRepository{coll: coll}, nil
}

// Create inserts one user and copies MongoDB's generated ID back into that user.
func (repo *UserRepository) Create(ctx context.Context, user *models.User) error {
	// Do not let a slow database operation hold the request open indefinitely.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// InsertOne sends this user document to the users collection.
	res, err := repo.coll.InsertOne(ctx, user)
	if err != nil {
		// Convert the driver's database-specific duplicate error into the app's
		// simpler ErrEmailTaken value so the handler can return HTTP 409.
		if mongo.IsDuplicateKeyError(err) {
			return ErrEmailTaken
		}
		return err
	}

	// MongoDB creates the ID during insertion. Check its type before assigning it.
	id, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return errors.New("unexpected id type from MongoDB")
	}
	user.ID = id
	return nil
}
