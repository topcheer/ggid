package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Organization entity.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class Organization {
    public String id;
    public String name;

    @JsonProperty("parent_id")
    public String parentId;

    public String getId() { return id; }
    public String getName() { return name; }
    public String getParentId() { return parentId; }
}
