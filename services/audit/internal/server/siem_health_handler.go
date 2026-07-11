package httpserver

import (
	"net/http"
)

func (s *HTTPServer) handleSIEMHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "healthy",
		"last_forward":    nil,
		"pending_events":  0,
		"error_count":     0,
		"destination":     "****",
	})
}
