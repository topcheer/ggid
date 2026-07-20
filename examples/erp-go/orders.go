package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "orders:read") { return }
		uid := currentUserID(r)
		showAll := false
		info := getUser(r.Context())
		if info != nil {
			for _, p := range info.Permissions {
				if p == "orders:read:all" || p == "admin" { showAll = true; break }
			}
		}
		list := []*Order{}
		for _, o := range orders {
			if showAll || o.CreatedBy == uid { list = append(list, o) }
		}
		writeJSON(w, 200, map[string]any{"items": list, "total": len(list)})

	case http.MethodPost:
		if !requirePerm(w, r, "orders:write") { return }
		var o Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil { writeError(w, 400, "invalid body"); return }
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

func handleOrderByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/orders/")
	parts := strings.Split(rest, "/")
	id := parts[0]

	if len(parts) >= 2 && parts[1] == "approve" && r.Method == http.MethodPut {
		if !requirePerm(w, r, "orders:approve") { return }
		o, ok := orders[id]
		if !ok { writeError(w, 404, "order not found"); return }
		o.Status = "approved"
		o.UpdatedAt = time.Now()
		addAudit("orders.approve", "order", "success", currentUserID(r))
		writeJSON(w, 200, o)
		return
	}

	o, ok := orders[id]
	if !ok { writeError(w, 404, "order not found"); return }

	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "orders:read") { return }
		writeJSON(w, 200, o)
	case http.MethodPut:
		if !requirePerm(w, r, "orders:write") { return }
		var upd Order
		if err := json.NewDecoder(r.Body).Decode(&upd); err != nil { writeError(w, 400, "invalid body"); return }
		o.Customer = upd.Customer; o.Quantity = upd.Quantity; o.Amount = upd.Amount
		o.UpdatedAt = time.Now()
		addAudit("orders.update", "order", "success", currentUserID(r))
		writeJSON(w, 200, o)
	case http.MethodDelete:
		if !requirePerm(w, r, "orders:write") { return }
		delete(orders, id)
		addAudit("orders.delete", "order", "success", currentUserID(r))
		writeJSON(w, 200, map[string]bool{"deleted": true})
	default:
		writeError(w, 405, "method not allowed")
	}
}
