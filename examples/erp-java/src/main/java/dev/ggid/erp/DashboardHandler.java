package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.Map;

/**
 * / — Dashboard showing current user info, permissions, and module access
 */
public class DashboardHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;

        // Build permission-aware module list
        Map<String, Object> modules = new java.util.LinkedHashMap<>();
        modules.put("inventory", user.hasPermission("inventory:read"));
        modules.put("orders", user.hasPermission("orders:read") || user.hasPermission("orders:read:all"));
        modules.put("users", user.hasPermission("users:read"));
        modules.put("roles", user.hasPermission("roles:read"));
        modules.put("audit", user.hasPermission("audit:read"));
        modules.put("orgs", true); // readable by all authenticated users

        Map<String, Object> response = Map.of(
                "user", Map.of(
                        "sub", user.getSubject() != null ? user.getSubject() : "unknown",
                        "email", user.getClaim("email") != null ? user.getClaim("email") : "",
                        "name", user.getClaim("name") != null ? user.getClaim("name") : "",
                        "tenant_id", user.getClaim("tenant_id") != null ? user.getClaim("tenant_id") : ""
                ),
                "permissions", user.getPermissions(),
                "modules", modules,
                "stats", Map.of(
                        "inventory_count", Main.inventory.size(),
                        "orders_count", Main.orders.size(),
                        "audit_count", Main.auditLogs.size()
                )
        );
        sendJson(exchange, 200, json(response));
    }
}
