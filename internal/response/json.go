package response

import (
	"encoding/json"
	"net/http"
)

// JSON writes data using the standard JSON format and the supplied HTTP status.
// This keeps every endpoint's response headers and encoding consistent.
func JSON(w http.ResponseWriter, status int, data any) {
	// Tell browsers and API tools how to interpret the response body.
	w.Header().Set("Content-Type", "application/json")
	// Write the status before the body because the first body write sends headers.
	w.WriteHeader(status)
	// Convert Go values (such as structs and maps) into JSON text.
	json.NewEncoder(w).Encode(data)
}

// Error gives clients a consistent JSON shape for an error message.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
