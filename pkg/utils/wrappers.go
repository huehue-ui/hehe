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
