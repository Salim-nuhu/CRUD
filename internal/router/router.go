package router

import (
	"net/http"

	"github.com/Salim-nuhu/CRUD/internal/handlers"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// New builds the route table and returns it as an http.Handler.
// Returning the interface (not *ServeMux) means that in Step 5 we can
// wrap the result in middleware without changing anything else.
func New(users *handlers.UserHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", health)

	mux.HandleFunc("POST /users", users.Create)
	mux.HandleFunc("GET /users", users.List)
	mux.HandleFunc("GET /users/{id}", users.GetByID)
	mux.HandleFunc("PUT /users/{id}", users.Update)
	mux.HandleFunc("DELETE /users/{id}", users.Delete)

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
