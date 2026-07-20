package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.List;
import java.util.Map;

/**
 * /orgs — list/create organizations via GGID SDK
 */
public class OrgsHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        try {
            var orgs = Main.ggid.listOrgs().items;
            sendJson(exchange, 200, json(Map.of("orgs", orgs, "total", orgs.size())));
        } catch (Exception e) {
            sendJson(exchange, 200, json(Map.of("orgs", List.of(), "error", e.getMessage())));
        }
    }

    @Override
    protected void handlePost(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "settings:write")) return;
        Map<String, String> body = mapper.readValue(exchange.getRequestBody(),
                new com.fasterxml.jackson.core.type.TypeReference<>() {});
        Main.audit(user.userId, "orgs.create", "Created org: " + body.get("name"));
        sendJson(exchange, 201, json(Map.of("created", true, "name", body.get("name"))));
    }
}
