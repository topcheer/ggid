package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Result of a policy check (ABAC or RBAC).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class PolicyResult {
    public boolean allowed;
    public String reason;

    public PolicyResult() {}

    public PolicyResult(boolean allowed, String reason) {
        this.allowed = allowed;
        this.reason = reason;
    }

    public boolean isAllowed() { return allowed; }
    public String getReason() { return reason; }
}
