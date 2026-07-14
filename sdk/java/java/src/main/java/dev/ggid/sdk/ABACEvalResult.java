package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

/**
 * Result of ABAC policy evaluation.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class ABACEvalResult {
    public boolean matched;
    @JsonProperty("matched_rules")
    public List<String> matchedRules;

    public boolean isMatched() { return matched; }
    public List<String> getMatchedRules() { return matchedRules; }
}
