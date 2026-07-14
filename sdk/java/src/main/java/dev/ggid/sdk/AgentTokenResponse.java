package dev.ggid.sdk;

import java.util.List;

/**
 * AI Agent token exchange response.
 */
public class AgentTokenResponse {
    private String accessToken;
    private String tokenType;
    private int expiresIn;
    private String scope;
    private String agentId;
    private int delegationDepthRemaining;

    public String getAccessToken() { return accessToken; }
    public void setAccessToken(String accessToken) { this.accessToken = accessToken; }
    public String getTokenType() { return tokenType; }
    public void setTokenType(String tokenType) { this.tokenType = tokenType; }
    public int getExpiresIn() { return expiresIn; }
    public void setExpiresIn(int expiresIn) { this.expiresIn = expiresIn; }
    public String getScope() { return scope; }
    public void setScope(String scope) { this.scope = scope; }
    public String getAgentId() { return agentId; }
    public void setAgentId(String agentId) { this.agentId = agentId; }
    public int getDelegationDepthRemaining() { return delegationDepthRemaining; }
    public void setDelegationDepthRemaining(int delegationDepthRemaining) { this.delegationDepthRemaining = delegationDepthRemaining; }
}
