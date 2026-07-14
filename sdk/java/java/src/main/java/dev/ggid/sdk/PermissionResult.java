package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Result of a permission check.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class PermissionResult {
    public boolean allowed;
    public String reason;

    public boolean isAllowed() { return allowed; }
    public String getReason() { return reason; }
}
