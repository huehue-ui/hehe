package utils

import (
	"encoding/json"
	"net/http"
)

func MethodGuard(method string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			RequestCount.WithLabelValues(r.URL.Path, r.Method).Inc()

			if r.Method != method {
				ErrorJSON(w, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func ErrorJSON(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// ResponseWriterWrapper helps capture the status code for metrics
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	wroteHeader bool
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rww *ResponseWriterWrapper) WriteHeader(statusCode int) {
	if rww.wroteHeader {
		return
	}
	rww.statusCode = statusCode
	rww.ResponseWriter.WriteHeader(statusCode)
	rww.wroteHeader = true
}

// Write satisfies the http.ResponseWriter interface and ensures the status code is recorded
// if WriteHeader has not been called.
func (rww *ResponseWriterWrapper) Write(b []byte) (int, error) {
	if !rww.wroteHeader {
		// Default to 200 OK if WriteHeader is not called before Write
		rww.WriteHeader(http.StatusOK)
	}
	return rww.ResponseWriter.Write(b)
}

func (rww *ResponseWriterWrapper) StatusCode() int {
	return rww.statusCode
}
