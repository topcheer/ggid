package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Authenticated user information extracted from JWT.
 */
public class GGIDUser {
    @JsonProperty("user_id")
    public String userId;

    @JsonProperty("tenant_id")
    public String tenantId;

    @JsonProperty("username")
    public String username;

    @JsonProperty("email")
    public String email;

    @JsonProperty("roles")
    public String[] roles;

    @JsonProperty("scopes")
    public String[] scopes;

    public boolean hasRole(String role) {
        if (roles == null) return false;
        for (String r : roles) {
            if (r.equals(role) || r.equals("admin")) return true;
        }
        return false;
    }

    public boolean hasScope(String scope) {
        if (scopes == null) return false;
        for (String s : scopes) {
            if (s.equals(scope)) return true;
        }
        return false;
    }
}
