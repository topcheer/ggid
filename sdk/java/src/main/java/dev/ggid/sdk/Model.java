package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

@JsonIgnoreProperties(ignoreUnknown = true)
public class TokenSet {
    @JsonProperty("access_token")
    private String accessToken;
    @JsonProperty("refresh_token")
    private String refreshToken;
    @JsonProperty("id_token")
    private String idToken;
    @JsonProperty("token_type")
    private String tokenType = "Bearer";
    @JsonProperty("expires_in")
    private int expiresIn;
    
    public String getAccessToken() { return accessToken; }
    public void setAccessToken(String v) { this.accessToken = v; }
    public String getRefreshToken() { return refreshToken; }
    public void setRefreshToken(String v) { this.refreshToken = v; }
    public int getExpiresIn() { return expiresIn; }
    public void setExpiresIn(int v) { this.expiresIn = v; }
}

@JsonIgnoreProperties(ignoreUnknown = true)
class User {
    private String id;
    private String username;
    private String email;
    private String status;
    @JsonProperty("display_name")
    private String displayName;
    
    public String getId() { return id; }
    public String getUsername() { return username; }
    public String getEmail() { return email; }
    public String getStatus() { return status; }
}

@JsonIgnoreProperties(ignoreUnknown = true)
class Role {
    private String id;
    private String name;
    private String key;
    private String description;
    @JsonProperty("system_role")
    private boolean systemRole;
    
    public String getId() { return id; }
    public String getName() { return name; }
    public String getKey() { return key; }
}

@JsonIgnoreProperties(ignoreUnknown = true)
class PolicyResult {
    private boolean allowed;
    private String reason;
    
    public boolean isAllowed() { return allowed; }
    public String getReason() { return reason; }
}

class GGIDException extends Exception {
    public GGIDException(String message) { super(message); }
    public GGIDException(String message, Throwable cause) { super(message, cause); }
}
