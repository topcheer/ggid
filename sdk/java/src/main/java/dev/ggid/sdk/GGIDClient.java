package dev.ggid.sdk;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.*;

/**
 * GGID IAM Platform API client.
 * 
 * Usage:
 *   GGIDClient client = GGIDClient.builder()
 *       .gatewayUrl("https://iam.example.com")
 *       .tenantId("00000000-0000-0000-0000-000000000001")
 *       .build();
 */
public class GGIDClient {
    
    private final String gatewayUrl;
    private final String tenantId;
    private final HttpClient httpClient;
    private final ObjectMapper mapper;
    
    private GGIDClient(Builder builder) {
        this.gatewayUrl = builder.gatewayUrl;
        this.tenantId = builder.tenantId;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(10))
                .build();
        this.mapper = new ObjectMapper();
    }
    
    public static Builder builder() {
        return new Builder();
    }
    
    // --- Auth ---
    
    public TokenSet login(String username, String password) throws GGIDException {
        Map<String, String> body = Map.of("username", username, "password", password);
        return post("/api/v1/auth/login", body, "", TokenSet.class);
    }
    
    public String register(String username, String email, String password, String name) throws GGIDException {
        Map<String, String> body = new HashMap<>();
        body.put("username", username);
        body.put("email", email);
        body.put("password", password);
        body.put("name", name != null ? name : "");
        Map<String, Object> resp = post("/api/v1/auth/register", body, "", Map.class);
        return (String) resp.get("user_id");
    }
    
    // --- Users ---
    
    @SuppressWarnings("unchecked")
    public List<User> listUsers(String token) throws GGIDException {
        Map<String, Object> resp = get("/api/v1/users", token, Map.class);
        List<Map<String, Object>> usersData = (List<Map<String, Object>>) resp.get("users");
        if (usersData == null) usersData = (List<Map<String, Object>>) resp.get("");
        if (usersData == null) return Collections.emptyList();
        
        List<User> users = new ArrayList<>();
        for (Map<String, Object> data : usersData) {
            users.add(mapper.convertValue(data, User.class));
        }
        return users;
    }
    
    public User getUser(String token, String userId) throws GGIDException {
        return get("/api/v1/users/" + userId, token, User.class);
    }
    
    public void deleteUser(String token, String userId) throws GGIDException {
        request("DELETE", "/api/v1/users/" + userId, null, token);
    }
    
    // --- RBAC ---
    
    @SuppressWarnings("unchecked")
    public List<Role> listRoles(String token) throws GGIDException {
        Map<String, Object> resp = get("/api/v1/roles", token, Map.class);
        List<Map<String, Object>> rolesData = (List<Map<String, Object>>) resp.get("roles");
        if (rolesData == null) return Collections.emptyList();
        
        List<Role> roles = new ArrayList<>();
        for (Map<String, Object> data : rolesData) {
            roles.add(mapper.convertValue(data, Role.class));
        }
        return roles;
    }
    
    public PolicyResult checkPermission(String token, String resource, String action) throws GGIDException {
        Map<String, String> body = Map.of("resource", resource, "action", action);
        return post("/api/v1/policies/check", body, token, PolicyResult.class);
    }
    
    // --- Internal HTTP ---
    
    private <T> T get(String path, String token, Class<T> responseType) throws GGIDException {
        return request("GET", path, null, token, responseType);
    }
    
    private <T> T post(String path, Object body, String token, Class<T> responseType) throws GGIDException {
        return request("POST", path, body, token, responseType);
    }
    
    private void request(String method, String path, Object body, String token) throws GGIDException {
        request(method, path, body, token, Map.class);
    }
    
    @SuppressWarnings("unchecked")
    private <T> T request(String method, String path, Object body, String token, Class<T> responseType) throws GGIDException {
        try {
            String jsonBody = body != null ? mapper.writeValueAsString(body) : "";
            
            HttpRequest.Builder reqBuilder = HttpRequest.newBuilder()
                    .uri(URI.create(gatewayUrl + path))
                    .header("X-Tenant-ID", tenantId)
                    .header("Content-Type", "application/json")
                    .timeout(Duration.ofSeconds(30));
            
            if (token != null && !token.isEmpty()) {
                reqBuilder.header("Authorization", "Bearer " + token);
            }
            
            if ("GET".equals(method)) {
                reqBuilder.GET();
            } else if ("DELETE".equals(method)) {
                reqBuilder.DELETE();
            } else {
                reqBuilder.method(method, HttpRequest.BodyPublishers.ofString(jsonBody));
            }
            
            HttpResponse<String> response = httpClient.send(reqBuilder.build(), HttpResponse.BodyHandlers.ofString());
            
            if (response.statusCode() >= 400) {
                throw new GGIDException("API error: " + response.statusCode() + " " + response.body());
            }
            
            if (responseType == Map.class) {
                return (T) mapper.readValue(response.body(), Map.class);
            }
            return mapper.readValue(response.body(), responseType);
            
        } catch (GGIDException e) {
            throw e;
        } catch (Exception e) {
            throw new GGIDException("Request failed: " + e.getMessage(), e);
        }
    }
    
    // --- Builder ---
    
    public static class Builder {
        private String gatewayUrl;
        private String tenantId = "00000000-0000-0000-0000-000000000001";
        
        public Builder gatewayUrl(String url) {
            this.gatewayUrl = url.replaceAll("/$", "");
            return this;
        }
        
        public Builder tenantId(String id) {
            this.tenantId = id;
            return this;
        }
        
        public GGIDClient build() {
            if (gatewayUrl == null || gatewayUrl.isEmpty()) {
                throw new IllegalArgumentException("gatewayUrl is required");
            }
            return new GGIDClient(this);
        }
    }
}
