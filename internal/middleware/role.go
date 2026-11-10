package middleware

import (
	"net/http"
	"slices"

	"github.com/Salim-nuhu/CRUD/internal/auth"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// RequireRole allows the request only if the caller's role is one of the
// allowed roles. It must run AFTER RequireAuth, because RequireAuth is
// what puts the identity into the context.
func RequireRole(allowed ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := auth.IdentityFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "not logged in")
				return
			}
			if !slices.Contains(allowed, id.Role) {
				response.Error(w, http.StatusForbidden, "you do not have permission to do this")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
