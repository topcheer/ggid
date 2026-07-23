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
    private final JwtVerifier jwtVerifier;

    public GGIDClient(Config config) {
        this.gatewayUrl = config.gatewayUrl.replaceAll("/$", "");
        this.tenantId = config.tenantId != null ? config.tenantId : "00000000-0000-0000-0000-000000000001";
        this.apiKey = config.apiKey;
        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(Duration.ofSeconds(10))
                .readTimeout(Duration.ofSeconds(30))
                .writeTimeout(Duration.ofSeconds(10))
                .build();
        this.jwtVerifier = new JwtVerifier(gatewayUrl, null, 30);
    }

    /**
     * Verify a JWT token and return the authenticated user.
     * Uses JWKS + RS256 signature verification via JwtVerifier.
     *
     * @param token the JWT access token to verify
     * @return the authenticated user, or null if verification fails
     */
    public GGIDUser verifyUser(String token) {
        try {
            return jwtVerifier.verifyUser(token);
        } catch (Exception e) {
            return null;
        }
    }

    // -----------------------------------------------------------------------
    // Auth
    // -----------------------------------------------------------------------

    public TokenSet login(String username, String password, String clientId) throws GGIDException, IOException {
        FormBody.Builder fb = new FormBody.Builder()
                .add("grant_type", "password")
                .add("username", username)
                .add("password", password);
        if (clientId != null && !clientId.isEmpty())
            fb.add("client_id", clientId);
        FormBody formBody = fb.build();
        Request request = new Request.Builder()
                .url(gatewayUrl + "/api/v1/oauth/token")
                .header("X-Tenant-ID", tenantId)
                .header("Content-Type", "application/x-www-form-urlencoded")
                .post(formBody)
                .build();
        return execute(request, TokenSet.class);
    }

    public TokenSet refreshToken(String refreshToken) throws GGIDException, IOException {
        return post("/api/v1/auth/refresh", Map.of("refresh_token", refreshToken),
                TokenSet.class);
    }

    /**
     * Exchange client credentials for an access token (RFC 6749 §4.4).
     * Used for machine-to-machine (M2M) authentication.
     *
     * @param clientId     OAuth2 client ID
     * @param clientSecret OAuth2 client secret
     * @param scope        Optional space-delimited scopes
     * @return TokenSet with access_token
     */
    public TokenSet clientCredentials(String clientId, String clientSecret, String scope)
            throws GGIDException, IOException {
        FormBody.Builder fb = new FormBody.Builder()
                .add("grant_type", "client_credentials")
                .add("client_id", clientId)
                .add("client_secret", clientSecret);
        if (scope != null && !scope.isEmpty()) {
            fb.add("scope", scope);
        }
        FormBody formBody = fb.build();
        Request request = new Request.Builder()
                .url(gatewayUrl + "/api/v1/oauth/token")
                .header("X-Tenant-ID", tenantId)
                .header("Content-Type", "application/x-www-form-urlencoded")
                .post(formBody)
                .build();
        return execute(request, TokenSet.class);
    }

    /**
     * Exchange a SAML assertion for an access token using OAuth2 SAML2-bearer grant
     * (RFC 7522). Used by Service Providers after receiving a SAMLResponse from the IdP.
     *
     * @param samlResponse Base64-encoded SAMLResponse from the IdP
     * @param clientId     OAuth2 client ID
     * @return TokenSet with access_token
     */
    public TokenSet exchangeSAMLToken(String samlResponse, String clientId)
            throws GGIDException, IOException {
        FormBody formBody = new FormBody.Builder()
                .add("grant_type", "urn:ietf:params:oauth:grant-type:saml2-bearer")
                .add("assertion", samlResponse)
                .add("client_id", clientId)
                .build();
        Request request = new Request.Builder()
                .url(gatewayUrl + "/api/v1/oauth/token")
                .header("X-Tenant-ID", tenantId)
                .header("Content-Type", "application/x-www-form-urlencoded")
                .post(formBody)
                .build();
        return execute(request, TokenSet.class);
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
    // Policy: RBAC + ABAC
    // -----------------------------------------------------------------------

    /**
     * Check if a user has permission to perform an action on a resource.
     * Calls POST /api/v1/policies/check with the user's token.
     *
     * @param token       Access token (Bearer)
     * @param userId      User UUID (from JWT sub claim)
     * @param resourceType  Resource type (e.g. "inventory", "orders", "invoices")
     * @param action      Action (e.g. "read", "write", "delete", "approve")
     * @return PolicyResult with allowed flag and reason
     */
    public PolicyResult checkPermission(String token, String userId, String resourceType, String action)
            throws GGIDException, IOException {
        java.util.Map<String, Object> body = new java.util.HashMap<>();
        body.put("user_id", userId);
        body.put("resource_type", resourceType);
        body.put("action", action);
        Request request = buildRequest("POST", "/api/v1/policies/check", body)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, PolicyResult.class);
    }

    /**
     * Check if the current token holder has permission.
     * Extracts user_id from JWT automatically.
     */
    public PolicyResult checkPermission(String token, String resourceType, String action)
            throws GGIDException, IOException {
        // Extract user_id from JWT sub claim
        String userId = extractUserIdFromToken(token);
        return checkPermission(token, userId, resourceType, action);
    }

    /**
     * Full ABAC policy evaluation with context attributes.
     * Calls POST /api/v1/policies/abac/evaluate
     */
    public PolicyResult checkPolicy(String token, PolicyCheckRequest req)
            throws GGIDException, IOException {
        Request request = buildRequest("POST", "/api/v1/policies/abac/evaluate", req)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, PolicyResult.class);
    }

    /**
     * Assign a role to a user.
     * Calls POST /api/v1/policies/roles/{roleId}/users/{userId}
     */
    public void assignRole(String token, String userId, String roleId)
            throws GGIDException, IOException {
        String path = "/api/v1/policies/roles/" + roleId + "/users/" + userId;
        Request request = buildRequest("POST", path, Map.of())
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        execute(request, Void.class);
    }

    /**
     * Revoke a role from a user.
     * Calls DELETE /api/v1/policies/roles/{roleId}/users/{userId}
     */
    public void revokeRole(String token, String userId, String roleId)
            throws GGIDException, IOException {
        String path = "/api/v1/policies/roles/" + roleId + "/users/" + userId;
        Request request = buildRequest("DELETE", path, null)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        execute(request, Void.class);
    }

    /**
     * Get all roles assigned to a user.
     * Calls GET /api/v1/policies/users/{userId}/roles
     */
    public List<Role> getUserRoles(String token, String userId)
            throws GGIDException, IOException {
        String path = "/api/v1/policies/users/" + userId + "/roles";
        Request request = buildRequest("GET", path, null)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, new TypeReference<List<Role>>() {});
    }

    /**
     * List all permissions in a tree structure.
     * Calls GET /api/v1/policies/permissions/tree
     */
    public List<Permission> listPermissions(String token)
            throws GGIDException, IOException {
        Request request = buildRequest("GET", "/api/v1/policies/permissions/tree", null)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, new TypeReference<List<Permission>>() {});
    }

    /**
     * Evaluate ABAC conditions against attributes.
     * Calls POST /api/v1/policies/abac/evaluate
     */
    public ABACEvalResult evaluateABAC(String token, ABACEvalRequest req)
            throws GGIDException, IOException {
        Request request = buildRequest("POST", "/api/v1/policies/abac/evaluate", req)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, ABACEvalResult.class);
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

    /**
     * Extract user_id (sub claim) from a JWT without verification.
     * Used internally for permission checks.
     */
    private String extractUserIdFromToken(String token) {
        try {
            String[] parts = token.split("\\.");
            if (parts.length < 2) return "";
            String payload = new String(java.util.Base64.getUrlDecoder().decode(parts[1]));
            @SuppressWarnings("unchecked")
            java.util.Map<String, Object> claims = mapper.readValue(payload, java.util.Map.class);
            Object sub = claims.get("sub");
            return sub != null ? sub.toString() : "";
        } catch (Exception e) {
            return "";
        }
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
        RequestBody reqBody = null;
        if (body != null && !body.equals(Void.class)) {
            reqBody = RequestBody.create(mapper.writeValueAsString(body), JSON);
        }
        // For GET/DELETE with no body, pass null to avoid OkHttp IllegalArgumentException
        Request.Builder builder = new Request.Builder()
                .url(gatewayUrl + path)
                .header("X-Tenant-ID", tenantId);

        if (reqBody != null) {
            builder.header("Content-Type", "application/json");
        }

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
    // Agent Identity
    // -----------------------------------------------------------------------

    public Agent registerAgent(String token, String name, String agentType,
                                String ownerUserId, List<String> allowedScopes)
            throws GGIDException, IOException {
        java.util.Map<String, Object> body = new java.util.HashMap<>();
        body.put("name", name);
        body.put("type", agentType);
        body.put("owner_user_id", ownerUserId);
        body.put("allowed_scopes", allowedScopes);
        Request request = buildRequest("POST", "/api/v1/agents/register", body)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, Agent.class);
    }

    public List<Agent> listAgents(String token) throws GGIDException, IOException {
        Request request = buildRequest("GET", "/api/v1/agents", null)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        java.util.Map<String, Object> resp = execute(request, java.util.Map.class);
        Object items = resp.get("agents");
        if (items == null) return java.util.Collections.emptyList();
        return mapper.convertValue(items,
                mapper.getTypeFactory().constructCollectionType(List.class, Agent.class));
    }

    public AgentTokenResponse exchangeAgentToken(String agentId, String subjectToken,
                                                  List<String> scopes)
            throws GGIDException, IOException {
        java.util.Map<String, Object> body = new java.util.HashMap<>();
        body.put("agent_id", agentId);
        body.put("subject_token", subjectToken);
        body.put("scope", scopes);
        Request request = buildRequest("POST", "/api/v1/agents/token", body);
        return execute(request, AgentTokenResponse.class);
    }

    @SuppressWarnings("unchecked")
    public java.util.Map<String, Object> verifyAgentToken(String token) throws GGIDException, IOException {
        Request request = buildRequest("POST", "/api/v1/agents/verify",
                java.util.Map.of("token", token));
        return execute(request, java.util.Map.class);
    }

    // -----------------------------------------------------------------------
    // Access Request (IGA)
    // -----------------------------------------------------------------------

    @SuppressWarnings("unchecked")
    public java.util.Map<String, Object> createAccessRequest(String token, String userId,
                                                               String resource, String action,
                                                               String reason)
            throws GGIDException, IOException {
        java.util.Map<String, Object> body = new java.util.HashMap<>();
        body.put("user_id", userId);
        body.put("resource", resource);
        body.put("action", action);
        body.put("reason", reason);
        Request request = buildRequest("POST", "/api/v1/access-requests", body)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, java.util.Map.class);
    }

    @SuppressWarnings("unchecked")
    public List<java.util.Map<String, Object>> listAccessRequests(String token)
            throws GGIDException, IOException {
        Request request = buildRequest("GET", "/api/v1/access-requests", null)
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        java.util.Map<String, Object> resp = execute(request, java.util.Map.class);
        Object items = resp.get("requests");
        if (items == null) items = resp.get("data");
        if (items == null) return java.util.Collections.emptyList();
        return (List<java.util.Map<String, Object>>) items;
    }

    @SuppressWarnings("unchecked")
    public java.util.Map<String, Object> approveAccessRequest(String token, String requestId,
                                                                String comment)
            throws GGIDException, IOException {
        Request request = buildRequest("POST", "/api/v1/access-requests/" + requestId + "/approve",
                java.util.Map.of("comment", comment))
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, java.util.Map.class);
    }

    @SuppressWarnings("unchecked")
    public java.util.Map<String, Object> rejectAccessRequest(String token, String requestId,
                                                               String comment)
            throws GGIDException, IOException {
        Request request = buildRequest("POST", "/api/v1/access-requests/" + requestId + "/reject",
                java.util.Map.of("comment", comment))
                .newBuilder()
                .header("Authorization", "Bearer " + token)
                .build();
        return execute(request, java.util.Map.class);
    }

    // -----------------------------------------------------------------------
    // Webhooks
    // -----------------------------------------------------------------------

    public DiscoveryConfig getDiscovery() throws GGIDException, IOException {
        Request request = buildRequest("GET", "/.well-known/openid-configuration", null);
        return execute(request, DiscoveryConfig.class);
    }

    // -----------------------------------------------------------------------
    // Webhook Management
    // -----------------------------------------------------------------------

    public List<Webhook> listWebhooks(String token) throws GGIDException, IOException {
        Request request = buildRequest("GET", "/api/v1/webhooks", null);
        if (token != null && !token.isEmpty()) {
            request = request.newBuilder().header("Authorization", "Bearer " + token).build();
        }
        java.util.Map<String, Object> resp = execute(request, java.util.Map.class);
        Object items = resp.get("webhooks");
        if (items == null) items = resp.get("data");
        if (items == null) return java.util.Collections.emptyList();
        return mapper.convertValue(items,
            mapper.getTypeFactory().constructCollectionType(List.class, Webhook.class));
    }

    public Webhook createWebhook(String token, Webhook webhook) throws GGIDException, IOException {
        Request request = buildRequest("POST", "/api/v1/webhooks", webhook);
        if (token != null && !token.isEmpty()) {
            request = request.newBuilder().header("Authorization", "Bearer " + token).build();
        }
        return execute(request, Webhook.class);
    }

    public void deleteWebhook(String token, String webhookId) throws GGIDException, IOException {
        Request request = buildRequest("DELETE", "/api/v1/webhooks?id=" + webhookId, null);
        if (token != null && !token.isEmpty()) {
            request = request.newBuilder().header("Authorization", "Bearer " + token).build();
        }
        execute(request, Void.class);
    }

    // -----------------------------------------------------------------------
    // Token Introspection (RFC 7662)
    // -----------------------------------------------------------------------

    public IntrospectionResult introspectToken(String token, String tokenToCheck)
            throws GGIDException, IOException {
        java.util.Map<String, String> body = java.util.Map.of(
            "token", tokenToCheck,
            "token_type_hint", "access_token"
        );
        Request request = buildRequest("POST", "/api/v1/oauth/introspect", body);
        if (token != null && !token.isEmpty()) {
            request = request.newBuilder().header("Authorization", "Bearer " + token).build();
        }
        return execute(request, IntrospectionResult.class);
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
