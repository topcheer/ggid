package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.List;
import java.util.Map;

/**
 * /roles — list/create/assign roles
 * Requires roles:read / roles:write
 */
public class RolesHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "roles:read")) return;
        try {
            var roles = Main.ggid.listRoles();
            sendJson(exchange, 200, json(Map.of("roles", roles, "total", roles.size())));
        } catch (Exception e) {
            sendJson(exchange, 200, json(Map.of("roles", List.of(), "error", e.getMessage())));
        }
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "roles:write")) return;
        Map<String, String> body = mapper.readValue(exchange.getRequestBody(),
                new com.fasterxml.jackson.core.type.TypeReference<>() {});
        Main.audit(user.getSubject(), "roles.create", "Created role: " + body.get("name"));
        sendJson(exchange, 201, json(Map.of("created", true, "name", body.get("name"))));
    }
}
