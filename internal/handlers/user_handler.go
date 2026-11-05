package handlers

import (
	// json decodes incoming request bodies.
	"encoding/json"
	// errors classifies decoding and repository errors.
	"errors"
	// auth hashes new passwords before persistence.
	"github.com/Salim-nuhu/CRUD/internal/auth"
	// models defines stored users and request payload structures.
	"github.com/Salim-nuhu/CRUD/internal/models"
	// repository performs user persistence operations.
	"github.com/Salim-nuhu/CRUD/internal/repository"
	// response writes consistent JSON API responses.
	"github.com/Salim-nuhu/CRUD/internal/response"
	// validation normalizes input and reports field-level problems.
	"github.com/Salim-nuhu/CRUD/internal/validation"
	// bson parses and represents MongoDB identifiers.
	"go.mongodb.org/mongo-driver/v2/bson"
	// io provides the end-of-input marker used during JSON validation.
	"io"
	// log records internal errors without exposing sensitive values.
	"log"
	// http defines handlers, requests, responses, and status codes.
	"net/http"
	// strconv parses numeric paging query parameters.
	"strconv"
	// time stamps newly created user records.
	"time"
)

// UserHandler serves HTTP endpoints that create and manage user records.
type UserHandler struct {
	// repo provides the database operations used by these endpoints.
	repo *repository.UserRepository
}

// NewUserHandler binds a user repository to a new handler.
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	// Retain the repository for each incoming user request.
	return &UserHandler{repo: repo}
}

// ---------- handlers ----------

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Prepare a value to receive the new user's JSON fields.
	var req models.CreateUserRequest
	// Stop if the request body cannot be decoded safely.
	if !decodeJSON(w, r, &req) {
		return
	}

	// Normalize and validate the shared name and email fields.
	name, email, errs := validation.User(req.Name, req.Email)
	// Validate the password separately because it is not part of user updates.
	if msg := validation.Password(req.Password); msg != "" {
		// Attach any password problem to the same field-error map.
		errs["password"] = msg
	}
	// Return all detected field errors together instead of partially creating a user.
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	// Hash only after validation passes: hashing is deliberately slow.
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("hash password: %v", err) // never log the password itself
		response.Error(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	// Use a single UTC time for both creation and last-update timestamps.
	now := time.Now().UTC()
	// Build the database record with normalized fields and the password hash.
	user := models.User{
		// Store the validated display name.
		Name: name,
		// Store the normalized email address.
		Email: email,
		// Persist only the one-way hash rather than the submitted password.
		PasswordHash: hash,
		// Record when this user was first created.
		CreatedAt: now,
		// Initially, the creation and update timestamps are identical.
		UpdatedAt: now,
	}

	// Insert the user using the request context and handle database failures.
	if err := h.repo.Create(r.Context(), &user); err != nil {
		writeRepoError(w, "create user", err)
		return
	}
	// Return the created user, including its assigned ID, with 201 Created.
	response.JSON(w, http.StatusCreated, user)
}

// GetByID returns the user identified by the URL's {id} parameter.
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the path parameter before accessing storage.
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	// Fetch the matching user using the request's context.
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		// Convert expected and unexpected repository outcomes to HTTP responses.
		writeRepoError(w, "get user", err)
		return
	}
	// Return the found user as JSON.
	response.JSON(w, http.StatusOK, user)
}

// List returns a bounded page of users.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	// Default the page to one and cap it to a reasonable maximum.
	page := queryInt(r, "page", 1, 100000)
	limit := queryInt(r, "limit", 20, 100) // default 20, never more than 100
	// Convert the one-based page number into a zero-based record offset.
	skip := (page - 1) * limit

	// Load only the requested number of users after the computed offset.
	users, err := h.repo.List(r.Context(), int64(limit), int64(skip))
	if err != nil {
		// Map database errors to an appropriate client response.
		writeRepoError(w, "list users", err)
		return
	}

	// Include pagination metadata with the returned user records.
	response.JSON(w, http.StatusOK, map[string]any{
		// data is the current page of users.
		"data": users,
		// page identifies the one-based page returned.
		"page": page,
		// limit identifies the maximum result count for this page.
		"limit": limit,
	})
}

// Update validates and saves the name and email of an existing user.
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Resolve the target user ID from the URL before reading its update payload.
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	// Prepare a value to receive the update JSON fields.
	var req models.UpdateUserRequest
	// Stop if the body is malformed, oversized, or contains extra values.
	if !decodeJSON(w, r, &req) {
		return
	}

	// Normalize and validate the replacement name and email.
	name, email, errs := validation.User(req.Name, req.Email)
	// Reject the update if either field has a validation problem.
	if len(errs) > 0 {
		writeValidationError(w, errs)
		return
	}

	// Persist the validated fields and retrieve the updated user document.
	user, err := h.repo.Update(r.Context(), id, name, email)
	if err != nil {
		// Convert repository outcomes into client-facing status codes.
		writeRepoError(w, "update user", err)
		return
	}
	// Return the updated user as JSON.
	response.JSON(w, http.StatusOK, user)
}

// Delete removes the user identified by the URL's {id} parameter.
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the target ID before accessing storage.
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	// Delete the record and map missing-user or database errors to HTTP responses.
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepoError(w, "delete user", err)
		return
	}
	// A successful deletion has no response body.
	w.WriteHeader(http.StatusNoContent)
}

// ---------- helpers ----------

// decodeJSON reads the request body into dst. On failure it writes the
// error response itself and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	// Limit untrusted request bodies to one megabyte before decoding.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB cap

	// Create a decoder that can stream the body into the supplied destination.
	dec := json.NewDecoder(r.Body)
	// Reject fields that are not declared by the request model.
	dec.DisallowUnknownFields()

	// Decode the first JSON value into the request structure.
	if err := dec.Decode(dst); err != nil {
		// Prepare to recognize the maximum-size error separately.
		var tooBig *http.MaxBytesError
		// Return a specific response for oversized, empty, or malformed bodies.
		switch {
		case errors.As(err, &tooBig):
			response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
		case errors.Is(err, io.EOF):
			response.Error(w, http.StatusBadRequest, "request body is empty")
		default:
			response.Error(w, http.StatusBadRequest, "invalid JSON body")
		}
		// Tell the caller not to continue processing this request.
		return false
	}

	// Reject bodies like {"name":"a"}{"name":"b"}: after the first JSON
	// value there must be nothing left.
	// Try decoding again; only end-of-input is acceptable after the first value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		// Reject trailing JSON values or other data after the request object.
		response.Error(w, http.StatusBadRequest, "body must contain a single JSON object")
		// Tell the caller the body did not have the required single-value shape.
		return false
	}
	// The body contains exactly one valid JSON value.
	return true
}

// writeValidationError returns field-specific input errors with HTTP 422.
func writeValidationError(w http.ResponseWriter, errs validation.Errors) {
	// Keep the response envelope consistent while including per-field messages.
	response.JSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error":  "validation failed",
		"fields": errs,
	})
}

// parseID reads {id} from the URL and converts it to a MongoDB ObjectID.
func parseID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	// Read the id path value and decode its hexadecimal MongoDB representation.
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	// Return a client error for missing or malformed identifiers.
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return bson.ObjectID{}, false
	}
	// Return the decoded ID and indicate successful parsing.
	return id, true
}

// queryInt reads an integer query parameter. A missing or invalid value
// gives def; a value above max is clamped down to max.
func queryInt(r *http.Request, key string, def, max int) int {
	// Parse the query parameter as a base-10 integer.
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	// Use the caller-provided default for malformed or non-positive values.
	if err != nil || n < 1 {
		return def
	}
	// Clamp valid values so clients cannot request excessive work.
	if n > max {
		return max
	}
	// Return the parsed value when it is within the allowed range.
	return n
}

// writeRepoError turns repository errors into the right HTTP response.
func writeRepoError(w http.ResponseWriter, op string, err error) {
	// Select client responses for known outcomes and log unknown server failures.
	switch {
	case errors.Is(err, repository.ErrNotFound):
		// Translate a missing user into HTTP 404.
		response.Error(w, http.StatusNotFound, "user not found")
	case errors.Is(err, repository.ErrEmailTaken):
		// Translate duplicate email conflicts into HTTP 409.
		response.Error(w, http.StatusConflict, "email already in use")
	default:
		// Log internal details for operators without sending them to API clients.
		log.Printf("%s: %v", op, err)
		// Return a generic HTTP 500 response to avoid leaking internals.
		response.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}
