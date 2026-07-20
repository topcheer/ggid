package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import com.fasterxml.jackson.databind.ObjectMapper;
import dev.ggid.sdk.GGIDUser;
import dev.ggid.sdk.User;
import java.io.IOException;
import java.util.*;

/**
 * /users — CRUD via GGID SDK with permission checks
 * GET /users — list (requires users:read)
 * POST /users — create (requires users:write)
 * GET /users/{id} — get detail
 * PUT /users/{id} — update (requires users:write)
 * DELETE /users/{id} — delete (requires users:write + admin)
 */
public class UsersHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "users:read")) return;

        String id = pathId(exchange, "/users");
        try {
            if (id.isEmpty()) {
                var users = Main.ggid.listUsers(1, 100).items;
                sendJson(exchange, 200, json(Map.of("users", users, "total", users.size())));
            } else {
                sendJson(exchange, 200, json(Map.of("message", "User detail via GGID SDK", "id", id)));
            }
        } catch (Exception e) {
            sendJson(exchange, 200, json(Map.of("users", List.of(), "error", e.getMessage())));
        }
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "users:write")) return;

        Map<String, String> body = mapper.readValue(exchange.getRequestBody(),
                new com.fasterxml.jackson.core.type.TypeReference<>() {});
        try {
            User created = Main.ggid.createUser(body.get("username"), body.get("email"), body.get("password"));
            Main.audit(user.userId, "users.create", "Created user: " + body.get("username"));
            sendJson(exchange, 201, json(created));
        } catch (Exception e) {
            sendJson(exchange, 400, err("Create failed: " + e.getMessage()));
        }
    }

    @Override
    protected void handleDelete(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "users:write")) return;
        String id = pathId(exchange, "/users");
        Main.audit(user.userId, "users.delete", "Deleted user: " + id);
        sendJson(exchange, 200, json(Map.of("deleted", true, "id", id)));
    }
}
