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

var ErrPropertyNotFound = errors.New("property not found")

type PropertyRepository struct {
	coll *mongo.Collection
}

func NewPropertyRepository(db *mongo.Database) (*PropertyRepository, error) {
	coll := db.Collection("properties")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Indexes make common queries fast, like a book's index saves you from
	// reading every page. Without them MongoDB scans the whole collection.
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// The public list: active properties, newest first.
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
		// "Show me this owner's properties" (Step 4).
		{Keys: bson.D{{Key: "owner_id", Value: 1}}},
	})
	if err != nil {
		return nil, err
	}
	return &PropertyRepository{coll: coll}, nil
}

func (repo *PropertyRepository) Create(ctx context.Context, p *models.Property) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := repo.coll.InsertOne(ctx, p)
	if err != nil {
		return err
	}
	id, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return errors.New("unexpected id type from MongoDB")
	}
	p.ID = id
	return nil
}

func (repo *PropertyRepository) GetByID(ctx context.Context, id bson.ObjectID) (*models.Property, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var p models.Property
	err := repo.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrPropertyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List returns one page of active properties, newest first, plus the total
// number of active properties. Contact details are never loaded.
func (repo *PropertyRepository) List(ctx context.Context, limit, skip int64) ([]models.Property, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"status": models.StatusActive}

	total, err := repo.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		// _id as a tie-breaker keeps the order stable between pages.
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip).
		SetProjection(bson.M{"contact": 0}) // 0 means "leave this field out"

	cursor, err := repo.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	props := []models.Property{}
	if err := cursor.All(ctx, &props); err != nil {
		return nil, 0, err
	}
	return props, total, nil
}

// ListByOwner returns one page of ALL of an owner's properties, whatever
// their status, newest first. It includes contact details: they are the
// owner's own.
func (repo *PropertyRepository) ListByOwner(ctx context.Context, ownerID bson.ObjectID, limit, skip int64) ([]models.Property, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"owner_id": ownerID}

	total, err := repo.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := repo.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	props := []models.Property{}
	if err := cursor.All(ctx, &props); err != nil {
		return nil, 0, err
	}
	return props, total, nil
}

// Update replaces the editable details of a property. The filter matches
// BOTH the property id and its owner, so you can only update your own.
func (repo *PropertyRepository) Update(ctx context.Context, id, ownerID bson.ObjectID, req models.CreatePropertyRequest) (*models.Property, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"title":         req.Title,
		"description":   req.Description,
		"listing_type":  req.ListingType,
		"property_type": req.PropertyType,
		"price":         req.Price,
		"currency":      req.Currency,
		"location":      req.Location,
		"bedrooms":      req.Bedrooms,
		"bathrooms":     req.Bathrooms,
		"size_sqm":      req.SizeSqm,
		"amenities":     req.Amenities,
		"contact":       req.Contact,
		"updated_at":    time.Now().UTC(),
	}}
	return repo.findOneAndUpdate(ctx, bson.M{"_id": id, "owner_id": ownerID}, update)
}

// SetStatus changes only the status. Again, owner-scoped.
func (repo *PropertyRepository) SetStatus(ctx context.Context, id, ownerID bson.ObjectID, status string) (*models.Property, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}}
	return repo.findOneAndUpdate(ctx, bson.M{"_id": id, "owner_id": ownerID}, update)
}

// Delete removes a property. Owner-scoped: someone else's property counts
// as "not found".
func (repo *PropertyRepository) Delete(ctx context.Context, id, ownerID bson.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := repo.coll.DeleteOne(ctx, bson.M{"_id": id, "owner_id": ownerID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrPropertyNotFound
	}
	return nil
}

// findOneAndUpdate is shared by Update and SetStatus. It returns the
// document as it is AFTER the change.
func (repo *PropertyRepository) findOneAndUpdate(ctx context.Context, filter, update bson.M) (*models.Property, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var p models.Property
	err := repo.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrPropertyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
