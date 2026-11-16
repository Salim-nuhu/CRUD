package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Salim-nuhu/CRUD/internal/auth"
	"github.com/Salim-nuhu/CRUD/internal/models"
	"github.com/Salim-nuhu/CRUD/internal/repository"
	"github.com/Salim-nuhu/CRUD/internal/response"
	"github.com/Salim-nuhu/CRUD/internal/validation"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PropertyHandler struct {
	repo *repository.PropertyRepository
}

func NewPropertyHandler(repo *repository.PropertyRepository) *PropertyHandler {
	return &PropertyHandler{repo: repo}
}

// ---------- public ----------

// List is public. It returns active properties that match the buyer's
// filters, without contact details.
func (h *PropertyHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, errs := validation.PropertyFilter(r.URL.Query())
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	page := queryInt(r, "page", 1, 100000)
	limit := queryInt(r, "limit", 20, 100)
	skip := (page - 1) * limit

	props, total, err := h.repo.List(r.Context(), filter, int64(limit), int64(skip))
	if err != nil {
		writePropertyError(w, "list properties", err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":  props,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// GetByID is public, behind OptionalAuth. Contact details are shown only to
// logged-in users.
func (h *PropertyHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePropertyID(w, r)
	if !ok {
		return
	}

	property, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writePropertyError(w, "get property", err)
		return
	}

	identity, loggedIn := auth.IdentityFromContext(r.Context())
	isOwner := loggedIn && identity.UserID == property.OwnerID.Hex()

	// Closed and hidden listings exist only for their owner.
	if property.Status != models.StatusActive && !isOwner {
		response.Error(w, http.StatusNotFound, "property not found")
		return
	}

	if !loggedIn {
		property.Contact = nil // a nil contact is omitted from the JSON
	}
	response.JSON(w, http.StatusOK, property)
}

// ---------- owner only (the router enforces the role) ----------

// Create posts a new property.
func (h *PropertyHandler) Create(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	var req models.CreatePropertyRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	clean, errs := validation.Property(req)
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	now := time.Now().UTC()
	property := models.Property{
		OwnerID:      ownerID,
		Title:        clean.Title,
		Description:  clean.Description,
		ListingType:  clean.ListingType,
		PropertyType: clean.PropertyType,
		Price:        clean.Price,
		Currency:     clean.Currency,
		Location:     clean.Location,
		Bedrooms:     clean.Bedrooms,
		Bathrooms:    clean.Bathrooms,
		SizeSqm:      clean.SizeSqm,
		Amenities:    clean.Amenities,
		Images:       []string{},
		Contact:      &clean.Contact,
		Status:       models.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.repo.Create(r.Context(), &property); err != nil {
		writePropertyError(w, "create property", err)
		return
	}
	response.JSON(w, http.StatusCreated, property)
}

// Mine lists all of the caller's own properties, whatever their status.
func (h *PropertyHandler) Mine(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	page := queryInt(r, "page", 1, 100000)
	limit := queryInt(r, "limit", 20, 100)
	skip := (page - 1) * limit

	props, total, err := h.repo.ListByOwner(r.Context(), ownerID, int64(limit), int64(skip))
	if err != nil {
		writePropertyError(w, "list my properties", err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":  props,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// Update edits the details of one of the caller's properties.
func (h *PropertyHandler) Update(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := parsePropertyID(w, r)
	if !ok {
		return
	}

	var req models.CreatePropertyRequest // editing uses the same form as posting
	if !decodeJSON(w, r, &req) {
		return
	}

	clean, errs := validation.Property(req)
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	property, err := h.repo.Update(r.Context(), id, ownerID, clean)
	if err != nil {
		writePropertyError(w, "update property", err)
		return
	}
	response.JSON(w, http.StatusOK, property)
}

// UpdateStatus sets a property to active, closed or hidden.
func (h *PropertyHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := parsePropertyID(w, r)
	if !ok {
		return
	}

	var req models.UpdateStatusRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	status, msg := validation.PropertyStatus(req.Status)
	if msg != "" {
		writeValidationError(w, validation.Errors{"status": msg})
		return
	}

	property, err := h.repo.SetStatus(r.Context(), id, ownerID, status)
	if err != nil {
		writePropertyError(w, "update property status", err)
		return
	}
	response.JSON(w, http.StatusOK, property)
}

// Delete permanently removes one of the caller's properties.
func (h *PropertyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	id, ok := parsePropertyID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id, ownerID); err != nil {
		writePropertyError(w, "delete property", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- helpers ----------

// currentUserID returns the logged-in user's id, taken from the token. On
// failure it writes the error response itself and returns false.
func currentUserID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "not logged in")
		return bson.ObjectID{}, false
	}
	id, err := bson.ObjectIDFromHex(identity.UserID)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid token")
		return bson.ObjectID{}, false
	}
	return id, true
}

func parsePropertyID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid property id")
		return bson.ObjectID{}, false
	}
	return id, true
}

func writePropertyError(w http.ResponseWriter, op string, err error) {
	if errors.Is(err, repository.ErrPropertyNotFound) {
		response.Error(w, http.StatusNotFound, "property not found")
		return
	}
	log.Printf("%s: %v", op, err)
	response.Error(w, http.StatusInternalServerError, "something went wrong")
}
