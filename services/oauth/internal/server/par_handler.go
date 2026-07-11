package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PAREntry stores a pushed authorization request.
type PAREntry struct {
	RequestURI  string         `json:"request_uri"`
	ClientID    string         `json:"client_id"`
	Params      map[string]any `json:"params"`
	CreatedAt   time.Time      `json:"created_at"`
	ExpiresAt   time.Time      `json:"expires_at"`
	Used        bool           `json:"used"`
}

var (
	parStoreMu sync.RWMutex
	parStore   = make(map[string]*PAREntry)
)

// POST /api/v1/oauth/par — push authorization request, return request_uri.
// GET /api/v1/oauth/par/{request_uri} — retrieve stored request.
func handlePAR(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			ClientID    string         `json:"client_id"`
			Params      map[string]any `json:"params"`
			ExpirySecs  int            `json:"expiry_seconds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
			return
		}
		if req.ClientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id is required"})
			return
		}
		if req.Params == nil {
			req.Params = map[string]any{}
		}
		if req.ExpirySecs <= 0 {
			req.ExpirySecs = 60 // RFC 9126: recommended 60s
		}

		now := time.Now().UTC()
		requestURI := "urn:ietf:params:oauth:request_uri:" + uuid.New().String()

		entry := &PAREntry{
			RequestURI: requestURI, ClientID: req.ClientID,
			Params: req.Params, CreatedAt: now,
			ExpiresAt: now.Add(time.Duration(req.ExpirySecs) * time.Second),
		}

		parStoreMu.Lock()
		parStore[requestURI] = entry
		parStoreMu.Unlock()

		writeJSON(w, http.StatusCreated, map[string]any{
			"request_uri": requestURI,
			"expires_in":  req.ExpirySecs,
		})
		return
	}

	if r.Method == http.MethodGet {
		// Extract request_uri from path
		uri := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/par/")
		if uri == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request_uri is required"})
			return
		}

		parStoreMu.RLock()
		entry, ok := parStore[uri]
		parStoreMu.RUnlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "request_uri not found"})
			return
		}

		if time.Now().UTC().After(entry.ExpiresAt) {
			writeJSON(w, http.StatusGone, map[string]any{"error": "request_uri expired"})
			return
		}

		writeJSON(w, http.StatusOK, entry)
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
}
