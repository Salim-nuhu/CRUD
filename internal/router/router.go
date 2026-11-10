package router

import (
	"net/http"

	"github.com/Salim-nuhu/CRUD/internal/auth"
	"github.com/Salim-nuhu/CRUD/internal/handlers"
	"github.com/Salim-nuhu/CRUD/internal/middleware"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

func New(
	users *handlers.UserHandler,
	authH *handlers.AuthHandler,
	protect middleware.Middleware,
	ownerOnly middleware.Middleware,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /auth/login", authH.Login)
	mux.HandleFunc("POST /users", users.Create) // registration

	// Protected routes: a valid token is required
	mux.Handle("GET /users", protect(http.HandlerFunc(users.List)))
	mux.Handle("GET /users/me", protect(http.HandlerFunc(users.Me)))
	mux.Handle("GET /users/{id}", protect(http.HandlerFunc(users.GetByID)))
	mux.Handle("PUT /users/{id}", protect(http.HandlerFunc(users.Update)))
	mux.Handle("DELETE /users/{id}", protect(http.HandlerFunc(users.Delete)))

	// TEMPORARY: delete after testing roles
	mux.Handle("GET /owner-check", protect(ownerOnly(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			id, _ := auth.IdentityFromContext(r.Context())
			response.JSON(w, http.StatusOK, map[string]string{
				"message": "welcome, owner",
				"role":    id.Role,
			})
		},
	))))

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
