package com.example.erp.model;

/**
 * Sales order for the ERP demo.
 */
public class Order {
    public String id;
    public String customerName;
    public String product;
    public int quantity;
    public double total;
    public String status; // pending, confirmed, shipped, delivered

    public Order() {}

    public Order(String id, String customerName, String product, int quantity, double total, String status) {
        this.id = id;
        this.customerName = customerName;
        this.product = product;
        this.quantity = quantity;
        this.total = total;
        this.status = status;
    }
}
