package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	ggidmw "github.com/ggid/ggid/sdk/go/middleware"
)

// handleOrders — GET (list, requires orders:read) / POST (create, requires orders:write)
func handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "orders:read") {
			return
		}
		// If user has orders:read:all, show all orders; otherwise filter by created_by
		uid := currentUserID(r)
		showAll := false
		for _, p := range getPermissions(r) {
			if p == "orders:read:all" || p == "admin" {
				showAll = true
				break
			}
		}
		list := make([]*Order, 0, len(orders))
		for _, o := range orders {
			if showAll || o.CreatedBy == uid {
				list = append(list, o)
			}
		}
		writeJSON(w, 200, map[string]any{"items": list, "total": len(list), "filtered": !showAll})

	case http.MethodPost:
		if !requirePerm(w, r, "orders:write") {
			return
		}
		var o Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		o.ID = nextOrderID()
		o.Status = "pending"
		o.CreatedBy = currentUserID(r)
		o.CreatedAt = time.Now()
		o.UpdatedAt = time.Now()
		orders[o.ID] = &o
		addAudit("orders.create", "order", "success", o.CreatedBy)
		writeJSON(w, 201, o)

	default:
		writeError(w, 405, "method not allowed")
	}
}

// handleOrderByID — GET/PUT/DELETE /api/orders/:id + PUT /api/orders/:id/approve
func handleOrderByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)

	// Check for approve action
	if r.Method == http.MethodPut && len(r.URL.Path) > len("/api/orders/") {
		parts := splitPath(r.URL.Path)
		if len(parts) >= 2 && parts[1] == "approve" {
			if !requirePerm(w, r, "orders:approve") {
				return
			}
			o, ok := orders[parts[0]]
			if !ok {
				writeError(w, 404, "order not found")
				return
			}
			o.Status = "approved"
			o.UpdatedAt = time.Now()
			addAudit("orders.approve", "order", "success", currentUserID(r))
			writeJSON(w, 200, o)
			return
		}
	}

	o, ok := orders[id]
	if !ok {
		writeError(w, 404, "order not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "orders:read") {
			return
		}
		writeJSON(w, 200, o)

	case http.MethodPut:
		if !requirePerm(w, r, "orders:write") {
			return
		}
		var update Order
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		o.Customer = update.Customer
		o.Quantity = update.Quantity
		o.Amount = update.Amount
		o.UpdatedAt = time.Now()
		addAudit("orders.update", "order", "success", currentUserID(r))
		writeJSON(w, 200, o)

	case http.MethodDelete:
		if !requirePerm(w, r, "orders:write") {
			return
		}
		delete(orders, id)
		addAudit("orders.delete", "order", "success", currentUserID(r))
		writeJSON(w, 200, map[string]bool{"deleted": true})

	default:
		writeError(w, 405, "method not allowed")
	}
}

func getPermissions(r *http.Request) []string {
	info, ok := ggidmw.FromContext(r.Context())
	if !ok {
		return nil
	}
	return info.Permissions
}

// suppress unused import
var _ = ggidmw.NewMiddleware

func splitPath(path string) []string {
	rest := path[len("/api/orders/"):]
	return strings.Split(rest, "/")
}
