package com.example.erp.controller;

import com.example.erp.model.Order;
import com.example.erp.model.Product;
import com.example.erp.security.RequirePermission;
import dev.ggid.sdk.GGIDUser;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;

import java.util.*;

/**
 * Main ERP controller — demonstrates fine-grained RBAC.
 *
 * Each endpoint is annotated with @RequirePermission which calls
 * GGID's policy check API before execution.
 *
 * Different roles see different modules and have different action buttons:
 * - Sales Manager: orders (full) + customers (full) + inventory (read) + reports (read)
 * - Warehouse Manager: inventory (full) + orders (read/write)
 * - Finance Officer: invoices (full) + payments (full) + orders (read)
 * - HR Manager: employees (full)
 * - Production Manager: production (full) + inventory (read)
 * - System Admin: everything
 */
@Controller
public class ErpController {

    // --- Mock data store (in-memory for demo) ---
    private final List<Product> products = new ArrayList<>(List.of(
        new Product("P001", "Laptop Pro 15", "LP15-001", 120, 1299.00),
        new Product("P002", "Wireless Mouse", "WM-002", 500, 29.99),
        new Product("P003", "USB-C Hub", "UCH-003", 80, 49.99),
        new Product("P004", "4K Monitor 27", "4KM-004", 45, 449.00)
    ));

    private final List<Order> orders = new ArrayList<>(List.of(
        new Order("ORD-001", "Acme Corp", "Laptop Pro 15", 10, 12990.00, "confirmed"),
        new Order("ORD-002", "TechStart LLC", "Wireless Mouse", 50, 1499.50, "shipped"),
        new Order("ORD-003", "Global Trade", "4K Monitor 27", 5, 2245.00, "pending")
    ));

    // === Dashboard ===

    @GetMapping("/")
    public String dashboard(HttpServletRequest request, Model model) {
        GGIDUser user = (GGIDUser) request.getAttribute("currentUser");
        model.addAttribute("user", user);
        model.addAttribute("products", products);
        model.addAttribute("orders", orders);
        model.addAttribute("pageTitle", "Dashboard");
        return "dashboard";
    }

    // === Inventory Module ===

    @GetMapping("/inventory")
    @RequirePermission(resource = "inventory", action = "read")
    public String inventory(Model model) {
        model.addAttribute("products", products);
        model.addAttribute("pageTitle", "Inventory Management");
        return "inventory";
    }

    @GetMapping("/inventory/new")
    @RequirePermission(resource = "inventory", action = "write")
    public String newProductForm(Model model) {
        model.addAttribute("pageTitle", "New Product");
        return "product-form";
    }

    @PostMapping("/inventory/create")
    @RequirePermission(resource = "inventory", action = "write")
    public String createProduct(@RequestParam String name, @RequestParam String sku,
                                 @RequestParam int stock, @RequestParam double price,
                                 Model model) {
        String id = "P" + String.format("%03d", products.size() + 1);
        products.add(new Product(id, name, sku, stock, price));
        model.addAttribute("message", "Product created: " + name);
        return "redirect:/inventory";
    }

    @PostMapping("/inventory/{id}/delete")
    @RequirePermission(resource = "inventory", action = "delete")
    public String deleteProduct(@PathVariable String id) {
        products.removeIf(p -> p.id.equals(id));
        return "redirect:/inventory";
    }

    // === Orders Module ===

    @GetMapping("/orders")
    @RequirePermission(resource = "orders", action = "read")
    public String orders(Model model) {
        model.addAttribute("orders", orders);
        model.addAttribute("pageTitle", "Sales Orders");
        return "orders";
    }

    @PostMapping("/orders/{id}/approve")
    @RequirePermission(resource = "orders", action = "approve")
    public String approveOrder(@PathVariable String id) {
        orders.stream().filter(o -> o.id.equals(id)).forEach(o -> o.status = "confirmed");
        return "redirect:/orders";
    }

    @PostMapping("/orders/{id}/ship")
    @RequirePermission(resource = "orders", action = "write")
    public String shipOrder(@PathVariable String id) {
        orders.stream().filter(o -> o.id.equals(id)).forEach(o -> o.status = "shipped");
        return "redirect:/orders";
    }

    // === Reports Module ===

    @GetMapping("/reports")
    @RequirePermission(resource = "reports", action = "read")
    public String reports(Model model) {
        double totalInventory = products.stream().mapToDouble(p -> p.stock * p.price).sum();
        double totalOrders = orders.stream().mapToDouble(o -> o.total).sum();
        model.addAttribute("totalInventory", totalInventory);
        model.addAttribute("totalOrders", totalOrders);
        model.addAttribute("productCount", products.size());
        model.addAttribute("orderCount", orders.size());
        model.addAttribute("pageTitle", "Reports & Analytics");
        return "reports";
    }

    // === Permission Debug ===

    @GetMapping("/my-permissions")
    public String myPermissions(HttpServletRequest request, Model model) {
        GGIDUser user = (GGIDUser) request.getAttribute("currentUser");
        model.addAttribute("user", user);
        model.addAttribute("pageTitle", "My Permissions");
        return "my-permissions";
    }
}
