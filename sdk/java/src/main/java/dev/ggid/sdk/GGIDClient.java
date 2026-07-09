package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.List;
import java.util.Map;

/**
 * GGID IAM SDK client for Java.
 * Integrate GGID identity and access management into Java backends.
 */
public class GGIDClient {

    private final String baseURL;
    private final String apiKey;
    private final HttpClient httpClient;
    private final ObjectMapper mapper;

    private GGIDClient(Builder builder) {
        this.baseURL = builder.baseURL.replaceAll("/$", "");
        this.apiKey = builder.apiKey;
        this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofSeconds(10))
                .build();
        this.mapper = new ObjectMapper()
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    /**
     * Check if a user has permission for an action on a resource.
     */
    public boolean checkPermission(String userId, String resource, String action) throws Exception {
        Map<String, String> body = Map.of("user_id", userId, "resource", resource, "action", action);
        PermissionResult result = post("/api/v1/policies/check", body, PermissionResult.class);
        return result.allowed;
    }

    /**
     * Create a new user (requires API key).
     */
    public User createUser(CreateUserRequest req) throws Exception {
        return post("/api/v1/users", req, User.class);
    }

    /**
     * Get a user by ID.
     */
    public User getUser(String userId) throws Exception {
        return get("/api/v1/users/" + userId, User.class);
    }

    /**
     * List users with pagination.
     */
    public PageResult<User> listUsers(int page, int pageSize) throws Exception {
        String path = "/api/v1/users?page=" + page + "&page_size=" + pageSize;
        return get(path, mapper.getTypeFactory().constructType(PageResult.class));
    }

    // --- HTTP helpers ---

    private <T> T get(String path, Class<T> type) throws Exception {
        HttpRequest request = buildRequest("GET", path, null);
        return execute(request, type);
    }

    private <T> T get(String path, com.fasterxml.jackson.databind.JavaType type) throws Exception {
        HttpRequest request = buildRequest("GET", path, null);
        return execute(request, type);
    }

    private <T> T post(String path, Object body, Class<T> type) throws Exception {
        String json = mapper.writeValueAsString(body);
        HttpRequest request = buildRequest("POST", path, json);
        return execute(request, type);
    }

    private HttpRequest buildRequest(String method, String path, String jsonBody) {
        HttpRequest.Builder builder = HttpRequest.newBuilder()
                .uri(URI.create(baseURL + path))
                .timeout(Duration.ofSeconds(30));

        if (apiKey != null && !apiKey.isEmpty()) {
            builder.header("X-API-Key", apiKey);
        }

        if ("POST".equals(method) && jsonBody != null) {
            builder.header("Content-Type", "application/json");
            builder.POST(HttpRequest.BodyPublishers.ofString(jsonBody));
        } else {
            builder.GET();
        }

        return builder.build();
    }

    @SuppressWarnings("unchecked")
    private <T> T execute(HttpRequest request, Class<T> type) throws Exception {
        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() >= 400) {
            throw new GGIDException("API error (status " + response.statusCode() + "): " + response.body());
        }
        return mapper.readValue(response.body(), type);
    }

    private <T> T execute(HttpRequest request, com.fasterxml.jackson.databind.JavaType type) throws Exception {
        HttpResponse<String> response = httpClient.send(request, HttpResponse.BodyHandlers.ofString());
        if (response.statusCode() >= 400) {
            throw new GGIDException("API error (status " + response.statusCode() + "): " + response.body());
        }
        return mapper.readValue(response.body(), type);
    }

    // --- DTOs ---

    public static class Builder {
        private String baseURL;
        private String apiKey;

        public Builder baseURL(String baseURL) {
            this.baseURL = baseURL;
            return this;
        }

        public Builder apiKey(String apiKey) {
            this.apiKey = apiKey;
            return this;
        }

        public GGIDClient build() {
            if (baseURL == null || baseURL.isEmpty()) {
                throw new IllegalArgumentException("baseURL is required");
            }
            return new GGIDClient(this);
        }
    }

    public static class PermissionResult {
        @JsonProperty("allowed")
        public boolean allowed;
    }

    public static class User {
        @JsonProperty("id") public String id;
        @JsonProperty("tenant_id") public String tenantId;
        @JsonProperty("username") public String username;
        @JsonProperty("email") public String email;
        @JsonProperty("phone") public String phone;
        @JsonProperty("status") public String status;
        @JsonProperty("email_verified") public boolean emailVerified;
        @JsonProperty("created_at") public String createdAt;
        @JsonProperty("updated_at") public String updatedAt;
    }

    public static class CreateUserRequest {
        @JsonProperty("username") public String username;
        @JsonProperty("email") public String email;
        @JsonProperty("password") public String password;
        @JsonProperty("phone") public String phone;
    }

    public static class PageResult<T> {
        @JsonProperty("items") public List<T> items;
        @JsonProperty("total_count") public int totalCount;
        @JsonProperty("page") public int page;
        @JsonProperty("page_size") public int pageSize;
    }

    public static class GGIDException extends Exception {
        public GGIDException(String message) {
            super(message);
        }
    }
}
