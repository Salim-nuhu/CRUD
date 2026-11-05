package middleware

import (
	// log records panic values and their stack traces for operators.
	"log"
	// http provides the handler interfaces and response status codes.
	"net/http"
	// debug captures the stack trace associated with a panic.
	"runtime/debug"

	// response writes the generic JSON error sent to the client.
	"github.com/Salim-nuhu/CRUD/internal/response"
)

// Recover catches unexpected handler panics and returns an internal error.
func Recover(next http.Handler) http.Handler {
	// Adapt panic recovery logic to a standard HTTP handler.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Run recovery after the wrapped handler returns or panics.
		defer func() {
			// recover returns the panic value, or nil when no panic occurred.
			if err := recover(); err != nil {
				// Go uses this special value to deliberately abort a
				// response. It is not a bug, so let it continue upward.
				if err == http.ErrAbortHandler {
					// Preserve the HTTP server's special abort behavior.
					panic(err)
				}
				// Log diagnostic details for developers and operations staff.
				log.Printf("panic: %v\n%s", err, debug.Stack())
				// Hide internal panic details behind a generic server response.
				response.Error(w, http.StatusInternalServerError, "something went wrong")
			}
		}()

		// Execute the next handler, with the deferred recovery guard in place.
		next.ServeHTTP(w, r)
	})
}
