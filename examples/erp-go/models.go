package main

import (
	"fmt"
	"time"
)

// Product represents an inventory item
 type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Order represents a sales order
 type Order struct {
	ID         string    `json:"id"`
	Customer   string    `json:"customer"`
	ProductID  string    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	Amount     float64   `json:"amount"`
	Status     string    `json:"status"` // pending, approved, shipped, cancelled
	OrgID      string    `json:"org_id"`
	GroupID    string    `json:"group_id"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditEntry represents an audit log entry
 type AuditEntry struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Result    string    `json:"result"`
	ActorID   string    `json:"actor_id"`
	Timestamp time.Time `json:"timestamp"`
}

// In-memory stores (demo only — real implementation would use DB)
var (
	products  = map[string]*Product{}
	orders    = map[string]*Order{}
	auditLog  = []AuditEntry{}
	productSeq = 0
	orderSeq   = 0
)

func nextProductID() string {
	productSeq++
	return fmt.Sprintf("PROD-%04d", productSeq)
}

func nextOrderID() string {
	orderSeq++
	return fmt.Sprintf("ORD-%04d", orderSeq)
}

func addAudit(action, resource, result, actorID string) {
	auditLog = append(auditLog, AuditEntry{
		ID:        fmt.Sprintf("AUD-%d", len(auditLog)+1),
		Action:    action,
		Resource:  resource,
		Result:    result,
		ActorID:   actorID,
		Timestamp: time.Now(),
	})
}
