package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * RBAC role.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class Role {
    public String id;
    public String key;
    public String name;
    public String description;

    @JsonProperty("system_role")
    public boolean systemRole;

    public String getId() { return id; }
    public String getKey() { return key; }
    public String getName() { return name; }
    public String getDescription() { return description; }
}
