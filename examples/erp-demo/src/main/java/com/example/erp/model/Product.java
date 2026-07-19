package com.example.erp.model;

/**
 * Simple product/inventory item for the ERP demo.
 */
public class Product {
    public String id;
    public String name;
    public String sku;
    public int stock;
    public double price;

    public Product() {}

    public Product(String id, String name, String sku, int stock, double price) {
        this.id = id;
        this.name = name;
        this.sku = sku;
        this.stock = stock;
        this.price = price;
    }
}
