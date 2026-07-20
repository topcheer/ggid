import java.io.*;
import java.net.*;
import java.net.http.*;
import java.util.*;
import com.google.gson.*;

/**
 * GGID OAuth 2.0 Demo with fine-grained permission control.
 *
 * Features:
 *   - OAuth authorization code flow login
 *   - Dashboard with role badges + permission checklist
 *   - Inventory page (inventory:read, write shows buttons)
 *   - Orders page (orders:read, approve/write buttons)
 *   - Admin page (admin scope only)
 *   - 403 page for unauthorized access
 *
 * Run:
 *   GGID_URL=http://localhost:8080 CLIENT_ID=xxx CLIENT_SECRET=xxx javac OAuthDemo.java && java OAuthDemo
 */
public class OAuthDemo {
    private static final Gson gson = new Gson();
    private static String ggidUrl = env("GGID_URL", "http://localhost:8080");
    private static String clientId = env("CLIENT_ID", "demo-client");
    private static String clientSecret = env("CLIENT_SECRET", "demo-secret");
    private static String redirectUri = env("REDIRECT_URI", "http://localhost:3002/auth/callback");
    private static int port = Integer.parseInt(env("PORT", "3002"));
    private static final Map<String, String> sessions = new HashMap<>(); // cookie → JSON session

    public static void main(String[] args) throws IOException {
        ServerSocket server = new ServerSocket(port);
        System.out.println("Java OAuth demo on http://localhost:" + port);
        while (true) {
            Socket client = server.accept();
            handle(client);
        }
    }

    static void handle(Socket client) {
        try (client; BufferedReader in = new BufferedReader(new InputStreamReader(client.getInputStream()));
             OutputStream out = client.getOutputStream()) {
            String requestLine = in.readLine();
            if (requestLine == null) return;
            String[] parts = requestLine.split(" ");
            String method = parts[0];
            String path = parts[1];

            // Read headers
            Map<String, String> headers = new HashMap<>();
            String line;
            while ((line = in.readLine()) != null && !line.isEmpty()) {
                int idx = line.indexOf(":");
                if (idx > 0) headers.put(line.substring(0, idx).trim().toLowerCase(), line.substring(idx + 1).trim());
            }

            String cookie = headers.getOrDefault("cookie", "");
            String sessionJson = extractSession(cookie);

            String resp;
            if (path.equals("/") || path.equals("/index.html")) {
                if (sessionJson != null) {
                    resp = redirect("/dashboard");
                } else {
                    resp = html(homePage());
                }
            } else if (path.startsWith("/auth/login")) {
                resp = redirect(ggidUrl + "/oauth/authorize?response_type=code&client_id=" + clientId
                    + "&redirect_uri=" + URLEncoder.encode(redirectUri, "UTF-8") + "&scope=openid+profile+email");
            } else if (path.startsWith("/auth/callback")) {
                String code = path.split("code=")[1].split("&")[0];
                String token = exchangeCode(code);
                if (token != null) {
                    String sid = UUID.randomUUID().toString();
                    Map<String, Object> claims = decodeJWT(token);
                    Map<String, Object> sess = new LinkedHashMap<>();
                    sess.put("token", token);
                    sess.put("username", claims.getOrDefault("sub", "user"));
                    sess.put("scopes", claims.getOrDefault("scopes", List.of()));
                    sessions.put(sid, gson.toJson(sess));
                    resp = "HTTP/1.1 302 Found\r\nSet-Cookie: ggid_session=" + sid + "; Path=/; Max-Age=3600\r\nLocation: /dashboard\r\n\r\n";
                } else {
                    resp = html("<h1>Login failed</h1>");
                }
            } else if (path.startsWith("/auth/logout")) {
                resp = "HTTP/1.1 302 Found\r\nSet-Cookie: ggid_session=; Path=/; Max-Age=0\r\nLocation: /\r\n\r\n";
            } else if (path.startsWith("/dashboard")) {
                resp = requireAuth(sessionJson, sess -> html(dashboardPage(sess)));
            } else if (path.startsWith("/inventory")) {
                resp = requireAuth(sessionJson, sess -> {
                    if (!hasPermission(sess, "inventory:read")) return forbidden("inventory:read");
                    return html(inventoryPage(sess));
                });
            } else if (path.startsWith("/orders")) {
                resp = requireAuth(sessionJson, sess -> {
                    if (!hasPermission(sess, "orders:read")) return forbidden("orders:read");
                    return html(ordersPage(sess));
                });
            } else if (path.startsWith("/admin")) {
                resp = requireAuth(sessionJson, sess -> {
                    if (!hasPermission(sess, "admin")) return forbidden("admin");
                    return html(adminPage(sess));
                });
            } else {
                resp = "HTTP/1.1 404 Not Found\r\n\r\n<h1>404</h1>";
            }
            out.write(resp.getBytes());
            out.flush();
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    @SuppressWarnings("unchecked")
    static boolean hasPermission(Map<String, Object> session, String perm) {
        List<String> scopes = (List<String>) session.getOrDefault("scopes", List.of());
        if (scopes.contains("platform:admin") || scopes.contains("admin")) return true;
        List<String> roles = scopes.stream().map(String::toLowerCase).toList();
        return switch (perm) {
            case "inventory:read" -> roles.stream().anyMatch(r -> r.contains("warehouse") || r.contains("sales") || r.contains("admin"));
            case "inventory:write" -> roles.stream().anyMatch(r -> r.contains("warehouse") || r.contains("admin"));
            case "orders:read" -> true;
            case "orders:write" -> roles.stream().anyMatch(r -> r.contains("sales") || r.contains("warehouse") || r.contains("admin"));
            case "orders:approve" -> roles.stream().anyMatch(r -> r.contains("sales") || r.contains("admin"));
            case "admin" -> scopes.contains("platform:admin") || scopes.contains("admin");
            default -> false;
        };
    }

    static String homePage() {
        return "<h1>Java OAuth Demo</h1><p><a href=\"/auth/login\">Login with GGID</a></p>";
    }

    @SuppressWarnings("unchecked")
    static String dashboardPage(Map<String, Object> sess) {
        List<String> scopes = (List<String>) sess.getOrDefault("scopes", List.of());
        String username = (String) sess.getOrDefault("username", "user");
        StringBuilder sb = new StringBuilder();
        sb.append("<h1>Dashboard</h1><p>Welcome <b>").append(username).append("</b></p>");
        sb.append("<p>Scopes: ");
        scopes.forEach(s -> sb.append("<span style='background:#3b82f6;color:#fff;padding:2px 8px;margin:2px;border-radius:4px'>").append(s).append("</span>"));
        sb.append("</p><h3>Permissions</h3><ul>");
        for (String p : new String[]{"inventory:read","inventory:write","orders:read","orders:write","orders:approve","admin"}) {
            sb.append("<li style='color:").append(hasPermission(sess, p) ? "green" : "red").append("'>")
              .append(hasPermission(sess, p) ? "YES" : "NO").append(" ").append(p).append("</li>");
        }
        sb.append("</ul>");
        sb.append(menu(sess));
        return sb.toString();
    }

    static String inventoryPage(Map<String, Object> sess) {
        boolean canWrite = hasPermission(sess, "inventory:write");
        StringBuilder sb = new StringBuilder("<h1>Inventory</h1>");
        if (canWrite) sb.append("<button>New Item</button>");
        else sb.append("<p><em>Read-only access.</em></p>");
        sb.append("<table border=1><tr><th>SKU</th><th>Name</th><th>Stock</th>");
        if (canWrite) sb.append("<th>Actions</th>");
        sb.append("</tr><tr><td>SKU-001</td><td>Widget A</td><td>150</td>");
        if (canWrite) sb.append("<td><button>Edit</button> <button>Delete</button></td>");
        sb.append("</tr></table>");
        sb.append(menu(sess));
        return sb.toString();
    }

    static String ordersPage(Map<String, Object> sess) {
        boolean canWrite = hasPermission(sess, "orders:write");
        boolean canApprove = hasPermission(sess, "orders:approve");
        StringBuilder sb = new StringBuilder("<h1>Orders</h1>");
        if (canWrite) sb.append("<button>New Order</button>");
        sb.append("<table border=1><tr><th>Order#</th><th>Customer</th><th>Status</th>");
        if (canApprove) sb.append("<th>Actions</th>");
        sb.append("</tr><tr><td>ORD-001</td><td>Acme</td><td>Pending</td>");
        if (canApprove) sb.append("<td><button>Approve</button></td>");
        sb.append("</tr></table>");
        sb.append(menu(sess));
        return sb.toString();
    }

    static String adminPage(Map<String, Object> sess) {
        return "<h1>Admin Panel</h1><p>Welcome, administrator.</p><ul><li>User Management</li><li>System Settings</li></ul>" + menu(sess);
    }

    static String forbidden(String perm) {
        return "<div style='text-align:center;padding:40px'><h1 style='color:red'>403 Access Denied</h1><p>Required: <code>" + perm + "</code></p><a href='/dashboard'>Back</a></div>";
    }

    static String menu(Map<String, Object> sess) {
        StringBuilder sb = new StringBuilder("<hr><div style='padding:8px'>");
        sb.append("<a href='/dashboard'>Dashboard</a> | ");
        if (hasPermission(sess, "orders:read")) sb.append("<a href='/orders'>Orders</a> | ");
        if (hasPermission(sess, "inventory:read")) sb.append("<a href='/inventory'>Inventory</a> | ");
        if (hasPermission(sess, "admin")) sb.append("<a href='/admin'>Admin</a> | ");
        sb.append("<a href='/auth/logout' style='color:red'>Logout</a></div>");
        return sb.toString();
    }

    interface AuthHandler { String handle(Map<String, Object> session); }

    static String requireAuth(String sessionJson, AuthHandler handler) {
        if (sessionJson == null) return redirect("/auth/login");
        Map<String, Object> sess = gson.fromJson(sessionJson, Map.class);
        return handler.handle(sess);
    }

    static String extractSession(String cookie) {
        if (cookie == null || !cookie.contains("ggid_session=")) return null;
        String sid = cookie.split("ggid_session=")[1].split(";")[0];
        return sessions.get(sid);
    }

    static String exchangeCode(String code) {
        try {
            HttpClient client = HttpClient.newHttpClient();
            String body = "grant_type=authorization_code&code=" + code + "&client_id=" + clientId + "&client_secret=" + clientSecret + "&redirect_uri=" + URLEncoder.encode(redirectUri, "UTF-8");
            HttpRequest req = HttpRequest.newBuilder()
                .uri(URI.create(ggidUrl + "/api/v1/oauth/token"))
                .header("Content-Type", "application/x-www-form-urlencoded")
                .POST(HttpRequest.BodyPublishers.ofString(body))
                .build();
            HttpResponse<String> resp = client.send(req, HttpResponse.BodyHandlers.ofString());
            Map<String, Object> m = gson.fromJson(resp.body(), Map.class);
            return (String) m.get("access_token");
        } catch (Exception e) { return null; }
    }

    @SuppressWarnings("unchecked")
    static Map<String, Object> decodeJWT(String token) {
        String[] parts = token.split("\\.");
        if (parts.length < 2) return Map.of();
        String payload = parts[1].replace("-", "+").replace("_", "/");
        while (payload.length() % 4 != 0) payload += "=";
        return gson.fromJson(new String(Base64.getDecoder().decode(payload)), Map.class);
    }

    static String html(String body) {
        return "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<!DOCTYPE html><html><body style='font-family:sans-serif;max-width:800px;margin:40px'>" + body + "</body></html>";
    }

    static String redirect(String url) {
        return "HTTP/1.1 302 Found\r\nLocation: " + url + "\r\n\r\n";
    }

    static String env(String key, String def) {
        String v = System.getenv(key);
        return v != null && !v.isEmpty() ? v : def;
    }
}
