// OAuth Demo — Java SDK
// Run: GGID_URL=https://ggid.iot2.win CLIENT_ID=xxx CLIENT_SECRET=xxx java OAuthDemo
import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpExchange;
import java.io.*;
import java.net.*;
import java.nio.charset.StandardCharsets;

public class OAuthDemo {
    static String ggidUrl = System.getenv().getOrDefault("GGID_URL", "http://localhost:8080");
    static String clientId = System.getenv().getOrDefault("CLIENT_ID", "");
    static String clientSecret = System.getenv().getOrDefault("CLIENT_SECRET", "");
    static String redirectURI = System.getenv().getOrDefault("REDIRECT_URI", "http://localhost:9096/callback");
    static String tenantId = System.getenv().getOrDefault("TENANT_ID", "00000000-0000-0000-0000-000000000001");

    public static void main(String[] args) throws Exception {
        int port = Integer.parseInt(System.getenv().getOrDefault("PORT", "9096"));
        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/", new HomeHandler());
        server.createContext("/login", new LoginHandler());
        server.createContext("/callback", new CallbackHandler());
        server.start();
        System.out.println("OAuth demo on :" + port + " (GGID: " + ggidUrl + ")");
    }

    static class HomeHandler implements HttpHandler {
        public void handle(HttpExchange t) throws IOException {
            String user = extractParam(t.getRequestURI().getQuery(), "user");
            String html = user.isEmpty()
                ? "<h1>GGID OAuth Demo</h1><a href='/login'>Login with GGID</a>"
                : "<h1>GGID OAuth Demo</h1><pre>" + user + "</pre>";
            t.sendResponseHeaders(200, html.length());
            t.getResponseBody().write(html.getBytes());
            t.getResponseBody().close();
        }
    }

    static class LoginHandler implements HttpHandler {
        public void handle(HttpExchange t) throws IOException {
            String authUrl = ggidUrl + "/api/v1/oauth/authorize?response_type=code&client_id="
                + clientId + "&redirect_uri=" + URLEncoder.encode(redirectURI, "UTF-8")
                + "&scope=openid+profile&state=demo123";
            t.sendResponseHeaders(302, 0);
            t.getResponseHeaders().set("Location", authUrl);
            t.getResponseBody().close();
        }
    }

    static class CallbackHandler implements HttpHandler {
        public void handle(HttpExchange t) throws IOException {
            String code = extractParam(t.getRequestURI().getQuery(), "code");
            // Token exchange via HTTP POST (simplified)
            String tokenUrl = ggidUrl + "/api/v1/oauth/token";
            String formData = "grant_type=authorization_code&code=" + code
                + "&redirect_uri=" + URLEncoder.encode(redirectURI, "UTF-8")
                + "&client_id=" + clientId + "&client_secret=" + clientSecret;

            HttpURLConnection conn = (HttpURLConnection) new URL(tokenUrl).openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/x-www-form-urlencoded");
            conn.setRequestProperty("X-Tenant-ID", tenantId);
            conn.setDoOutput(true);
            conn.getOutputStream().write(formData.getBytes());

            BufferedReader reader = new BufferedReader(new InputStreamReader(conn.getInputStream()));
            StringBuilder resp = new StringBuilder();
            String line;
            while ((line = reader.readLine()) != null) resp.append(line);
            reader.close();

            // Simplified: redirect home with response
            String redirect = "/?user=" + URLEncoder.encode(resp.toString(), "UTF-8");
            t.sendResponseHeaders(302, 0);
            t.getResponseHeaders().set("Location", redirect);
            t.getResponseBody().close();
        }
    }

    static String extractParam(String query, String key) {
        if (query == null) return "";
        for (String pair : query.split("&")) {
            String[] kv = pair.split("=", 2);
            if (kv.length == 2 && kv[0].equals(key)) {
                try { return URLDecoder.decode(kv[1], "UTF-8"); }
                catch (Exception e) { return kv[1]; }
            }
        }
        return "";
    }
}
