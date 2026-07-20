package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleInventory — GET (list, requires inventory:read) / POST (create, requires inventory:write)
func handleInventory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "inventory:read") {
			return
		}
		list := make([]*Product, 0, len(products))
		for _, p := range products {
			list = append(list, p)
		}
		writeJSON(w, 200, map[string]any{"items": list, "total": len(list)})

	case http.MethodPost:
		if !requirePerm(w, r, "inventory:write") {
			return
		}
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		p.ID = nextProductID()
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
		products[p.ID] = &p
		addAudit("inventory.create", "product", "success", currentUserID(r))
		writeJSON(w, 201, p)

	default:
		writeError(w, 405, "method not allowed")
	}
}

// handleInventoryByID — GET/PUT/DELETE /api/inventory/:id
func handleInventoryByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	p, ok := products[id]
	if !ok {
		writeError(w, 404, "product not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !requirePerm(w, r, "inventory:read") {
			return
		}
		writeJSON(w, 200, p)

	case http.MethodPut:
		if !requirePerm(w, r, "inventory:write") {
			return
		}
		var update Product
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeError(w, 400, "invalid body")
			return
		}
		p.Name = update.Name
		p.SKU = update.SKU
		p.Price = update.Price
		p.Stock = update.Stock
		p.Category = update.Category
		p.UpdatedAt = time.Now()
		addAudit("inventory.update", "product", "success", currentUserID(r))
		writeJSON(w, 200, p)

	case http.MethodDelete:
		if !requirePerm(w, r, "inventory:delete") {
			return
		}
		delete(products, id)
		addAudit("inventory.delete", "product", "success", currentUserID(r))
		writeJSON(w, 200, map[string]bool{"deleted": true})

	default:
		writeError(w, 405, "method not allowed")
	}
}
