package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import com.fasterxml.jackson.databind.ObjectMapper;
import dev.ggid.sdk.TokenSet;
import java.io.IOException;
import java.util.Map;

/**
 * POST /auth/login — username/password → JWT token with permissions
 */
public class AuthHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        sendJson(exchange, 200, json(Map.of("message", "POST to /auth/login with username+password")));
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        Map<String, String> body = mapper.readValue(exchange.getRequestBody(),
                new com.fasterxml.jackson.core.type.TypeReference<>() {});
        try {
            TokenSet tokens = Main.ggid.login(body.get("username"), body.get("password"));
            Main.audit(body.get("username"), "auth.login", "User logged in");
            sendJson(exchange, 200, mapper.writeValueAsString(tokens));
        } catch (Exception e) {
            sendJson(exchange, 401, json(Map.of("error", "Login failed: " + e.getMessage())));
        }
    }
}
