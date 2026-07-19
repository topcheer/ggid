package dev.ggid.sdk;

import com.auth0.jwk.Jwk;
import com.auth0.jwk.JwkException;
import com.auth0.jwk.JwkProvider;
import com.auth0.jwk.UrlJwkProvider;
import com.auth0.jwt.JWT;
import com.auth0.jwt.algorithms.Algorithm;
import com.auth0.jwt.exceptions.JWTVerificationException;
import com.auth0.jwt.interfaces.DecodedJWT;
import com.auth0.jwt.interfaces.Verification;

import java.net.URL;
import java.security.interfaces.RSAPublicKey;

/**
 * Verifies RS256-signed JWT tokens against the GGID JWKS endpoint.
 * <p>
 * Usage:
 * <pre>{@code
 * JwtVerifier verifier = new JwtVerifier("https://ggid.example.com/.well-known/jwks.json");
 * GGIDUser user = verifier.verify(token);
 * if (user == null) { /* invalid token *&#47; }
 * }</pre>
 */
public class JwtVerifier {

    private final JwkProvider jwkProvider;
    private final String issuer;
    private final int leewaySeconds;

    /**
     * @param jwksUrl URL of the GGID JWKS endpoint (e.g. https://ggid.example.com/.well-known/jwks.json)
     */
    public JwtVerifier(String jwksUrl) {
        this(jwksUrl, null, 0);
    }

    /**
     * @param jwksUrl       URL of the GGID JWKS endpoint
     * @param issuer        Expected issuer claim (optional but recommended)
     * @param leewaySeconds Clock skew tolerance in seconds
     */
    public JwtVerifier(String jwksUrl, String issuer, int leewaySeconds) {
        try {
            this.jwkProvider = new UrlJwkProvider(new URL(jwksUrl));
        } catch (Exception e) {
            throw new IllegalArgumentException("Invalid JWKS URL: " + jwksUrl, e);
        }
        this.issuer = issuer;
        this.leewaySeconds = leewaySeconds;
    }

    /**
     * Verify and decode a JWT token.
     *
     * @param token Raw JWT string
     * @return DecodedJWT on success, null on failure
     */
    public DecodedJWT verify(String token) {
        try {
            // First decode to get the key ID
            DecodedJWT decoded = JWT.decode(token);
            String kid = decoded.getKeyId();
            if (kid == null) {
                return null;
            }

            // Fetch the matching public key from JWKS
            Jwk jwk = jwkProvider.get(kid);
            if (!(jwk.getPublicKey() instanceof RSAPublicKey)) {
                return null;
            }
            RSAPublicKey publicKey = (RSAPublicKey) jwk.getPublicKey();

            // Build verifier
            Verification verification = JWT.require(Algorithm.RSA256(publicKey, null));
            if (issuer != null && !issuer.isEmpty()) {
                verification.withIssuer(issuer);
            }
            if (leewaySeconds > 0) {
                verification.acceptLeeway(leewaySeconds);
            }

            return verification.build().verify(token);
        } catch (JWTVerificationException | JwkException e) {
            return null;
        } catch (Exception e) {
            return null;
        }
    }

    /**
     * Verify a JWT and return a populated GGIDUser.
     *
     * @param token Raw JWT string
     * @return GGIDUser on success, null on failure
     */
    public GGIDUser verifyUser(String token) {
        DecodedJWT jwt = verify(token);
        if (jwt == null) {
            return null;
        }
        GGIDUser user = new GGIDUser();
        user.userId = jwt.getSubject();
        user.tenantId = jwt.getClaim("tenant_id").asString();
        user.username = jwt.getClaim("username").asString();
        if (user.username == null) {
            user.username = jwt.getClaim("preferred_username").asString();
        }
        user.email = jwt.getClaim("email").asString();

        // Roles: check "roles" claim first, fall back to "scopes" (GGID format)
        String[] roles = jwt.getClaim("roles").asArray(String.class);
        if (roles == null) {
            // GGID puts role keys in "scopes" array
            roles = jwt.getClaim("scopes").asArray(String.class);
        }
        if (roles != null) {
            user.roles = roles;
        }

        // Scopes: support both "scope" (OAuth string) and "scopes" (GGID array)
        String[] scopesArr = jwt.getClaim("scopes").asArray(String.class);
        if (scopesArr != null) {
            user.scopes = scopesArr;
        } else {
            String scope = jwt.getClaim("scope").asString();
            if (scope != null && !scope.isEmpty()) {
                user.scopes = scope.split(" ");
            }
        }

        return user;
    }
}
