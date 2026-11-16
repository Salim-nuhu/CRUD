package models

// Sort options for browsing properties.
const (
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
)

// PropertyFilter holds the search choices a buyer made. An empty text field
// or a zero number means "no filter on this".
type PropertyFilter struct {
	Query        string
	Region       string
	City         string
	ListingType  string
	PropertyType string
	Currency     string
	MinPrice     int64
	MaxPrice     int64
	MinBedrooms  int
	MinBathrooms int
	Sort         string
}
