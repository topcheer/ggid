package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * /inventory — CRUD with permission checks and org-level filtering
 * GET /inventory — list (inventory:read)
 * POST /inventory — create (inventory:write)
 * DELETE /inventory/{id} — delete (inventory:write)
 */
public class InventoryHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "inventory:read")) return;

        List<InventoryItem> items = new ArrayList<>(Main.inventory.values());
        // Filter by org if user doesn't have inventory:read:all
        if (!user.hasPermission("inventory:read:all")) {
            String userOrg = "";
            if (userOrg != null && !userOrg.isEmpty()) {
                items.removeIf(item -> !userOrg.equals(item.orgId));
            }
        }
        sendJson(exchange, 200, json(Map.of("inventory", items, "total", items.size())));
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "inventory:write")) return;

        InventoryItem item = mapper.readValue(exchange.getRequestBody(), InventoryItem.class);
        if (item.id == null) item.id = "INV-" + System.currentTimeMillis();
        Main.inventory.put(item.id, item);
        Main.audit(user.userId, "inventory.create", "Created: " + item.id);
        sendJson(exchange, 201, json(item));
    }

    @Override
    protected void handleDelete(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "inventory:write")) return;
        String id = pathId(exchange, "/inventory");
        Main.inventory.remove(id);
        Main.audit(user.userId, "inventory.delete", "Deleted: " + id);
        sendJson(exchange, 200, json(Map.of("deleted", true, "id", id)));
    }
}
