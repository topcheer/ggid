package com.example.erp.security;

import java.lang.annotation.*;

/**
 * Method-level annotation for permission-based access control.
 *
 * Usage:
 *   @RequirePermission(resource = "inventory", action = "write")
 *   public String createProduct(...) { ... }
 *
 * Checked by PermissionAspect which calls GGID checkPermission API.
 */
@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
@Documented
public @interface RequirePermission {
    /** Resource type: inventory, orders, invoices, customers, etc. */
    String resource();
    /** Action: read, write, delete, approve, etc. */
    String action();
}
