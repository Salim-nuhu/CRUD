package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Allowed values for the fixed-choice fields.
const (
	ListingSale = "sale"
	ListingRent = "rent"

	PropertyHouse      = "house"
	PropertyApartment  = "apartment"
	PropertyLand       = "land"
	PropertyCommercial = "commercial"

	StatusActive = "active" // visible to everyone
	StatusClosed = "closed" // sold or rented
	StatusHidden = "hidden" // taken down by the owner
)

type Location struct {
	Region  string `bson:"region"            json:"region"`
	City    string `bson:"city"              json:"city"`
	Address string `bson:"address,omitempty" json:"address,omitempty"`
}

type Contact struct {
	Phone    string `bson:"phone"              json:"phone"`
	Email    string `bson:"email,omitempty"    json:"email,omitempty"`
	WhatsApp string `bson:"whatsapp,omitempty" json:"whatsapp,omitempty"`
}

// Property is what we store in MongoDB and send to clients.
type Property struct {
	ID           bson.ObjectID `bson:"_id,omitempty"  json:"id"`
	OwnerID      bson.ObjectID `bson:"owner_id"       json:"owner_id"`
	Title        string        `bson:"title"          json:"title"`
	Description  string        `bson:"description"    json:"description"`
	ListingType  string        `bson:"listing_type"   json:"listing_type"`
	PropertyType string        `bson:"property_type"  json:"property_type"`
	Price        int64         `bson:"price"          json:"price"` // whole currency units; per month for rentals
	Currency     string        `bson:"currency"       json:"currency"`
	Location     Location      `bson:"location"       json:"location"`
	Bedrooms     int           `bson:"bedrooms"       json:"bedrooms"`
	Bathrooms    int           `bson:"bathrooms"      json:"bathrooms"`
	SizeSqm      int           `bson:"size_sqm"       json:"size_sqm"`
	Amenities    []string      `bson:"amenities"      json:"amenities"`
	Images       []string      `bson:"images"         json:"images"`
	// Contact is a pointer so it can be nil. A nil contact is left out of the
	// JSON entirely, which is how we hide it from people who are not logged in.
	Contact   *Contact  `bson:"contact"    json:"contact,omitempty"`
	Status    string    `bson:"status"     json:"status"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// CreatePropertyRequest is what an owner may send when posting a property.
// No owner_id, status or images: the server decides those.
type CreatePropertyRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ListingType  string   `json:"listing_type"`
	PropertyType string   `json:"property_type"`
	Price        int64    `json:"price"`
	Currency     string   `json:"currency"`
	Location     Location `json:"location"`
	Bedrooms     int      `json:"bedrooms"`
	Bathrooms    int      `json:"bathrooms"`
	SizeSqm      int      `json:"size_sqm"`
	Amenities    []string `json:"amenities"`
	Contact      Contact  `json:"contact"`
}

// UpdateStatusRequest is what an owner sends to change a listing's status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}
