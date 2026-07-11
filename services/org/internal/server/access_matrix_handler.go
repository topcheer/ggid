package httpserver

import (
	"net/http"
	"strings"
)

// GET /api/v1/organizations/{id}/access-matrix
func (s *HTTPServer) handleAccessMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed"); return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	orgID := ""; if len(parts) >= 4 { orgID = parts[3] }
	depts := []string{"Engineering", "Sales", "Marketing", "Operations"}
	roles := []string{"admin", "manager", "developer", "viewer"}
	grid := make([]map[string]any, len(depts))
	for i, d := range depts {
		cells := make(map[string]int)
		for _, role := range roles { cells[role] = (i + 1) * (len(role) % 5 + 2) }
		grid[i] = map[string]any{"department": d, "cells": cells}
	}
	writeJSON(w, http.StatusOK, map[string]any{"org_id": orgID, "departments": depts, "roles": roles, "grid": grid})
}
