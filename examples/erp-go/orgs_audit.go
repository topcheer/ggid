package main

import (
	"encoding/json"
	"net/http"

	ggid "github.com/ggid/ggid/sdk/go"
)

// handleOrgs — GET (orgs:read) / POST (orgs:write) via GGID SDK
func handleOrgs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "orgs:read") {
			return
		}
		result, err := ggidClient.ListOrgs(r.Context(), &ggid.ListOptions{PageSize: 50})
		if err != nil {
			writeJSON(w, 200, map[string]any{"items": []any{}, "error": err.Error()})
			return
		}
		writeJSON(w, 200, result)

	case http.MethodPost:
		if !requirePerm(w, r, "orgs:write") {
			return
		}
		var req ggid.CreateOrgRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		org, err := ggidClient.CreateOrg(r.Context(), &req)
		if err != nil {
			writeError(w, 500, "create org failed: "+err.Error())
			return
		}
		addAudit("orgs.create", "org", "success", currentUserID(r))
		writeJSON(w, 201, org)

	default:
		writeError(w, 405, "method not allowed")
	}
}

// handleAudit — GET (audit:read) returns local audit log
func handleAudit(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	if !requirePerm(w, r, "audit:read") {
		return
	}
	writeJSON(w, 200, map[string]any{
		"items": auditLog,
		"total": len(auditLog),
	})
}

// handleDashboard — GET (dashboard:read) returns summary metrics
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet) {
		return
	}
	if !requirePerm(w, r, "dashboard:read") {
		return
	}
	pendingOrders := 0
	approvedOrders := 0
	for _, o := range orders {
		if o.Status == "pending" {
			pendingOrders++
		} else if o.Status == "approved" {
			approvedOrders++
		}
	}
	writeJSON(w, 200, map[string]any{
		"products":      len(products),
		"orders":        len(orders),
		"pending":       pendingOrders,
		"approved":      approvedOrders,
		"audit_entries": len(auditLog),
	})
}
