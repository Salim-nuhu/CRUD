package router

import (
	"net/http"

	"github.com/Salim-nuhu/CRUD/internal/handlers"
	"github.com/Salim-nuhu/CRUD/internal/middleware"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

func New(
	users *handlers.UserHandler,
	authH *handlers.AuthHandler,
	props *handlers.PropertyHandler,
	protect middleware.Middleware,
	optional middleware.Middleware,
	ownerOnly middleware.Middleware,
) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /auth/login", authH.Login)
	mux.HandleFunc("POST /users", users.Create) // registration
	mux.HandleFunc("GET /properties", props.List)

	// Public, but behaves differently when logged in (shows contact details)
	mux.Handle("GET /properties/{id}", optional(http.HandlerFunc(props.GetByID)))

	// Logged in
	mux.Handle("GET /users", protect(http.HandlerFunc(users.List)))
	mux.Handle("GET /users/me", protect(http.HandlerFunc(users.Me)))
	mux.Handle("GET /users/{id}", protect(http.HandlerFunc(users.GetByID)))
	mux.Handle("PUT /users/{id}", protect(http.HandlerFunc(users.Update)))
	mux.Handle("DELETE /users/{id}", protect(http.HandlerFunc(users.Delete)))

	// Logged in AND an owner: protect runs first, then ownerOnly.
	// "/properties/mine" is more specific than "/properties/{id}", so Go
	// picks it for that exact path.
	mux.Handle("GET /properties/mine", protect(ownerOnly(http.HandlerFunc(props.Mine))))
	mux.Handle("POST /properties", protect(ownerOnly(http.HandlerFunc(props.Create))))
	mux.Handle("PUT /properties/{id}", protect(ownerOnly(http.HandlerFunc(props.Update))))
	mux.Handle("PATCH /properties/{id}/status", protect(ownerOnly(http.HandlerFunc(props.UpdateStatus))))
	mux.Handle("DELETE /properties/{id}", protect(ownerOnly(http.HandlerFunc(props.Delete))))

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
