package validation

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Salim-nuhu/CRUD/internal/models"
)

const (
	maxPrice     = 1_000_000_000_000
	maxAmenities = 20
)

var (
	listingTypes  = []string{models.ListingSale, models.ListingRent}
	propertyTypes = []string{
		models.PropertyHouse, models.PropertyApartment,
		models.PropertyLand, models.PropertyCommercial,
	}
	currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
)

// Property cleans and validates a new property. It returns the cleaned
// request plus any problems found (empty = valid).
func Property(req models.CreatePropertyRequest) (models.CreatePropertyRequest, Errors) {
	errs := Errors{}

	// Clean first, then check.
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.ListingType = strings.ToLower(strings.TrimSpace(req.ListingType))
	req.PropertyType = strings.ToLower(strings.TrimSpace(req.PropertyType))
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Location.Region = strings.TrimSpace(req.Location.Region)
	req.Location.City = strings.TrimSpace(req.Location.City)
	req.Location.Address = strings.TrimSpace(req.Location.Address)

	checkLength(errs, "title", req.Title, 5, 120)
	checkLength(errs, "description", req.Description, 20, 5000)

	if !slices.Contains(listingTypes, req.ListingType) {
		errs["listing_type"] = "listing_type must be sale or rent"
	}
	if !slices.Contains(propertyTypes, req.PropertyType) {
		errs["property_type"] = "property_type must be house, apartment, land or commercial"
	}

	switch {
	case req.Price <= 0:
		errs["price"] = "price must be greater than 0"
	case req.Price > maxPrice:
		errs["price"] = "price is too large"
	}
	if !currencyRe.MatchString(req.Currency) {
		errs["currency"] = "currency must be a 3-letter code such as USD"
	}

	checkLength(errs, "location.region", req.Location.Region, 2, 100)
	checkLength(errs, "location.city", req.Location.City, 2, 100)
	if utf8.RuneCountInString(req.Location.Address) > 200 {
		errs["location.address"] = "address must be at most 200 characters"
	}

	checkRange(errs, "bedrooms", req.Bedrooms, 0, 50)
	checkRange(errs, "bathrooms", req.Bathrooms, 0, 50)
	checkRange(errs, "size_sqm", req.SizeSqm, 0, 1_000_000)

	req.Amenities = cleanAmenities(errs, req.Amenities)

	// Contact details: the phone is required, the others are optional.
	phone, msg := Phone(req.Contact.Phone, true)
	if msg != "" {
		errs["contact.phone"] = msg
	}
	req.Contact.Phone = phone

	req.Contact.Email = strings.ToLower(strings.TrimSpace(req.Contact.Email))
	if req.Contact.Email != "" &&
		(len(req.Contact.Email) > maxEmailLen || !validEmail(req.Contact.Email)) {
		errs["contact.email"] = "email is not valid"
	}

	whatsapp, msg := Phone(req.Contact.WhatsApp, false)
	if msg != "" {
		errs["contact.whatsapp"] = msg
	}
	req.Contact.WhatsApp = whatsapp

	return req, errs
}

// checkLength requires a value between min and max characters.
func checkLength(errs Errors, field, value string, min, max int) {
	n := utf8.RuneCountInString(value)
	switch {
	case n == 0:
		errs[field] = field + " is required"
	case n < min:
		errs[field] = fmt.Sprintf("%s must be at least %d characters", field, min)
	case n > max:
		errs[field] = fmt.Sprintf("%s must be at most %d characters", field, max)
	}
}

func checkRange(errs Errors, field string, value, min, max int) {
	if value < min || value > max {
		errs[field] = fmt.Sprintf("%s must be between %d and %d", field, min, max)
	}
}

// cleanAmenities trims, lowercases and removes duplicates.
func cleanAmenities(errs Errors, in []string) []string {
	if len(in) > maxAmenities {
		errs["amenities"] = "at most 20 amenities are allowed"
		return nil
	}

	out := []string{} // not nil, so it shows as [] in JSON
	seen := map[string]bool{}
	for _, a := range in {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if n := utf8.RuneCountInString(a); n < 2 || n > 50 {
			errs["amenities"] = "each amenity must be 2 to 50 characters"
			return nil
		}
		if !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}

var statuses = []string{models.StatusActive, models.StatusClosed, models.StatusHidden}

// PropertyStatus checks that a status is one an owner may choose.
func PropertyStatus(status string) (string, string) {
	status = strings.ToLower(strings.TrimSpace(status))
	if !slices.Contains(statuses, status) {
		return "", "status must be active, closed or hidden"
	}
	return status, ""
}
