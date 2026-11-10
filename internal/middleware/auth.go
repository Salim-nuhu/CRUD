package middleware

import (
	"net/http"
	"strings"

	"github.com/Salim-nuhu/CRUD/internal/auth"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// RequireAuth rejects any request without a valid Bearer token. On success
// it stores the caller's identity (id and role) in the request context.
func RequireAuth(tokens *auth.TokenManager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Expected header:  Authorization: Bearer <token>
			scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
			token = strings.TrimSpace(token)

			if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
				w.Header().Set("WWW-Authenticate", "Bearer")
				response.Error(w, http.StatusUnauthorized, "missing or malformed Authorization header")
				return
			}

			identity, err := tokens.Parse(token)
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := auth.WithIdentity(r.Context(), identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
