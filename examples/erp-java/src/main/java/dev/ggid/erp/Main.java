package dev.ggid.erp;

import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.GGIDUser;

import java.util.Base64;
import java.util.Map;
import java.util.List;
import java.util.ArrayList;
import java.util.concurrent.ConcurrentHashMap;

import com.fasterxml.jackson.databind.ObjectMapper;

public class Main {
    private static final int DEFAULT_PORT = 8080;
    static final String GGID_URL = System.getenv().getOrDefault("GGID_URL", "https://ggid.iot2.win");
    static final String TENANT_ID = System.getenv().getOrDefault("TENANT_ID", "00000000-0000-0000-0000-000000000001");
    static final ObjectMapper mapper = new ObjectMapper();

    static final Map<String, InventoryItem> inventory = new ConcurrentHashMap<>();
    static final Map<String, Order> orders = new ConcurrentHashMap<>();
    static final Map<String, AuditLog> auditLogs = new ConcurrentHashMap<>();

    static final GGIDClient ggid = new GGIDClient(new GGIDClient.Config(GGID_URL));

    public static void main(String[] args) throws Exception {
        int port = Integer.parseInt(System.getenv().getOrDefault("PORT", String.valueOf(DEFAULT_PORT)));
        seedData();

        com.sun.net.httpserver.HttpServer server = com.sun.net.httpserver.HttpServer.create(new java.net.InetSocketAddress(port), 0);
        server.createContext("/api/auth", new AuthHandler());
        server.createContext("/api/users", new UsersHandler());
        server.createContext("/api/roles", new RolesHandler());
        server.createContext("/api/orgs", new OrgsHandler());
        server.createContext("/api/inventory", new InventoryHandler());
        server.createContext("/api/orders", new OrdersHandler());
        server.createContext("/api/audit", new AuditHandler());
        server.createContext("/api/dashboard", new DashboardHandler());
        server.createContext("/health", new HealthHandler());
        server.setExecutor(java.util.concurrent.Executors.newFixedThreadPool(10));
        server.start();
        System.out.println("ERP Java Demo on port " + port + " (GGID: " + GGID_URL + ")");
    }

    private static void seedData() {
        inventory.put("INV-001", new InventoryItem("INV-001", "Widget Pro", 100, 29.99, "team-a"));
        inventory.put("INV-002", new InventoryItem("INV-002", "Cloud License", 50, 999.00, "team-b"));
        inventory.put("INV-003", new InventoryItem("INV-003", "Hardware Kit", 25, 459.00, "team-a"));
        orders.put("ORD-001", new Order("ORD-001", "Acme Corp", "Widget Pro X1", 100, 2999.00, "pending", "team-a", "alice"));
        orders.put("ORD-002", new Order("ORD-002", "TechStart", "Cloud License", 5, 4995.00, "approved", "team-b", "bob"));
        orders.put("ORD-003", new Order("ORD-003", "Global Dynamics", "Hardware Kit", 10, 4590.00, "pending", "team-a", "alice"));
        audit("system", "startup", "Demo data initialized");
    }

    static void audit(String actor, String action, String detail) {
        String id = "LOG-" + System.currentTimeMillis();
        auditLogs.put(id, new AuditLog(id, actor, action, detail, java.time.Instant.now().toString()));
    }

    static void sendJson(com.sun.net.httpserver.HttpExchange exchange, int status, String json) throws java.io.IOException {
        byte[] resp = json.getBytes(java.nio.charset.StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.sendResponseHeaders(status, resp.length);
        try (java.io.OutputStream os = exchange.getResponseBody()) { os.write(resp); }
    }

    static String extractToken(com.sun.net.httpserver.HttpExchange exchange) {
        String auth = exchange.getRequestHeaders().getFirst("Authorization");
        if (auth != null && auth.startsWith("Bearer ")) return auth.substring(7);
        return null;
    }

    // Verify token via GGID introspect endpoint (no inline base64 decode)
    static GGIDUser verifyToken(String token) {
        if (token == null) return null;
        try {
            String form = "token=" + java.net.URLEncoder.encode(token, java.nio.charset.StandardCharsets.UTF_8) +
                "&client_id=demo&client_secret=demo";
            java.net.HttpURLConnection conn = (java.net.HttpURLConnection) new java.net.URL(GGID_URL + "/api/v1/oauth/introspect").openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/x-www-form-urlencoded");
            conn.setRequestProperty("X-Tenant-ID", TENANT_ID);
            conn.setDoOutput(true);
            conn.getOutputStream().write(form.getBytes(java.nio.charset.StandardCharsets.UTF_8));
            if (conn.getResponseCode() != 200) return null;
            @SuppressWarnings("unchecked")
            Map<String, Object> result = mapper.readValue(conn.getInputStream(), Map.class);
            if (!Boolean.TRUE.equals(result.get("active"))) return null;
            GGIDUser user = new GGIDUser();
            user.userId = (String) result.get("sub");
            user.tenantId = (String) result.get("tenant_id");
            user.email = (String) result.get("email");
            user.username = (String) result.getOrDefault("username", result.get("sub"));
            @SuppressWarnings("unchecked")
            List<String> permList = (List<String>) result.get("permissions");
            user.permissions = permList != null ? permList.toArray(new String[0]) : new String[0];
            @SuppressWarnings("unchecked")
            List<String> rolesList = (List<String>) result.get("roles");
            user.roles = rolesList != null ? rolesList.toArray(new String[0]) : new String[0];
            return user;
        } catch (Exception e) {
            return null;
        }
    }
}
