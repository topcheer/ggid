package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

// FAPIConfigResponse represents the FAPI 2.0 configuration and enabled clients.
type FAPIConfigResponse struct {
	Enabled           bool                     `json:"enabled"`
	RequiredRules     []string                 `json:"required_rules"`
	EnabledClients    []FAPIEnabledClient      `json:"enabled_clients"`
	GlobalEnforcement bool                     `json:"global_enforcement"`
}

// FAPIEnabledClient describes a client with FAPI 2.0 enabled.
type FAPIEnabledClient struct {
	ClientID   string `json:"client_id"`
	Name       string `json:"name"`
	EnabledAt  string `json:"enabled_at"`
}

// FAPIConfigUpdateRequest toggles FAPI 2.0 for a specific client.
type FAPIConfigUpdateRequest struct {
	ClientID string `json:"client_id"`
	Enabled  bool   `json:"enabled"`
}

func handleFAPIConfig(oauthSvc *service.OAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := parseTenantContext(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleFAPIConfigGet(w, ctx, oauthSvc)
		case http.MethodPut:
			handleFAPIConfigPut(w, r, ctx, oauthSvc)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func handleFAPIConfigGet(w http.ResponseWriter, ctx context.Context, oauthSvc *service.OAuthService) {
	clients, _, err := oauthSvc.ListClients(ctx, 1000, 0)
	if err != nil {
		writeInternalError(w, "FAPIConfigGet", err)
		return
	}

	var enabledClients []FAPIEnabledClient
	for _, c := range clients {
		if c.FAPI2_0() {
			enabledClients = append(enabledClients, FAPIEnabledClient{
				ClientID:  c.ClientID,
				Name:      c.Name,
				EnabledAt: c.UpdatedAt.Format(time.RFC3339),
			})
		}
	}

	resp := FAPIConfigResponse{
		Enabled:        true,
		RequiredRules:  []string{
			"PKCE_S256",
			"PAR_REQUIRED",
			"DPOP_REQUIRED",
			"RESPONSE_TYPE_CODE_ONLY",
			"NO_IMPLICIT_PASSWORD_GRANTS",
		},
		EnabledClients: enabledClients,
		GlobalEnforcement: false,
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleFAPIConfigPut(w http.ResponseWriter, r *http.Request, ctx context.Context, oauthSvc *service.OAuthService) {
	var req FAPIConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "invalid JSON body"})
		return
	}
	if req.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "client_id is required"})
		return
	}

	client, err := oauthSvc.GetClient(ctx, req.ClientID)
	if err != nil || client == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "error_description": "client not found"})
		return
	}

	client.SetFAPI2_0(req.Enabled)
	updated, err := oauthSvc.UpdateClientMetadata(ctx, req.ClientID, &service.ClientMetadataUpdate{
		Metadata: client.Metadata,
	})
	if err != nil {
		writeInternalError(w, "FAPIConfigPut", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":  updated.ClientID,
		"name":       updated.Name,
		"fapi_2_0":   updated.FAPI2_0(),
		"updated_at": updated.UpdatedAt,
	})
}

// enforceFAPIAuthorize checks FAPI 2.0 requirements during authorization.
func enforceFAPIAuthorize(client *domain.OAuthClient, r *http.Request) error {
	if client == nil || !client.FAPI2_0() {
		return nil
	}

	responseType := r.URL.Query().Get("response_type")
	if responseType != "code" {
		return errors.New("FAPI 2.0 client requires response_type=code")
	}

	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
	if codeChallengeMethod != "S256" {
		return errors.New("FAPI 2.0 client requires PKCE S256")
	}

	requestURI := r.URL.Query().Get("request_uri")
	if requestURI == "" {
		return errors.New("FAPI 2.0 client requires Pushed Authorization Request (request_uri)")
	}

	if r.Header.Get("DPoP") == "" {
		return errors.New("FAPI 2.0 client requires DPoP proof header")
	}

	for _, gt := range client.GrantTypes {
		if gt == "implicit" || gt == "password" {
			return errors.New("FAPI 2.0 client disallows implicit and password grants")
		}
	}

	return nil
}

// enforceFAPIToken checks FAPI 2.0 requirements during token exchange.
func enforceFAPIToken(client *domain.OAuthClient, r *http.Request) error {
	if client == nil || !client.FAPI2_0() {
		return nil
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		return errors.New("FAPI 2.0 client only allows authorization_code grant")
	}

	if r.Header.Get("DPoP") == "" {
		return errors.New("FAPI 2.0 client requires DPoP proof header")
	}

	codeVerifier := r.FormValue("code_verifier")
	if codeVerifier == "" {
		return errors.New("FAPI 2.0 client requires PKCE code_verifier")
	}

	for _, gt := range client.GrantTypes {
		if gt == "implicit" || gt == "password" {
			return errors.New("FAPI 2.0 client disallows implicit and password grants")
		}
	}

	return nil
}

// parseTenantContext extracts the tenant from the request header or query param.
func parseTenantContext(r *http.Request) (context.Context, error) {
	ctx := r.Context()
	if tc, err := tenant.FromContext(ctx); err == nil && tc != nil {
		return ctx, nil
	}

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		tenantIDStr = r.URL.Query().Get("tenant_id")
	}
	if tenantIDStr == "" {
		return nil, fmt.Errorf("X-Tenant-ID header or tenant_id query param required")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("valid X-Tenant-ID header or tenant_id query param required")
	}

	return tenant.WithContext(ctx, &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	}), nil
}
