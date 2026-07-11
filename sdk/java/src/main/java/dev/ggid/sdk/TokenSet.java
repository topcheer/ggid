package dev.ggid.sdk;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * JWT token response from GGID login.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class TokenSet {
    @JsonProperty("access_token")
    private String accessToken;

    @JsonProperty("refresh_token")
    private String refreshToken;

    @JsonProperty("id_token")
    private String idToken;

    @JsonProperty("token_type")
    private String tokenType = "Bearer";

    @JsonProperty("expires_in")
    private int expiresIn;

    public TokenSet() {}

    public TokenSet(String accessToken, String refreshToken, String tokenType, int expiresIn) {
        this.accessToken = accessToken;
        this.refreshToken = refreshToken;
        this.tokenType = tokenType;
        this.expiresIn = expiresIn;
    }

    public String getAccessToken() { return accessToken; }
    public void setAccessToken(String v) { this.accessToken = v; }

    public String getRefreshToken() { return refreshToken; }
    public void setRefreshToken(String v) { this.refreshToken = v; }

    public String getIdToken() { return idToken; }
    public void setIdToken(String v) { this.idToken = v; }

    public String getTokenType() { return tokenType; }
    public void setTokenType(String v) { this.tokenType = v; }

    public int getExpiresIn() { return expiresIn; }
    public void setExpiresIn(int v) { this.expiresIn = v; }
}
