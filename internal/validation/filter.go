package validation

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Salim-nuhu/CRUD/internal/models"
)

const maxFilterText = 100

var sortOptions = []string{models.SortNewest, models.SortPriceAsc, models.SortPriceDesc}

// PropertyFilter reads and checks the search choices in a query string.
func PropertyFilter(params url.Values) (models.PropertyFilter, Errors) {
	errs := Errors{}

	f := models.PropertyFilter{
		Query:        strings.TrimSpace(params.Get("q")),
		Region:       strings.TrimSpace(params.Get("region")),
		City:         strings.TrimSpace(params.Get("city")),
		ListingType:  strings.ToLower(strings.TrimSpace(params.Get("listing_type"))),
		PropertyType: strings.ToLower(strings.TrimSpace(params.Get("property_type"))),
		Currency:     strings.ToUpper(strings.TrimSpace(params.Get("currency"))),
		Sort:         strings.ToLower(strings.TrimSpace(params.Get("sort"))),
	}

	// Text filters: cap the length.
	for name, value := range map[string]string{"q": f.Query, "region": f.Region, "city": f.City} {
		if utf8.RuneCountInString(value) > maxFilterText {
			errs[name] = name + " is too long"
		}
	}

	// Fixed-choice filters: must be one of the allowed values, if given.
	if f.ListingType != "" && !slices.Contains(listingTypes, f.ListingType) {
		errs["listing_type"] = "listing_type must be sale or rent"
	}
	if f.PropertyType != "" && !slices.Contains(propertyTypes, f.PropertyType) {
		errs["property_type"] = "property_type must be house, apartment, land or commercial"
	}
	if f.Currency != "" && !currencyRe.MatchString(f.Currency) {
		errs["currency"] = "currency must be a 3-letter code such as USD"
	}

	// Sorting: default to newest, otherwise it must be on the whitelist.
	if f.Sort == "" {
		f.Sort = models.SortNewest
	} else if !slices.Contains(sortOptions, f.Sort) {
		errs["sort"] = "sort must be newest, price_asc or price_desc"
	}

	// Numbers.
	f.MinPrice = nonNegativeInt(errs, "min_price", params.Get("min_price"), maxPrice)
	f.MaxPrice = nonNegativeInt(errs, "max_price", params.Get("max_price"), maxPrice)
	if f.MinPrice > 0 && f.MaxPrice > 0 && f.MinPrice > f.MaxPrice {
		errs["min_price"] = "min_price cannot be greater than max_price"
	}
	f.MinBedrooms = int(nonNegativeInt(errs, "min_bedrooms", params.Get("min_bedrooms"), 50))
	f.MinBathrooms = int(nonNegativeInt(errs, "min_bathrooms", params.Get("min_bathrooms"), 50))

	return f, errs
}

// nonNegativeInt reads a whole number between 0 and max. Empty means 0
// (no filter). Anything else records an error.
func nonNegativeInt(errs Errors, field, raw string, max int64) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 || n > max {
		errs[field] = fmt.Sprintf("%s must be a whole number between 0 and %d", field, max)
		return 0
	}
	return n
}
