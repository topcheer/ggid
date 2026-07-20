package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.fasterxml.jackson.databind.ObjectMapper;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.Map;

/**
 * Base handler with auth check, permission guard, and JSON helpers.
 */
public abstract class BaseHandler implements HttpHandler {
    protected static final ObjectMapper mapper = new ObjectMapper();

    @Override
    public void handle(HttpExchange exchange) throws IOException {
        // CORS
        exchange.getResponseHeaders().set("Access-Control-Allow-Origin", "*");
        exchange.getResponseHeaders().set("Access-Control-Allow-Headers", "Authorization, Content-Type");
        exchange.getResponseHeaders().set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS");

        if ("OPTIONS".equalsIgnoreCase(exchange.getRequestMethod())) {
            exchange.sendResponseHeaders(204, -1);
            return;
        }

        try {
            switch (exchange.getRequestMethod().toUpperCase()) {
                case "GET" -> handleGet(exchange);
                case "POST" -> handlePost(exchange);
                case "PUT" -> handlePut(exchange);
                case "DELETE" -> handleDelete(exchange);
                default -> sendJson(exchange, 405, json(Map.of("error", "Method not allowed")));
            }
        } catch (Exception e) {
            sendJson(exchange, 500, json(Map.of("error", e.getMessage())));
        }
    }

    protected void handleGet(HttpExchange e) throws IOException { sendJson(e, 405, err("GET not supported")); }
    protected void handlePost(HttpExchange e) throws IOException { sendJson(e, 405, err("POST not supported")); }
    protected void handlePut(HttpExchange e) throws IOException { sendJson(e, 405, err("PUT not supported")); }
    protected void handleDelete(HttpExchange e) throws IOException { sendJson(e, 405, err("DELETE not supported")); }

    // --- Auth helpers ---

    protected GGIDUser requireAuth(HttpExchange exchange) throws IOException {
        String token = Main.extractToken(exchange);
        GGIDUser user = Main.verifyToken(token);
        if (user == null) {
            sendJson(exchange, 401, err("Unauthorized — valid Bearer token required"));
        }
        return user;
    }

    protected boolean requirePermission(HttpExchange exchange, GGIDUser user, String permission) throws IOException {
        if (user != null && user.hasPermission(permission)) return true;
        sendJson(exchange, 403, err("Forbidden — requires permission: " + permission));
        return false;
    }

    protected boolean requireAnyPermission(HttpExchange exchange, GGIDUser user, String... perms) throws IOException {
        for (String p : perms) {
            if (user != null && user.hasPermission(p)) return true;
        }
        sendJson(exchange, 403, err("Forbidden — requires one of: " + String.join(",", perms)));
        return false;
    }

    // --- JSON helpers ---

    protected void sendJson(HttpExchange exchange, int status, String json) throws IOException {
        Main.sendJson(exchange, status, json);
    }

    protected String json(Object obj) {
        try { return mapper.writeValueAsString(obj); }
        catch (Exception e) { return "{}"; }
    }

    protected String err(String msg) {
        return json(Map.of("error", msg));
    }

    protected String pathId(HttpExchange exchange, String prefix) {
        String path = exchange.getRequestURI().getPath();
        if (path.length() > prefix.length() + 1) {
            return path.substring(prefix.length() + 1);
        }
        return "";
    }
}
