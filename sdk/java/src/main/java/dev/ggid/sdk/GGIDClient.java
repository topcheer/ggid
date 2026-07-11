package dev.ggid.sdk;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import okhttp3.*;

import java.io.IOException;
import java.time.Duration;
import java.util.List;
import java.util.Map;

/**
 * GGID IAM Platform Java SDK client.
 * Provides user management, authentication, RBAC, and organization APIs.
 */
public class GGIDClient {

    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");
    private static final ObjectMapper mapper = new ObjectMapper();

    private final String gatewayUrl;
    private final String tenantId;
    private final String apiKey;
    private final OkHttpClient httpClient;

    public GGIDClient(Config config) {
        this.gatewayUrl = config.gatewayUrl.replaceAll("/$", "");
        this.tenantId = config.tenantId != null ? config.tenantId : "00000000-0000-0000-0000-000000000001";
        this.apiKey = config.apiKey;
        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(Duration.ofSeconds(10))
                .readTimeout(Duration.ofSeconds(30))
                .writeTimeout(Duration.ofSeconds(10))
                .build();
    }

    // -----------------------------------------------------------------------
    // Auth
    // -----------------------------------------------------------------------

    public TokenSet login(String username, String password) throws GGIDException, IOException {
        return post("/api/v1/auth/login", Map.of("username", username, "password", password),
                TokenSet.class);
    }

    public TokenSet refreshToken(String refreshToken) throws GGIDException, IOException {
        return post("/api/v1/auth/refresh", Map.of("refresh_token", refreshToken),
                TokenSet.class);
    }

    public void logout(String accessToken) throws GGIDException, IOException {
        post("/api/v1/auth/logout", Map.of("access_token", accessToken), Void.class);
    }

    // -----------------------------------------------------------------------
    // Users
    // -----------------------------------------------------------------------

    public User createUser(String username, String email, String password)
            throws GGIDException, IOException {
        return post("/api/v1/users", Map.of("username", username, "email", email,
                "password", password), User.class);
    }

    public User getUser(String userId) throws GGIDException, IOException {
        return get("/api/v1/users/" + userId, User.class);
    }

    public User updateUser(String userId, String email, String phone)
            throws GGIDException, IOException {
        java.util.Map<String, String> body = new java.util.HashMap<>();
        if (email != null && !email.isEmpty()) body.put("email", email);
        if (phone != null && !phone.isEmpty()) body.put("phone", phone);
        return patch("/api/v1/users/" + userId, body, User.class);
    }

    public void deleteUser(String userId) throws GGIDException, IOException {
        delete("/api/v1/users/" + userId);
    }

    public PageResult<User> listUsers(int page, int pageSize) throws GGIDException, IOException {
        return get("/api/v1/users?page=" + page + "&page_size=" + pageSize,
                new TypeReference<PageResult<User>>() {});
    }

    public void assignRole(String userId, String roleId) throws GGIDException, IOException {
        post("/api/v1/users/" + userId + "/roles", Map.of("role_id", roleId), Void.class);
    }

    // -----------------------------------------------------------------------
    // Roles
    // -----------------------------------------------------------------------

    public Role createRole(String key, String name) throws GGIDException, IOException {
        return post("/api/v1/roles", Map.of("key", key, "name", name), Role.class);
    }

    public PageResult<Role> listRoles() throws GGIDException, IOException {
        return get("/api/v1/roles", new TypeReference<PageResult<Role>>() {});
    }

    // -----------------------------------------------------------------------
    // Organizations
    // -----------------------------------------------------------------------

    public Organization createOrg(String name) throws GGIDException, IOException {
        return post("/api/v1/organizations", Map.of("name", name), Organization.class);
    }

    public PageResult<Organization> listOrgs() throws GGIDException, IOException {
        return get("/api/v1/organizations", new TypeReference<PageResult<Organization>>() {});
    }

    // -----------------------------------------------------------------------
    // Policy
    // -----------------------------------------------------------------------

    public PermissionResult checkPermission(String userId, String resource, String action)
            throws GGIDException, IOException {
        return post("/api/v1/policies/check", Map.of("user_id", userId,
                "resource", resource, "action", action), PermissionResult.class);
    }

    // -----------------------------------------------------------------------
    // OAuth/OIDC
    // -----------------------------------------------------------------------

    @SuppressWarnings("unchecked")
    public Map<String, Object> getOIDCDiscovery() throws GGIDException, IOException {
        return get("/.well-known/openid-configuration", Map.class);
    }

    @SuppressWarnings("unchecked")
    public Map<String, Object> getJWKS() throws GGIDException, IOException {
        return get("/oauth/jwks", Map.class);
    }

    @SuppressWarnings("unchecked")
    public Map<String, Object> getUserInfo(String accessToken) throws GGIDException, IOException {
        Request request = buildRequest("GET", "/oauth/userinfo", null)
                .newBuilder()
                .header("Authorization", "Bearer " + accessToken)
                .build();
        return execute(request, Map.class);
    }

    @SuppressWarnings("unchecked")
    public Map<String, Object> registerOAuthClient(String clientName, List<String> redirectUris,
                                                     List<String> grantTypes, String scope)
            throws GGIDException, IOException {
        return post("/api/v1/oauth/register", Map.of(
                "client_name", clientName,
                "redirect_uris", redirectUris,
                "grant_types", grantTypes != null ? grantTypes : List.of("authorization_code"),
                "scope", scope != null ? scope : ""
        ), Map.class);
    }

    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> listOAuthClients(String accessToken) throws GGIDException, IOException {
        Request request = buildRequest("GET", "/api/v1/oauth/clients", null)
                .newBuilder()
                .header("Authorization", "Bearer " + accessToken)
                .build();
        return execute(request, new TypeReference<List<Map<String, Object>>>() {});
    }

    public void deleteOAuthClient(String accessToken, String clientId) throws GGIDException, IOException {
        Request request = buildRequest("DELETE", "/api/v1/oauth/clients/" + clientId, null)
                .newBuilder()
                .header("Authorization", "Bearer " + accessToken)
                .build();
        execute(request, Void.class);
    }

    public void revokeToken(String token) throws GGIDException, IOException {
        post("/api/v1/oauth/revoke", Map.of("token", token), Void.class);
    }

    @SuppressWarnings("unchecked")
    public Map<String, Object> deviceAuthorization(String clientId, String scope)
            throws GGIDException, IOException {
        return post("/api/v1/oauth/device_authorization", Map.of(
                "client_id", clientId, "scope", scope), Map.class);
    }

    public String buildAuthorizeURL(String clientId, String redirectUri, String responseType,
                                     String scope, String state, String nonce,
                                     String codeChallenge, String codeChallengeMethod) {
        StringBuilder url = new StringBuilder(gatewayUrl)
                .append("/oauth/authorize?")
                .append("client_id=").append(clientId)
                .append("&redirect_uri=").append(redirectUri)
                .append("&response_type=").append(responseType);
        if (scope != null && !scope.isEmpty()) url.append("&scope=").append(scope);
        if (state != null && !state.isEmpty()) url.append("&state=").append(state);
        if (nonce != null && !nonce.isEmpty()) url.append("&nonce=").append(nonce);
        if (codeChallenge != null && !codeChallenge.isEmpty()) {
            url.append("&code_challenge=").append(codeChallenge);
            url.append("&code_challenge_method=")
              .append(codeChallengeMethod != null ? codeChallengeMethod : "S256");
        }
        return url.toString();
    }

    // -----------------------------------------------------------------------
    // Internal HTTP helpers
    // -----------------------------------------------------------------------

    private <T> T get(String path, Class<T> type) throws GGIDException, IOException {
        Request request = buildRequest("GET", path, null);
        return execute(request, type);
    }

    private <T> T get(String path, TypeReference<T> typeRef) throws GGIDException, IOException {
        Request request = buildRequest("GET", path, null);
        return execute(request, typeRef);
    }

    private <T> T post(String path, Object body, Class<T> type) throws GGIDException, IOException {
        Request request = buildRequest("POST", path, body);
        return execute(request, type);
    }

    private <T> T patch(String path, Object body, Class<T> type) throws GGIDException, IOException {
        Request request = buildRequest("PATCH", path, body);
        return execute(request, type);
    }

    private void delete(String path) throws GGIDException, IOException {
        Request request = buildRequest("DELETE", path, null);
        execute(request, Void.class);
    }

    private Request buildRequest(String method, String path, Object body) throws IOException {
        RequestBody reqBody = body != null && !body.equals(Void.class)
                ? RequestBody.create(mapper.writeValueAsString(body), JSON)
                : RequestBody.create("", null);

        Request.Builder builder = new Request.Builder()
                .url(gatewayUrl + path)
                .header("X-Tenant-ID", tenantId)
                .header("Content-Type", "application/json");

        if (apiKey != null && !apiKey.isEmpty()) {
            builder.header("X-API-Key", apiKey);
        }

        return builder.method(method, reqBody).build();
    }

    @SuppressWarnings("unchecked")
    private <T> T execute(Request request, Class<T> type) throws GGIDException, IOException {
        try (Response response = httpClient.newCall(request).execute()) {
            String bodyStr = response.body() != null ? response.body().string() : "";
            if (!response.isSuccessful()) {
                String code = "";
                String message = bodyStr;
                try {
                    Map<String, Object> parsed = mapper.readValue(bodyStr, Map.class);
                    code = (String) parsed.getOrDefault("code", "");
                    message = (String) parsed.getOrDefault("message", bodyStr);
                } catch (Exception ignored) {}
                throw new GGIDException(response.code(), message, code);
            }
            if (type == Void.class || bodyStr.isEmpty()) return null;
            return mapper.readValue(bodyStr, type);
        }
    }

    private <T> T execute(Request request, TypeReference<T> typeRef) throws GGIDException, IOException {
        try (Response response = httpClient.newCall(request).execute()) {
            String bodyStr = response.body() != null ? response.body().string() : "";
            if (!response.isSuccessful()) {
                throw new GGIDException(response.code(), bodyStr, "");
            }
            if (bodyStr.isEmpty()) return null;
            return mapper.readValue(bodyStr, typeRef);
        }
    }

    // -----------------------------------------------------------------------
    // Inner classes
    // -----------------------------------------------------------------------

    public static class Config {
        public String gatewayUrl;
        public String tenantId;
        public String apiKey;

        public Config(String gatewayUrl) {
            this.gatewayUrl = gatewayUrl;
        }
    }

}
