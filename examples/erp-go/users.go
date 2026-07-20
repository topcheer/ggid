package main

import (
	"encoding/json"
	"net/http"

	ggid "github.com/ggid/ggid/sdk/go"
)

// handleUsers — GET (list, users:read) / POST (create, users:write) via GGID SDK
func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "users:read") {
			return
		}
		// Use GGID SDK to list users
		result, err := ggidClient.ListUsers(r.Context(), &ggid.ListOptions{PageSize: 50})
		if err != nil {
			writeJSON(w, 200, map[string]any{"items": []any{}, "error": err.Error()})
			return
		}
		writeJSON(w, 200, result)

	case http.MethodPost:
		if !requirePerm(w, r, "users:write") {
			return
		}
		var req ggid.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		user, err := ggidClient.CreateUser(r.Context(), &req)
		if err != nil {
			writeError(w, 500, "create user failed: "+err.Error())
			return
		}
		addAudit("users.create", "user", "success", currentUserID(r))
		writeJSON(w, 201, user)

	default:
		writeError(w, 405, "method not allowed")
	}
}

// handleUserByID — GET/PUT/DELETE /api/users/:id
func handleUserByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "users:read") {
			return
		}
		user, err := ggidClient.GetUser(r.Context(), id)
		if err != nil {
			writeError(w, 404, "user not found: "+err.Error())
			return
		}
		writeJSON(w, 200, user)

	case http.MethodPut:
		if !requirePerm(w, r, "users:write") {
			return
		}
		var req ggid.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		user, err := ggidClient.UpdateUser(r.Context(), id, &req)
		if err != nil {
			writeError(w, 500, "update failed: "+err.Error())
			return
		}
		addAudit("users.update", "user", "success", currentUserID(r))
		writeJSON(w, 200, user)

	case http.MethodDelete:
		if !requirePerm(w, r, "users:delete") {
			return
		}
		if err := ggidClient.DeleteUser(r.Context(), id); err != nil {
			writeError(w, 500, "delete failed: "+err.Error())
			return
		}
		addAudit("users.delete", "user", "success", currentUserID(r))
		writeJSON(w, 200, map[string]bool{"deleted": true})

	default:
		writeError(w, 405, "method not allowed")
	}
}

// handleRoles — GET (roles:read) / POST (roles:write)
func handleRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "roles:read") {
			return
		}
		result, err := ggidClient.ListRoles(r.Context(), &ggid.ListOptions{PageSize: 50})
		if err != nil {
			writeJSON(w, 200, map[string]any{"items": []any{}, "error": err.Error()})
			return
		}
		writeJSON(w, 200, result)

	case http.MethodPost:
		if !requirePerm(w, r, "roles:write") {
			return
		}
		var req ggid.CreateRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		role, err := ggidClient.CreateRole(r.Context(), &req)
		if err != nil {
			writeError(w, 500, "create role failed: "+err.Error())
			return
		}
		addAudit("roles.create", "role", "success", currentUserID(r))
		writeJSON(w, 201, role)

	default:
		writeError(w, 405, "method not allowed")
	}
}
