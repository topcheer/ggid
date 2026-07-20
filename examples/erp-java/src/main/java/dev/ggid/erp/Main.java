package dev.ggid.erp;

import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDClient;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * ERP Java Demo — 7 module CRUD with GGID IAM integration.
 * Modules: auth, users, roles, orgs, inventory, orders, audit
 * Permission matrix: Viewer, Sales, Manager, Admin
 * Row-level filtering: orders module (orders:read:all sees all, else own only)
 */
public class Main {
    private static final int DEFAULT_PORT = 8080;
    private static final String GGID_URL = System.getenv().getOrDefault("GGID_URL", "https://ggid.iot2.win");
    private static final String TENANT_ID = System.getenv().getOrDefault("TENANT_ID", "00000000-0000-0000-0000-000000000001");

    // In-memory data stores (demo only; production would use a real DB)
    static final Map<String, InventoryItem> inventory = new ConcurrentHashMap<>();
    static final Map<String, Order> orders = new ConcurrentHashMap<>();
    static final Map<String, AuditLog> auditLogs = new ConcurrentHashMap<>();

    // The shared GGID client (for server-to-server admin operations)
    static final GGIDClient ggid = new GGIDClient(new GGIDClient.Config(
            GGID_URL, TENANT_ID, System.getenv("GGID_API_KEY")));

    public static void main(String[] args) throws IOException {
        int port = Integer.parseInt(System.getenv().getOrDefault("PORT", String.valueOf(DEFAULT_PORT)));

        // Seed demo data
        seedData();

        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);

        // Register routes — each module is a handler
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

    // --- Utility methods for handlers ---

    static void sendJson(HttpExchange exchange, int status, String json) throws IOException {
        byte[] resp = json.getBytes(java.nio.charset.StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.sendResponseHeaders(status, resp.length);
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(resp);
        }
    }

    static String extractToken(HttpExchange exchange) {
        String auth = exchange.getRequestHeaders().getFirst("Authorization");
        if (auth != null && auth.startsWith("Bearer ")) {
            return auth.substring(7);
        }
        return null;
    }

    static dev.ggid.sdk.GGIDUser verifyToken(String token) {
        if (token == null) return null;
        try {
            return ggid.verifyToken(token);
        } catch (Exception e) {
            return null;
        }
    }
}
