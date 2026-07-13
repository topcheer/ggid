package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

import java.util.List;
import java.util.Map;

/**
 * Request body for ABAC policy evaluation.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class ABACEvalRequest {
    public Map<String, String> attributes;
    public List<ABACCondition> conditions;

    public ABACEvalRequest() {}

    public ABACEvalRequest(Map<String, String> attributes, List<ABACCondition> conditions) {
        this.attributes = attributes;
        this.conditions = conditions;
    }

    public Map<String, String> getAttributes() { return attributes; }
    public List<ABACCondition> getConditions() { return conditions; }
}
