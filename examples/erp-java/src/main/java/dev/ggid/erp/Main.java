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
        server.createContext("/auth", new AuthHandler());
        server.createContext("/users", new UsersHandler());
        server.createContext("/roles", new RolesHandler());
        server.createContext("/orgs", new OrgsHandler());
        server.createContext("/inventory", new InventoryHandler());
        server.createContext("/orders", new OrdersHandler());
        server.createContext("/audit", new AuditHandler());
        server.createContext("/", new DashboardHandler());
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

    // Decode JWT to GGIDUser without external verification (demo only)
    static GGIDUser verifyToken(String token) {
        if (token == null) return null;
        try {
            String[] parts = token.split("\\.");
            if (parts.length < 2) return null;
            String payload = new String(Base64.getUrlDecoder().decode(parts[1]));
            @SuppressWarnings("unchecked")
            Map<String, Object> claims = mapper.readValue(payload, Map.class);

            GGIDUser user = new GGIDUser();
            user.userId = (String) claims.get("sub");
            user.tenantId = (String) claims.get("tenant_id");
            user.email = (String) claims.get("email");
            user.username = (String) claims.getOrDefault("name", claims.get("sub"));

            // Parse scopes from space-delimited string
            String scope = (String) claims.get("scope");
            List<String> perms = new ArrayList<>();
            if (scope != null) {
                for (String s : scope.split(" ")) {
                    if (!s.isEmpty()) perms.add(s);
                }
            }
            // Also check "scopes" array
            @SuppressWarnings("unchecked")
            List<String> scopesArr = (List<String>) claims.get("scopes");
            if (scopesArr != null) perms.addAll(scopesArr);
            user.permissions = perms.toArray(new String[0]);

            @SuppressWarnings("unchecked")
            List<String> rolesList = (List<String>) claims.get("roles");
            if (rolesList != null) user.roles = rolesList.toArray(new String[0]);
            else user.roles = new String[0];

            return user;
        } catch (Exception e) {
            return null;
        }
    }
}
