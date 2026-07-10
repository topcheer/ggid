package dev.ggid.sdk;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import okhttp3.*;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;

/**
 * GGID IAM Platform Java SDK client.
 *
 * Usage:
 *   GGIDClient client = new GGIDClient("https://iam.example.com");
 *   TokenSet tokens = client.login("admin", "Admin@123456");
 *   JsonNode users = client.listUsers(tokens.getAccessToken());
 */
public class GGIDClient {
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    private final String gatewayUrl;
    private final String tenantId;
    private final OkHttpClient httpClient;
    private final ObjectMapper mapper;

    public GGIDClient(String gatewayUrl) {
        this(gatewayUrl, "00000000-0000-0000-0000-000000000001");
    }

    public GGIDClient(String gatewayUrl, String tenantId) {
        this.gatewayUrl = gatewayUrl.replaceAll("/+$", "");
        this.tenantId = tenantId;
        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(10, java.util.concurrent.TimeUnit.SECONDS)
                .readTimeout(30, java.util.concurrent.TimeUnit.SECONDS)
                .build();
        this.mapper = new ObjectMapper();
    }

    // --- Auth ---

    public TokenSet login(String username, String password) throws GGIDException {
        Map<String, String> body = new HashMap<>();
        body.put("username", username);
        body.put("password", password);
        JsonNode resp = post("/api/v1/auth/login", body, null);
        return new TokenSet(
                resp.path("access_token").asText(),
                resp.path("refresh_token").asText(""),
                resp.path("token_type").asText("Bearer"),
                resp.path("expires_in").asInt(3600)
        );
    }

    public String register(String username, String email, String password, String name) throws GGIDException {
        Map<String, String> body = new HashMap<>();
        body.put("username", username);
        body.put("email", email);
        body.put("password", password);
        body.put("name", name != null ? name : "");
        JsonNode resp = post("/api/v1/auth/register", body, null);
        return resp.path("user_id").asText("");
    }

    // --- Users ---

    public JsonNode listUsers(String token) throws GGIDException {
        return get("/api/v1/users", token);
    }

    public JsonNode getUser(String token, String userId) throws GGIDException {
        return get("/api/v1/users/" + userId, token);
    }

    public void deleteUser(String token, String userId) throws GGIDException {
        delete("/api/v1/users/" + userId, token);
    }

    // --- RBAC ---

    public JsonNode listRoles(String token) throws GGIDException {
        return get("/api/v1/roles", token);
    }

    public JsonNode checkPermission(String token, String resource, String action) throws GGIDException {
        Map<String, String> body = new HashMap<>();
        body.put("resource", resource);
        body.put("action", action);
        return post("/api/v1/policies/check", body, token);
    }

    // --- Internal HTTP ---

    private JsonNode get(String path, String token) throws GGIDException {
        return execute("GET", path, null, token);
    }

    private JsonNode post(String path, Object body, String token) throws GGIDException {
        return execute("POST", path, body, token);
    }

    private void delete(String path, String token) throws GGIDException {
        execute("DELETE", path, null, token);
    }

    private JsonNode execute(String method, String path, Object body, String token) throws GGIDException {
        try {
            Request.Builder reqBuilder = new Request.Builder()
                    .url(gatewayUrl + path)
                    .header("X-Tenant-ID", tenantId)
                    .header("Content-Type", "application/json");

            if (token != null && !token.isEmpty()) {
                reqBuilder.header("Authorization", "Bearer " + token);
            }

            if (body != null) {
                String json = mapper.writeValueAsString(body);
                reqBuilder.method(method, RequestBody.create(json, JSON));
            } else {
                reqBuilder.method(method, null);
            }

            try (Response response = httpClient.newCall(reqBuilder.build()).execute()) {
                String respBody = response.body() != null ? response.body().string() : "{}";
                if (response.code() >= 400) {
                    throw new GGIDException("API error: " + response.code() + " - " + respBody);
                }
                return mapper.readTree(respBody);
            }
        } catch (IOException e) {
            throw new GGIDException("Request failed: " + e.getMessage(), e);
        }
    }
}
