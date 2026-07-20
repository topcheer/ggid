package dev.ggid.erp;

import com.sun.net.httpserver.HttpExchange;
import dev.ggid.sdk.GGIDUser;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * /audit — view audit logs (requires audit:read)
 */
public class AuditHandler extends BaseHandler {
    @Override
    protected void handleGet(HttpExchange exchange) throws IOException {
        GGIDUser user = requireAuth(exchange);
        if (user == null) return;
        if (!requirePermission(exchange, user, "audit:read")) return;

        List<AuditLog> logs = new ArrayList<>(Main.auditLogs.values());
        sendJson(exchange, 200, json(Map.of("audit_logs", logs, "total", logs.size())));
    }
}
