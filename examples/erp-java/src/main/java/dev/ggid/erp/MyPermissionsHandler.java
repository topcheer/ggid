package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Map;

/**
 * GET /api/my-permissions — returns the caller's permissions from JWT claims.
 * Matches the pattern used by Go, Node, Python, and C# demos.
 */
public class MyPermissionsHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;

        List<String> perms = user.permissions != null ? Arrays.asList(user.permissions) : new ArrayList<>();
        boolean canWriteOrders = perms.stream().anyMatch(p -> p.equals("orders:write") || p.equals("admin"));
        boolean canApprove = perms.stream().anyMatch(p -> p.equals("orders:approve") || p.equals("admin"));

        sendJson(exchange, 200, json(Map.of(
            "permissions", perms,
            "can_write_orders", canWriteOrders,
            "can_approve", canApprove
        )));
    }
}
