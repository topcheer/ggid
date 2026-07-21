package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import java.io.IOException;

/**
 * Simple health check handler — no auth required.
 */
public class HealthHandler implements HttpHandler {
    @Override
    public void handle(HttpExchange exchange) throws IOException {
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        String json = "{\"status\":\"ok\",\"app\":\"ERP Java Demo\",\"auth\":\"SAML 2.0 SSO\",\"tenant_id\":\""
                + Main.TENANT_ID + "\"}";
        byte[] resp = json.getBytes(java.nio.charset.StandardCharsets.UTF_8);
        exchange.sendResponseHeaders(200, resp.length);
        try (java.io.OutputStream os = exchange.getResponseBody()) { os.write(resp); }
    }
}
