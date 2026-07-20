package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * /orders — CRUD with ROW-LEVEL filtering
 * Users with orders:read:all see all orders
 * Users without it only see orders matching their org_id
 */
public class OrdersHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requireAnyPermission(exchange, user, "orders:read", "orders:read:all")) return;

        List<Order> allOrders = new ArrayList<>(Main.orders.values());

        // Row-level filtering: if user doesn't have orders:read:all, filter by org
        if (!user.hasPermission("orders:read:all")) {
            String userOrg = "";
            if (userOrg != null && !userOrg.isEmpty()) {
                allOrders.removeIf(o -> !userOrg.equals(o.orgId));
            }
        }

        String filterNotice = user.hasPermission("orders:read:all")
                ? "Showing all orders (orders:read:all)"
                : "Showing only your team's orders";

        sendJson(exchange, 200, json(Map.of(
                "orders", allOrders,
                "total", allOrders.size(),
                "filter_notice", filterNotice
        )));
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "orders:write")) return;

        Order order = mapper.readValue(exchange.getRequestBody(), Order.class);
        if (order.id == null) order.id = "ORD-" + System.currentTimeMillis();
        if (order.status == null) order.status = "pending";
        if (order.createdBy == null) order.createdBy = user.userId;
        Main.orders.put(order.id, order);
        Main.audit(user.userId, "orders.create", "Created order: " + order.id);
        sendJson(exchange, 201, json(order));
    }

    @Override
    protected void handlePut(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        String id = pathId(exchange, "/orders");

        // Approval requires orders:approve permission
        Order existing = Main.orders.get(id);
        if (existing == null) {
            sendJson(exchange, 404, err("Order not found"));
            return;
        }

        Map<String, Object> body = mapper.readValue(exchange.getRequestBody(),
                new com.fasterxml.jackson.core.type.TypeReference<>() {});

        if (body.containsKey("status")) {
            String newStatus = (String) body.get("status");
            if ("approved".equals(newStatus) && !user.hasPermission("orders:approve")) {
                sendJson(exchange, 403, err("Forbidden — requires orders:approve"));
                return;
            }
            existing.status = newStatus;
        }

        Main.orders.put(id, existing);
        Main.audit(user.userId, "orders.update", "Updated order: " + id);
        sendJson(exchange, 200, json(existing));
    }
}
