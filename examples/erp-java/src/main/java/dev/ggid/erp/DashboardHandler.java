package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public class DashboardHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;

        Map<String, Object> modules = new LinkedHashMap<>();
        modules.put("inventory", user.hasPermission("inventory:read"));
        modules.put("orders", user.hasPermission("orders:read") || user.hasPermission("orders:read:all"));
        modules.put("users", user.hasPermission("users:read"));
        modules.put("roles", user.hasPermission("roles:read"));
        modules.put("audit", user.hasPermission("audit:read"));
        modules.put("orgs", true);

        Map<String, Object> userInfo = new LinkedHashMap<>();
        userInfo.put("sub", user.userId != null ? user.userId : "unknown");
        userInfo.put("email", user.email != null ? user.email : "");
        userInfo.put("username", user.username != null ? user.username : "");
        userInfo.put("tenant_id", user.tenantId != null ? user.tenantId : "");
        userInfo.put("roles", user.roles != null ? Arrays.asList(user.roles) : List.of());

        Map<String, Object> response = new LinkedHashMap<>();
        response.put("user", userInfo);
        response.put("permissions", user.permissions != null ? Arrays.asList(user.permissions) : List.of());
        response.put("modules", modules);
        response.put("stats", Map.of(
                "inventory_count", Main.inventory.size(),
                "orders_count", Main.orders.size(),
                "audit_count", Main.auditLogs.size()
        ));
        sendJson(exchange, 200, json(response));
    }
}
