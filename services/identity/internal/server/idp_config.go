package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/identity/internal/idpconfig"
	"github.com/google/uuid"
)

// handleIdPConfig handles CRUD for per-tenant IdP configurations.
//
// Routes:
//
//	GET    /api/v1/tenants/{id}/idp-config            — list configs
//	POST   /api/v1/tenants/{id}/idp-config            — create config
//	GET    /api/v1/tenants/{id}/idp-config/{configId} — get config
//	PUT    /api/v1/tenants/{id}/idp-config/{configId} — update config
//	DELETE /api/v1/tenants/{id}/idp-config/{configId} — delete config
func (h *HTTPHandler) handleIdPConfig(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	// parts: ["api", "v1", "tenants", "{tenantID}", "idp-config", ...]

	if len(parts) < 5 || parts[4] != "idp-config" {
		writeJSONError(w, http.StatusBadRequest, "invalid idp-config path")
		return
	}

	tenantID, err := uuid.Parse(parts[3])
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	// Check for configId in path (single-resource operations).
	var configID *uuid.UUID
	if len(parts) >= 6 && parts[5] != "" {
		id, err := uuid.Parse(parts[5])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid config id")
			return
		}
		configID = &id
	}

	ctx := r.Context()

	if configID == nil {
		// Collection operations.
		switch r.Method {
		case http.MethodGet:
			configs, err := h.idpConfigSvc.List(ctx, tenantID)
			if err != nil {
				slog.Error("idp_config list error", "err", err)
				writeJSONError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"configs": configs})

		case http.MethodPost:
			var req struct {
				IdPType    string `json:"idp_type"`
				Name       string `json:"name"`
				ConfigJSON string `json:"config_json"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			cfg, err := h.idpConfigSvc.Create(ctx, tenantID, idpconfig.IdPType(req.IdPType), req.Name, req.ConfigJSON)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, cfg)

		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// Single-resource operations.
	switch r.Method {
	case http.MethodGet:
		cfg, err := h.idpConfigSvc.Get(ctx, *configID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)

	case http.MethodPut:
		var req struct {
			Name       string `json:"name"`
			ConfigJSON string `json:"config_json"`
			Enabled    bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		cfg, err := h.idpConfigSvc.Update(ctx, *configID, req.Name, req.ConfigJSON, req.Enabled)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)

	case http.MethodDelete:
		if err := h.idpConfigSvc.Delete(ctx, *configID); err != nil {
			slog.Error("idp_config delete error", "err", err)
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
