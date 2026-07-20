package dev.ggid.erp;

import java.time.Instant;

class InventoryItem {
    public String id, name, orgId;
    public int quantity;
    public double price;
    public InventoryItem() {}
    public InventoryItem(String id, String name, int qty, double price, String orgId) {
        this.id=id; this.name=name; this.quantity=qty; this.price=price; this.orgId=orgId;
    }
}

class Order {
    public String id, customerName, productName, status, orgId, createdBy;
    public int quantity;
    public double totalAmount;
    public Order() {}
    public Order(String id, String cust, String prod, int qty, double total, String status, String orgId, String createdBy) {
        this.id=id; this.customerName=cust; this.productName=prod; this.quantity=qty;
        this.totalAmount=total; this.status=status; this.orgId=orgId; this.createdBy=createdBy;
    }
}

class AuditLog {
    public String id, actor, action, detail, timestamp;
    public AuditLog() {}
    public AuditLog(String id, String actor, String action, String detail, String ts) {
        this.id=id; this.actor=actor; this.action=action; this.detail=detail; this.timestamp=ts;
    }
}
