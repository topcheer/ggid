package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * GGID user account.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class User {
    public String id;
    public String username;
    public String email;
    public String status;

    @JsonProperty("display_name")
    public String displayName;

    public String getId() { return id; }
    public String getUsername() { return username; }
    public String getEmail() { return email; }
    public String getStatus() { return status; }
    public String getDisplayName() { return displayName; }
}
