package com.example.erp.security;

import java.util.*;

/**
 * ERP Permission Matrix — defines what each role can do.
 *
 * This serves TWO purposes:
 * 1. Local fallback when GGID policy API is unreachable
 * 2. Reference documentation for setting up GGID permissions
 *
 * In production, these permissions live in GGID's database and are
 * checked via POST /api/v1/policies/check.
 *
 * Module -> Action Mapping:
 *   read   = view list/detail
 *   write  = create/edit
 *   delete = remove
 *   approve= approve/reject workflows
 */
public class ErpPermissions {

    // 8 ERP modules
    public static final String INVENTORY  = "inventory";
    public static final String ORDERS     = "orders";
    public static final String CUSTOMERS  = "customers";
    public static final String INVOICES   = "invoices";
    public static final String PAYMENTS   = "payments";
    public static final String EMPLOYEES  = "employees";
    public static final String PRODUCTION = "production";
    public static final String REPORTS    = "reports";

    // 6 ERP roles
    public static final String SALES_MANAGER      = "erp:sales_manager";
    public static final String WAREHOUSE_MANAGER   = "erp:warehouse_manager";
    public static final String FINANCE_OFFICER     = "erp:finance_officer";
    public static final String HR_MANAGER          = "erp:hr_manager";
    public static final String PRODUCTION_MANAGER  = "erp:production_manager";
    public static final String SYSTEM_ADMIN        = "erp:system_admin";

    // Permission matrix: role -> { module -> set of actions }
    private static final Map<String, Map<String, Set<String>>> MATRIX = new HashMap<>();

    static {
        // Sales Manager: full control on orders & customers, read on inventory & reports
        MATRIX.put(SALES_MANAGER, Map.of(
            ORDERS,     Set.of("read", "write", "approve"),
            CUSTOMERS,  Set.of("read", "write", "delete"),
            INVENTORY,  Set.of("read"),
            REPORTS,    Set.of("read")
        ));

        // Warehouse Manager: full control on inventory, read on orders, manage shipping
        MATRIX.put(WAREHOUSE_MANAGER, Map.of(
            INVENTORY,  Set.of("read", "write", "delete"),
            ORDERS,     Set.of("read", "write"),
            REPORTS,    Set.of("read")
        ));

        // Finance Officer: manage invoices & payments, read on orders & reports
        MATRIX.put(FINANCE_OFFICER, Map.of(
            INVOICES,   Set.of("read", "write", "approve", "delete"),
            PAYMENTS,   Set.of("read", "write", "approve"),
            ORDERS,     Set.of("read"),
            REPORTS,    Set.of("read")
        ));

        // HR Manager: manage employees, read reports
        MATRIX.put(HR_MANAGER, Map.of(
            EMPLOYEES,  Set.of("read", "write", "delete"),
            REPORTS,    Set.of("read")
        ));

        // Production Manager: manage production, read inventory & reports
        MATRIX.put(PRODUCTION_MANAGER, Map.of(
            PRODUCTION, Set.of("read", "write", "approve"),
            INVENTORY,  Set.of("read"),
            REPORTS,    Set.of("read")
        ));

        // System Admin: full access to everything
        MATRIX.put(SYSTEM_ADMIN, Map.of(
            INVENTORY,  Set.of("read", "write", "delete", "approve"),
            ORDERS,     Set.of("read", "write", "delete", "approve"),
            CUSTOMERS,  Set.of("read", "write", "delete"),
            INVOICES,   Set.of("read", "write", "delete", "approve"),
            PAYMENTS,   Set.of("read", "write", "approve"),
            EMPLOYEES,  Set.of("read", "write", "delete"),
            PRODUCTION, Set.of("read", "write", "approve"),
            REPORTS,    Set.of("read")
        ));
    }

    /**
     * Check if a role has a specific permission.
     */
    public static boolean isAllowed(String role, String module, String action) {
        Map<String, Set<String>> rolePerms = MATRIX.get(role);
        if (rolePerms == null) return false;
        Set<String> actions = rolePerms.get(module);
        if (actions == null) return false;
        return actions.contains(action);
    }

    /**
     * Get all modules a role can access (with their actions).
     */
    public static Map<String, Set<String>> getRolePermissions(String role) {
        return MATRIX.getOrDefault(role, Map.of());
    }

    /**
     * Get all ERP role keys.
     */
    public static List<String> getAllRoles() {
        return List.of(SALES_MANAGER, WAREHOUSE_MANAGER, FINANCE_OFFICER,
                HR_MANAGER, PRODUCTION_MANAGER, SYSTEM_ADMIN);
    }

    /**
     * Get all ERP modules.
     */
    public static List<String> getAllModules() {
        return List.of(INVENTORY, ORDERS, CUSTOMERS, INVOICES,
                PAYMENTS, EMPLOYEES, PRODUCTION, REPORTS);
    }
}
