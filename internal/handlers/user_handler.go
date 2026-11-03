package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Salim-nuhu/CRUD/internal/models"
	"github.com/Salim-nuhu/CRUD/internal/repository"
	"github.com/Salim-nuhu/CRUD/internal/response"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserHandler struct {
	repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// ---------- handlers ----------

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name, email, err := cleanUserInput(req.Name, req.Email)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	user := models.User{
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.repo.Create(r.Context(), &user); err != nil {
		writeRepoError(w, "create user", err)
		return
	}
	response.JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepoError(w, "get user", err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1, 1)
	limit := queryInt(r, "limit", 20, 100) // default 20, never more than 100
	skip := (page - 1) * limit

	users, err := h.repo.List(r.Context(), int64(limit), int64(skip))
	if err != nil {
		writeRepoError(w, "list users", err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":  users,
		"page":  page,
		"limit": limit,
	})
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req models.UpdateUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	name, email, err := cleanUserInput(req.Name, req.Email)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.repo.Update(r.Context(), id, name, email)
	if err != nil {
		writeRepoError(w, "update user", err)
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepoError(w, "delete user", err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204: success, nothing to send back
}

// ---------- helpers ----------

// decodeJSON reads the request body into dst. On failure it writes the
// error response itself and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB cap

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// cleanUserInput trims and validates the fields shared by create and update.
func cleanUserInput(name, email string) (string, string, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if name == "" || email == "" {
		return "", "", errors.New("name and email are required")
	}
	if !strings.Contains(email, "@") {
		return "", "", errors.New("email is not valid")
	}
	return name, email, nil
}

// parseID reads {id} from the URL and converts it to a MongoDB ObjectID.
func parseID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return bson.ObjectID{}, false
	}
	return id, true
}

// queryInt reads an integer query parameter, falling back to def when it is
// missing or invalid, and clamping it to the range [1, max].
func queryInt(r *http.Request, key string, def, max int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || n < 1 {
		return def
	}
	if n > max && key == "limit" {
		return max
	}
	return n
}

// writeRepoError turns repository errors into the right HTTP response.
func writeRepoError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		response.Error(w, http.StatusNotFound, "user not found")
	case errors.Is(err, repository.ErrEmailTaken):
		response.Error(w, http.StatusConflict, "email already in use")
	default:
		log.Printf("%s: %v", op, err) // real error stays in our logs
		response.Error(w, http.StatusInternalServerError, "something went wrong")
	}
}
