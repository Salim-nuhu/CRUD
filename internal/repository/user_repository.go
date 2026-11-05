package repository

import (
	// context carries cancellation and deadlines into database operations.
	"context"
	// errors creates and compares the named errors this package returns.
	"errors"
	// time provides maximum operation durations and UTC update timestamps.
	"time"

	// models defines the User structure that this repository stores and returns.
	"github.com/Salim-nuhu/CRUD/internal/models"
	// bson provides MongoDB's document types and ObjectID type.
	"go.mongodb.org/mongo-driver/v2/bson"
	// mongo contains the database, collection, and MongoDB error types.
	"go.mongodb.org/mongo-driver/v2/mongo"
	// options builds settings for MongoDB indexes and find operations.
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// These named errors let callers tell common user-related outcomes apart from
// unexpected database failures, without needing to inspect MongoDB's messages.
var (
	// ErrEmailTaken means MongoDB rejected an email that is already in use.
	ErrEmailTaken = errors.New("email already in use")
	// ErrNotFound means the requested user ID does not match a stored user.
	ErrNotFound = errors.New("user not found")
)

// UserRepository groups all database operations for users.
// Keeping the collection here is like giving each operation access to the
// same labeled drawer in the database filing cabinet.
type UserRepository struct {
	// coll is MongoDB's handle for the "users" collection.
	coll *mongo.Collection
}

// NewUserRepository selects the users collection and creates a unique index
// on email, so MongoDB itself prevents two users from sharing an email.
func NewUserRepository(db *mongo.Database) (*UserRepository, error) {
	// Collection gets a handle; MongoDB creates the collection when data is first
	// written if it does not already exist.
	coll := db.Collection("users")

	// Give index creation at most ten seconds so startup cannot wait forever.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// Always release the timer resources after the operation finishes.
	defer cancel()

	// Indexes speed up lookups and enforce rules. Value 1 means ascending order;
	// SetUnique makes MongoDB reject duplicate email values.
	_, err := coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		// Returning the error prevents startup from pretending the uniqueness
		// rule is active when MongoDB failed to create it.
		return nil, err
	}
	// Save the collection handle so later methods can use it.
	return &UserRepository{coll: coll}, nil
}

// Create stores a new user and fills in its MongoDB-generated ID.
func (repo *UserRepository) Create(ctx context.Context, user *models.User) error {
	// Limit the operation to five seconds, while still respecting any earlier
	// cancellation or deadline from the HTTP request.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// InsertOne sends this user document to MongoDB.
	res, err := repo.coll.InsertOne(ctx, user)
	if err != nil {
		// Convert the driver's duplicate-key error into the package-level error
		// so the HTTP handler can respond with a useful conflict status.
		if mongo.IsDuplicateKeyError(err) {
			return ErrEmailTaken
		}
		// Preserve other errors because they may indicate a timeout or outage.
		return err
	}

	// MongoDB assigns an ID on insert. Confirm it has the type expected by User.
	id, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		// A wrong ID type is unexpected and should not be reported as success.
		return errors.New("unexpected id type from MongoDB")
	}
	// The driver returns the generated ID separately, so copy it onto the user
	// that the caller passed in.
	user.ID = id
	// nil means the insert completed successfully.
	return nil
}

// GetByID finds one user by MongoDB's unique ID.
func (repo *UserRepository) GetByID(ctx context.Context, id bson.ObjectID) (*models.User, error) {
	// Keep a single database lookup from waiting longer than five seconds.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Decode copies the matching database document into this Go value.
	var user models.User
	// _id is MongoDB's standard field name for a document's unique ID.
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Give callers a stable package error rather than exposing a driver error.
		return nil, ErrNotFound
	}
	if err != nil {
		// Report other failures, such as a timeout or connection problem.
		return nil, err
	}
	// Return a pointer to the user we just decoded.
	return &user, nil
}

// List returns one page of users, newest first. limit is the maximum number of
// records to return; skip is how many matching records to pass over first.
func (repo *UserRepository) List(ctx context.Context, limit, skip int64) ([]models.User, error) {
	// Bound the query's runtime so it cannot hold resources forever.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Build the database query options. A sort value of -1 means descending,
	// so the most recently created user appears first.
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	// bson.D{} is an empty filter, meaning "match every user".
	cursor, err := repo.coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}

	// Start with an empty slice so an empty result becomes JSON [] rather than
	// null, which is usually easier for clients to handle as a list.
	users := []models.User{}
	// All reads every result from this page into users and closes the cursor.
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// Update changes a user's name and email, refreshes the update timestamp, and
// returns the saved version of that user.
func (repo *UserRepository) Update(ctx context.Context, id bson.ObjectID, name, email string) (*models.User, error) {
	// Prevent the update query from waiting longer than five seconds.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// $set changes only these fields, leaving the other document fields intact.
	// Use UTC so timestamps are consistent regardless of server location.
	update := bson.M{"$set": bson.M{
		"name":       name,
		"email":      email,
		"updated_at": time.Now().UTC(),
	}}
	// MongoDB can return either the old or new version; ask for the new version.
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	// Find the user by ID, apply the changes, and decode the updated document.
	var user models.User
	err := repo.coll.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&user)

	// Translate expected outcomes into the repository's stable error values.
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		// No record had this ID.
		return nil, ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		// Another user already owns the requested email.
		return nil, ErrEmailTaken
	case err != nil:
		// Keep unexpected database failures available to the caller.
		return nil, err
	}
	// No error means user contains the updated database document.
	return &user, nil
}

// Delete removes one user by ID. It returns ErrNotFound if there was no match.
func (repo *UserRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	// Bound the delete operation's runtime.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// DeleteOne removes at most one matching document.
	res, err := repo.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	// A successful database call can still match nothing, so check the count.
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	// nil indicates that one matching user was deleted.
	return nil
}

func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	// Limit email lookups so a slow database cannot hold the request indefinitely.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// Release the timeout timer when the lookup ends.
	defer cancel()

	// Prepare the value that will receive the matching user document.
	var user models.User
	// Search by the normalized email and decode the first match.
	err := repo.coll.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	// Translate a missing document into the repository's stable not-found error.
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	// Return other database errors to the handler for logging and response mapping.
	if err != nil {
		return nil, err
	}
	// Return the decoded account record.
	return &user, nil
}
