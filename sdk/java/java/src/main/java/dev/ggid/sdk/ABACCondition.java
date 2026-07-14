package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * A single attribute-based condition for ABAC evaluation.
 * Operators: eq, ne, in, regex, startsWith, endsWith, gt, lt
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class ABACCondition {
    public String field;
    public String operator;
    public String value;

    public ABACCondition() {}

    public ABACCondition(String field, String operator, String value) {
        this.field = field;
        this.operator = operator;
        this.value = value;
    }

    public String getField() { return field; }
    public String getOperator() { return operator; }
    public String getValue() { return value; }
}
