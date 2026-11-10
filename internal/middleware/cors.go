package middleware

import (
	"net/http"
	"slices"
)

// CORS lets browsers on the allowed origins call the API, and answers
// preflight (OPTIONS) requests itself.
func CORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// The response now depends on who is asking, so tell caches.
			w.Header().Add("Vary", "Origin")

			if origin != "" && slices.Contains(allowedOrigins, origin) {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				h.Set("Access-Control-Max-Age", "600") // browser may remember the answer for 10 minutes
			}

			// A preflight is an OPTIONS request asking for permission.
			// Answer it here: it must never reach the router.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
