package router

import (
	// http creates the route multiplexer and handler interfaces.
	"net/http"

	// handlers provides the endpoint methods registered below.
	"github.com/Salim-nuhu/CRUD/internal/handlers"
	// response formats the health endpoint's JSON result.
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// New registers application routes and returns their HTTP multiplexer.
func New(users *handlers.UserHandler, authH *handlers.AuthHandler) http.Handler {
	// Create a standard-library router for method-aware path patterns.
	mux := http.NewServeMux()

	// Register an unauthenticated endpoint for checking service health.
	mux.HandleFunc("GET /health", health)

	// Register the login endpoint that issues bearer tokens.
	mux.HandleFunc("POST /auth/login", authH.Login)

	// Register user creation with a JSON request body.
	mux.HandleFunc("POST /users", users.Create)
	// Register paginated user listing.
	mux.HandleFunc("GET /users", users.List)
	// Register lookup by the user's path ID.
	mux.HandleFunc("GET /users/{id}", users.GetByID)
	// Register replacement of the user's editable fields.
	mux.HandleFunc("PUT /users/{id}", users.Update)
	// Register deletion by the user's path ID.
	mux.HandleFunc("DELETE /users/{id}", users.Delete)

	// Return the configured multiplexer as the application's HTTP handler.
	return mux
}

// health reports that the HTTP process is available.
func health(w http.ResponseWriter, r *http.Request) {
	// Return a small JSON status payload with HTTP 200.
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
