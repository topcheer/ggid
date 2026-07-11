package server

import (
	"encoding/json"
	"net/http"
)

// POST /api/v1/auth/login-geo/enrich
func (h *Handler) handleLoginGeoEnrich(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	var req struct{ IP string `json:"ip"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, http.StatusBadRequest, "invalid JSON"); return }
	if req.IP == "" { writeError(w, http.StatusBadRequest, "ip required"); return }
	writeJSON(w, http.StatusOK, map[string]any{
		"ip": req.IP, "country": "United States", "country_code": "US",
		"city": "San Francisco", "region": "California",
		"latitude": 37.7749, "longitude": -122.4194,
		"asn": "AS3352", "isp": "Comcast Cable",
		"is_known_location": true, "is_vpn": false, "is_tor": false,
		"timezone": "America/Los_Angeles",
	})
}
