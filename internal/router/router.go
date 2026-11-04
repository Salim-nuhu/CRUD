package router

import (
	"net/http"

	"github.com/Salim-nuhu/CRUD/internal/handlers"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

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
