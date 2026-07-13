package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

import java.util.List;

/**
 * Permission node in the permission tree.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class Permission {
    public String id;
    public String name;
    public String resource;
    public String action;
    public String description;
    public List<Permission> children;

    public String getId() { return id; }
    public String getName() { return name; }
    public String getResource() { return resource; }
    public String getAction() { return action; }
    public String getDescription() { return description; }
    public List<Permission> getChildren() { return children; }
}
