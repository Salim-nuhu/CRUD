package repository

import (
	"context"
	"errors"
	"regexp"
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

	// Indexes make common queries fast, like the index at the back of a book.
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		// The public list: active properties, newest first.
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
		// Browsing by price.
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "price", Value: 1}}},
		// "Show me this owner's properties".
		{Keys: bson.D{{Key: "owner_id", Value: 1}}},
		// Keyword search over titles and descriptions (one text index per collection).
		{Keys: bson.D{{Key: "title", Value: "text"}, {Key: "description", Value: "text"}}},
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

// List returns one page of active properties that match the filter, plus the
// total number of matches. Contact details are never loaded.
func (repo *PropertyRepository) List(ctx context.Context, f models.PropertyFilter, limit, skip int64) ([]models.Property, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := buildFilter(f)

	total, err := repo.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSort(sortFor(f.Sort)).
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

// buildFilter turns the buyer's choices into a MongoDB query. Only the
// choices that were given are added. Hidden and closed properties are
// excluded no matter what.
func buildFilter(f models.PropertyFilter) bson.M {
	filter := bson.M{"status": models.StatusActive}

	if f.Query != "" {
		filter["$text"] = bson.M{"$search": f.Query}
	}
	if f.Region != "" {
		filter["location.region"] = exactInsensitive(f.Region)
	}
	if f.City != "" {
		filter["location.city"] = exactInsensitive(f.City)
	}
	if f.ListingType != "" {
		filter["listing_type"] = f.ListingType
	}
	if f.PropertyType != "" {
		filter["property_type"] = f.PropertyType
	}
	if f.Currency != "" {
		filter["currency"] = f.Currency
	}

	if f.MinPrice > 0 || f.MaxPrice > 0 {
		price := bson.M{}
		if f.MinPrice > 0 {
			price["$gte"] = f.MinPrice // greater than or equal
		}
		if f.MaxPrice > 0 {
			price["$lte"] = f.MaxPrice // less than or equal
		}
		filter["price"] = price
	}
	if f.MinBedrooms > 0 {
		filter["bedrooms"] = bson.M{"$gte": f.MinBedrooms}
	}
	if f.MinBathrooms > 0 {
		filter["bathrooms"] = bson.M{"$gte": f.MinBathrooms}
	}
	return filter
}

// exactInsensitive matches a whole value ignoring upper/lower case. The
// text is escaped first, so characters like "." and "*" mean themselves
// and cannot act as pattern commands.
func exactInsensitive(s string) bson.Regex {
	return bson.Regex{Pattern: "^" + regexp.QuoteMeta(s) + "$", Options: "i"}
}

// sortFor maps our sort words to real sort orders. The user's text never
// goes into the query. _id is the tie-breaker that keeps pages stable.
func sortFor(sort string) bson.D {
	switch sort {
	case models.SortPriceAsc:
		return bson.D{{Key: "price", Value: 1}, {Key: "_id", Value: 1}}
	case models.SortPriceDesc:
		return bson.D{{Key: "price", Value: -1}, {Key: "_id", Value: -1}}
	default:
		return bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}
	}
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
