package httputil

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
)

// WriteJSON encodes v as JSON and writes it with the given status code.
// Shared implementation to eliminate 18+ duplicate copies across services.
func WriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a standard JSON error response.
func WriteError(w http.ResponseWriter, code int, msg string) {
	WriteJSON(w, code, map[string]string{"error": msg})
}

// WriteJSONError is an alias for WriteError (some services used this name).
func WriteJSONError(w http.ResponseWriter, code int, msg string) {
	WriteJSON(w, code, map[string]string{"error": msg})
}

// GetEnv returns the env var value or the default if not set.
func GetEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GetEnvInt returns the env var as int or the default if not set/invalid.
func GetEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
