package dev.ggid.sdk;

import java.util.List;

/**
 * AI Agent Identity registration response.
 */
public class Agent {
    private String id;
    private String tenantId;
    private String name;
    private String type;
    private String ownerUserId;
    private String clientId;
    private String status;
    private List<String> allowedScopes;
    private int maxDelegationDepth;
    private String createdAt;

    public String getId() { return id; }
    public void setId(String id) { this.id = id; }
    public String getTenantId() { return tenantId; }
    public void setTenantId(String tenantId) { this.tenantId = tenantId; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getOwnerUserId() { return ownerUserId; }
    public void setOwnerUserId(String ownerUserId) { this.ownerUserId = ownerUserId; }
    public String getClientId() { return clientId; }
    public void setClientId(String clientId) { this.clientId = clientId; }
    public String getStatus() { return status; }
    public void setStatus(String status) { this.status = status; }
    public List<String> getAllowedScopes() { return allowedScopes; }
    public void setAllowedScopes(List<String> allowedScopes) { this.allowedScopes = allowedScopes; }
    public int getMaxDelegationDepth() { return maxDelegationDepth; }
    public void setMaxDelegationDepth(int maxDelegationDepth) { this.maxDelegationDepth = maxDelegationDepth; }
    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
}
