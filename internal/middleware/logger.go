package middleware

import (
	"log"
	"net/http"
	"time"
)

// statusRecorder remembers the status code a handler writes, because
// http.ResponseWriter has no way to read it back.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap lets Go's newer http helpers reach the real ResponseWriter.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// If a handler never calls WriteHeader, Go sends 200 by default.
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("%s %s -> %d (%s)",
			r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	})
}
