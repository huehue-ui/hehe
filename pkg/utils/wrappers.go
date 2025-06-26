package utils

import (
	"encoding/json"
	"net/http"
)

// MethodGuard is a middleware that ensures a handler only responds to a specific HTTP method.
// It also increments the RequestCount metric for every request it processes.
func MethodGuard(allowedMethod string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Increment the request counter metric with the path and method labels.
			RequestCount.WithLabelValues(r.URL.Path, r.Method).Inc()

			// Check if the request method matches the allowed method.
			if r.Method != allowedMethod {
				// If not, respond with a 405 Method Not Allowed error.
				ErrorJSON(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
				return
			}
			// If the method is allowed, pass the request to the next handler.
			next.ServeHTTP(w, r)
		})
	}
}

// WriteJSON is a helper function to write a JSON response.
// It sets the Content-Type header to "application/json", writes the HTTP status code,
// and then encodes the provided data `v` to JSON, sending it to the client.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encode the data directly to the response writer.
	// Error handling for json.NewEncoder(w).Encode(v) could be added here if needed.
	json.NewEncoder(w).Encode(v)
}

// ErrorJSON is a helper function to write a JSON error response.
// It uses WriteJSON to send a JSON object in the format: {"error": "message"}.
func ErrorJSON(w http.ResponseWriter, status int, msg string) {
	// Create a map to structure the error message in JSON.
	errorResponse := map[string]string{"error": msg}
	WriteJSON(w, status, errorResponse)
}
