package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Salim-nuhu/CRUD/internal/models"
	"github.com/Salim-nuhu/CRUD/internal/repository"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// UserHandler translates HTTP requests into repository operations.
// It is the counter between a web customer and the storage room.
type UserHandler struct {
	repo *repository.UserRepository
}

// NewUserHandler builds a handler with the repository it will use.
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// Create accepts a JSON request, checks it, stores the user, and returns JSON.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Cap the request body at 1 MB to avoid using excessive memory on huge input.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Decode the incoming JSON into the small set of fields the client may send.
	var req models.CreateUserRequest
	dec := json.NewDecoder(r.Body)
	// Reject misspelled or unexpected fields instead of silently ignoring them.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Normalize user input so spaces and email letter case do not create
	// accidental differences (for example, "Sam@Example.com" vs "sam@example.com").
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Required-field checks stop incomplete records from reaching the database.
	if req.Name == "" || req.Email == "" {
		response.Error(w, http.StatusBadRequest, "name and email are required")
		return
	}
	// This is a simple format check, not a complete proof that the email exists.
	if !strings.Contains(req.Email, "@") {
		response.Error(w, http.StatusBadRequest, "email is not valid")
		return
	}

	// Store timestamps in UTC so dates stay consistent across time zones.
	now := time.Now().UTC()
	user := models.User{
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Ask the repository to save the user; it is responsible for MongoDB details.
	if err := h.repo.Create(r.Context(), &user); err != nil {
		// A duplicate email is a client conflict, not a general server failure.
		if errors.Is(err, repository.ErrEmailTaken) {
			response.Error(w, http.StatusConflict, "email already in use")
			return
		}
		// Keep detailed diagnostics in the server log, but give clients a generic
		// message so internal database details are not exposed.
		log.Printf("create user: %v", err)
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	// 201 means a new resource was successfully created.
	response.JSON(w, http.StatusCreated, user)
}
