package middleware

// http defines the standard handler type used by the middleware functions.
import "net/http"

// Middleware wraps a handler and returns a new one.
// Middleware adapts a handler by wrapping it with additional request behavior.
type Middleware func(http.Handler) http.Handler

// Chain wraps h in the given middleware. The FIRST one listed is the
// OUTERMOST, so it sees the request first and the response last.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	// Apply in reverse so mws[0] ends up on the outside.
	for i := len(mws) - 1; i >= 0; i-- {
		// Wrap the current handler in the next middleware from the inside out.
		h = mws[i](h)
	}
	// Return the fully wrapped handler tree.
	return h
}
