package dev.ggid.sdk;

/** JWT token response from GGID login. */
public class TokenSet {
    private final String accessToken;
    private final String refreshToken;
    private final String tokenType;
    private final int expiresIn;

    public TokenSet(String accessToken, String refreshToken, String tokenType, int expiresIn) {
        this.accessToken = accessToken;
        this.refreshToken = refreshToken;
        this.tokenType = tokenType;
        this.expiresIn = expiresIn;
    }

    public String getAccessToken() { return accessToken; }
    public String getRefreshToken() { return refreshToken; }
    public String getTokenType() { return tokenType; }
    public int getExpiresIn() { return expiresIn; }
}
