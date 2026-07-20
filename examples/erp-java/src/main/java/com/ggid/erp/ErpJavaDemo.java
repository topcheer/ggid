package com.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;
import com.ggid.sdk.GGIDClient;
import com.ggid.sdk.model.*;
import com.google.gson.*;

import java.io.*;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.concurrent.*;

/**
 * Cross-Board ERP Demo — Java implementation.
 * Tests all GGID core features via Java SDK.
 *
 * Run: GGID_URL=https://ggid.iot2.win java -jar erp-java.jar
 */
public class ErpJavaDemo {

    private static final Gson gson = new GsonBuilder().setPrettyPrinting().create();
    private static final String ggidUrl = env("GGID_URL", "http://localhost:8080");
    private static final String tenantId = env("TENANT_ID", "00000000-0000-0000-0000-000000000001");
    private static final int port = Integer.parseInt(env("PORT", "9300"));
    private static final String adminUser = env("ADMIN_USERNAME", "admin");
    private static final String adminPass = env("ADMIN_PASSWORD", "");

    // In-memory data
    private static final List<Map<String, Object>> inventory = new CopyOnWriteArrayList<>(List.of(
        Map.of("id", "p001", "name", "Widget A", "stock", 150, "price", 29.99),
        Map.of("id", "p002", "name", "Widget B", "stock", 80, "price", 49.99),
        Map.of("id", "p003", "name", "Gadget C", "stock", 200, "price", 19.99)
    ));
    private static final List<Map<String, Object>> orders = new CopyOnWriteArrayList<>(List.of(
        Map.of("id", "o001", "customer", "Acme Corp", "product_id", "p001", "qty", 10, "status", "pending", "total", 299.90),
        Map.of("id", "o002", "customer", "Beta Inc", "product_id", "p002", "qty", 5, "status", "approved", "total", 249.95)
    ));

    public static void main(String[] args) throws Exception {
        var client = new GGIDClient(ggidUrl, tenantId);
        var server = HttpServer.create(new InetSocketAddress(port), 0);

        server.createContext("/", new RootHandler(client));
        server.setExecutor(Executors.newFixedThreadPool(8));
        server.start();
        System.out.println("ERP Java Demo on :" + port + " | GGID: " + ggidUrl);
    }

    static class RootHandler implements HttpHandler {
        private final GGIDClient client;

        RootHandler(GGIDClient client) { this.client = client; }

        @Override
        public void handle(HttpExchange ex) throws IOException {
            var path = ex.getRequestURI().getPath();
            var method = ex.getRequestMethod();
            var token = getToken(ex);

            try {
                // Public routes
                if (path.equals("/") || path.equals("/health")) {
                    sendJson(ex, 200, Map.of("app", "ERP Java Demo", "status", "ok"));
                    return;
                }

                if (path.equals("/api/auth/login") && method.equals("POST")) {
                    var body = parseBody(ex);
                    var result = client.login((String) body.get("username"), (String) body.get("password"));
                    sendJson(ex, 200, result);
                    return;
                }

                // Auth required
                if (token == null) {
                    sendJson(ex, 401, Map.of("error", "Bearer token required"));
                    return;
                }
                var perms = extractPermissions(token);

                // Inventory
                if (path.equals("/api/inventory") && method.equals("GET")) {
                    if (!hasPerm(perms, "inventory:read")) { sendJson(ex, 403, Map.of("error", "missing inventory:read")); return; }
                    sendJson(ex, 200, Map.of("items", inventory, "count", inventory.size()));
                    return;
                }
                if (path.equals("/api/inventory") && method.equals("POST")) {
                    if (!hasPerm(perms, "inventory:write")) { sendJson(ex, 403, Map.of("error", "missing inventory:write")); return; }
                    var body = parseBody(ex);
                    body.put("id", String.format("p%03d", inventory.size() + 1));
                    inventory.add(body);
                    sendJson(ex, 201, body);
                    return;
                }

                // Orders
                if (path.equals("/api/orders") && method.equals("GET")) {
                    if (!hasPerm(perms, "orders:read")) { sendJson(ex, 403, Map.of("error", "missing orders:read")); return; }
                    sendJson(ex, 200, Map.of("orders", orders, "count", orders.size()));
                    return;
                }
                if (path.equals("/api/orders") && method.equals("POST")) {
                    if (!hasPerm(perms, "orders:write")) { sendJson(ex, 403, Map.of("error", "missing orders:write")); return; }
                    var body = parseBody(ex);
                    body.put("id", String.format("o%03d", orders.size() + 1));
                    body.put("status", "pending");
                    orders.add(body);
                    sendJson(ex, 201, body);
                    return;
                }
                if (path.startsWith("/api/orders/") && path.endsWith("/approve") && method.equals("POST")) {
                    if (!hasPerm(perms, "orders:approve")) { sendJson(ex, 403, Map.of("error", "missing orders:approve")); return; }
                    var orderId = path.split("/")[3];
                    for (var o : orders) {
                        if (o.get("id").equals(orderId)) {
                            ((Map<String, Object>) o).put("status", "approved");
                            sendJson(ex, 200, o);
                            return;
                        }
                    }
                    sendJson(ex, 404, Map.of("error", "order not found"));
                    return;
                }

                // Users
                if (path.equals("/api/users") && method.equals("GET")) {
                    if (!hasPerm(perms, "users:read")) { sendJson(ex, 403, Map.of("error", "missing users:read")); return; }
                    var users = client.listUsers(token);
                    sendJson(ex, 200, users);
                    return;
                }

                // Roles
                if (path.equals("/api/roles") && method.equals("GET")) {
                    if (!hasPerm(perms, "roles:read")) { sendJson(ex, 403, Map.of("error", "missing roles:read")); return; }
                    var roles = client.listRoles(token);
                    sendJson(ex, 200, roles);
                    return;
                }

                // Audit
                if (path.equals("/api/audit") && method.equals("GET")) {
                    if (!hasPerm(perms, "audit:read")) { sendJson(ex, 403, Map.of("error", "missing audit:read")); return; }
                    var events = client.listAuditEvents(token, tenantId);
                    sendJson(ex, 200, events);
                    return;
                }

                // My Permissions
                if (path.equals("/api/my-permissions") && method.equals("GET")) {
                    sendJson(ex, 200, Map.of(
                        "permissions", perms,
                        "can_read_inventory", hasPerm(perms, "inventory:read"),
                        "can_write_orders", hasPerm(perms, "orders:write"),
                        "can_approve_orders", hasPerm(perms, "orders:approve")
                    ));
                    return;
                }

                sendJson(ex, 404, Map.of("error", "not found", "path", path));
            } catch (Exception e) {
                sendJson(ex, 500, Map.of("error", e.getMessage()));
            }
        }
    }

    // --- Helpers ---

    static String getToken(HttpExchange ex) {
        var auth = ex.getRequestHeaders().getFirst("Authorization");
        if (auth != null && auth.startsWith("Bearer ")) return auth.substring(7);
        return null;
    }

    @SuppressWarnings("unchecked")
    static List<String> extractPermissions(String token) {
        var parts = token.split("\\.");
        if (parts.length < 2) return List.of();
        var payload = parts[1];
        payload += "=".repeat((4 - payload.length() % 4) % 4);
        try {
            var bytes = Base64.getDecoder().decode(payload);
            var json = new String(bytes, StandardCharsets.UTF_8);
            var doc = JsonParser.parseString(json).getAsJsonObject();
            if (doc.has("permissions")) {
                var arr = doc.getAsJsonArray("permissions");
                var result = new ArrayList<String>();
                for (var e : arr) result.add(e.getAsString());
                return result;
            }
        } catch (Exception ignored) {}
        return List.of();
    }

    static boolean hasPerm(List<String> perms, String perm) {
        return perms.contains("admin") || perms.contains(perm);
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> parseBody(HttpExchange ex) throws IOException {
        var body = new String(ex.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
        if (body.isEmpty()) return new HashMap<>();
        return gson.fromJson(body, Map.class);
    }

    static void sendJson(HttpExchange ex, int status, Object data) throws IOException {
        var json = gson.toJson(data);
        var bytes = json.getBytes(StandardCharsets.UTF_8);
        ex.getResponseHeaders().set("Content-Type", "application/json");
        ex.sendResponseHeaders(status, bytes.length);
        try (var os = ex.getResponseBody()) { os.write(bytes); }
    }

    static String env(String key, String def) {
        var v = System.getenv(key);
        return v != null && !v.isEmpty() ? v : def;
    }
}
