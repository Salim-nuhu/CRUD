package handlers

import (
	// http supplies request and response types and status constants.
	"net/http"
	// strings normalizes user-provided email addresses.
	"strings"

	// auth verifies passwords and creates access tokens.
	"github.com/Salim-nuhu/CRUD/internal/auth"
	// models defines the login request structure.
	"github.com/Salim-nuhu/CRUD/internal/models"
	// repository fetches persisted user records.
	"github.com/Salim-nuhu/CRUD/internal/repository"
	// response writes consistent JSON responses.
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// AuthHandler handles authentication endpoints and their dependencies.
type AuthHandler struct {
	// repo looks up user records and their stored password hashes.
	repo *repository.UserRepository
	// tokens generates signed tokens after successful authentication.
	tokens *auth.TokenManager
}

// NewAuthHandler constructs an authentication handler from its dependencies.
func NewAuthHandler(repo *repository.UserRepository, tokens *auth.TokenManager) *AuthHandler {
	// Retain the shared dependencies for subsequent login requests.
	return &AuthHandler{repo: repo, tokens: tokens}
}

// Login validates credentials and returns a bearer token when they match.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Allocate the request value that will receive the submitted JSON fields.
	var req models.LoginRequest
	// Decode and validate the shape of the JSON body before using it.
	if !decodeJSON(w, r, &req) {
		return
	}

	// Normalize email casing and surrounding whitespace for consistent lookup.
	email := strings.ToLower(strings.TrimSpace(req.Email))
	// Reject missing credentials before calling the database or password checker.
	if email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}
	// bcrypt only reads 72 bytes, so anything longer can't be a real password.
	if len(req.Password) > 72 {
		// Return the same unauthorized response used for all invalid credentials.
		invalidCredentials(w)
		return
	}

	// Look up the account using the request's cancellation and deadline context.
	user, err := h.repo.GetByEmail(r.Context(), email)
	if err != nil {
		// Make nonexistent users take a password-check-like amount of time.
		if err == repository.ErrNotFound {
			auth.BurnTime(req.Password)
			// Do not reveal whether the email address exists.
			invalidCredentials(w)
			return
		}
		// Report unexpected repository failures through the shared error mapper.
		writeRepoError(w, "login", err)
		return
	}

	// Reject a password that does not match the stored bcrypt hash.
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		invalidCredentials(w)
		return
	}

	// Create a signed token whose subject identifies the authenticated user.
	token, err := h.tokens.Generate(user.ID.Hex())
	if err != nil {
		writeRepoError(w, "generate token", err) // logs it, replies 500
		return
	}

	// Return the token, its bearer scheme, and its lifetime in seconds.
	response.JSON(w, http.StatusOK, map[string]any{
		// token is the signed credential clients send on later requests.
		"token": token,
		// token_type tells clients to use the standard Bearer authorization scheme.
		"token_type": "Bearer",
		// expires_in lets clients know when they should request a new token.
		"expires_in": int(h.tokens.TTL().Seconds()),
	})
}

// One message for both "no such email" and "wrong password", so the
// response never reveals which emails have accounts.
func invalidCredentials(w http.ResponseWriter) {
	// Keep the response generic so callers cannot distinguish unknown emails
	// from incorrect passwords.
	response.Error(w, http.StatusUnauthorized, "invalid email or password")
}
