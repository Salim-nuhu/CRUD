package middleware

import (
	"net/http"
	"strings"

	"github.com/Salim-nuhu/CRUD/internal/auth"
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	response.Error(w, http.StatusUnauthorized, message)
}

// RequireAuth rejects any request without a valid Bearer token. On success
// it stores the caller's identity (id and role) in the request context.
func RequireAuth(tokens *auth.TokenManager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				unauthorized(w, "missing or malformed Authorization header")
				return
			}

			identity, err := tokens.Parse(token)
			if err != nil {
				unauthorized(w, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
		})
	}
}

// OptionalAuth is for routes anyone may call, but that behave differently
// for logged-in users.
//   - no Authorization header: continue as an anonymous visitor
//   - valid token: continue with the identity in the context
//   - token present but bad or expired: 401, so a frontend knows to log in again
func OptionalAuth(tokens *auth.TokenManager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				next.ServeHTTP(w, r)
				return
			}

			token, ok := bearerToken(r)
			if !ok {
				unauthorized(w, "malformed Authorization header")
				return
			}

			identity, err := tokens.Parse(token)
			if err != nil {
				unauthorized(w, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
		})
	}
}
