package middleware

import "net/http"

// Middleware wraps a handler and returns a new one.
type Middleware func(http.Handler) http.Handler

// Chain wraps h in the given middleware. The FIRST one listed is the
// OUTERMOST, so it sees the request first and the response last.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	// Apply in reverse so mws[0] ends up on the outside.
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
