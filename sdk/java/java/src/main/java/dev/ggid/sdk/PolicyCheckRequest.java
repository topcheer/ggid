package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.Map;

/**
 * Request body for ABAC policy evaluation.
 * Unlike the basic CheckPermission, this supports additional context attributes.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class PolicyCheckRequest {
    public String subject;
    public String resource;
    public String action;
    public Map<String, String> context;
    @JsonProperty("tenant_id")
    public String tenantId;

    public PolicyCheckRequest() {}

    public PolicyCheckRequest(String subject, String resource, String action) {
        this.subject = subject;
        this.resource = resource;
        this.action = action;
    }

    public PolicyCheckRequest(String subject, String resource, String action,
                               Map<String, String> context, String tenantId) {
        this.subject = subject;
        this.resource = resource;
        this.action = action;
        this.context = context;
        this.tenantId = tenantId;
    }

    public String getSubject() { return subject; }
    public String getResource() { return resource; }
    public String getAction() { return action; }
    public Map<String, String> getContext() { return context; }
    public String getTenantId() { return tenantId; }
}
