package middleware

import (
	// log records each completed request.
	"log"
	// http provides handlers, requests, and response writer interfaces.
	"net/http"
	// time measures request duration.
	"time"
)

// statusRecorder remembers the status code a handler writes, because
// http.ResponseWriter has no way to read it back.
type statusRecorder struct {
	// Embedding preserves the regular response-writer methods for wrapped handlers.
	http.ResponseWriter
	// status stores the most recently written HTTP response code.
	status int
}

// WriteHeader remembers the status and forwards it to the original writer.
func (r *statusRecorder) WriteHeader(code int) {
	// Save the code so the logger can report the response later.
	r.status = code
	// Preserve normal HTTP behavior by writing the same status downstream.
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap lets Go's newer http helpers reach the real ResponseWriter.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	// Expose the wrapped writer to helpers that support unwrapping.
	return r.ResponseWriter
}

// Logger logs the method, URL path, status, and elapsed time for each request.
func Logger(next http.Handler) http.Handler {
	// Adapt the middleware logic to the standard HTTP handler interface.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record the time at which this request begins.
		start := time.Now()

		// If a handler never calls WriteHeader, Go sends 200 by default.
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		// Run the rest of the handler chain with the status-recording writer.
		next.ServeHTTP(rec, r)

		// Log the completed request and round duration to microsecond precision.
		log.Printf("%s %s -> %d (%s)",
			r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	})
}
