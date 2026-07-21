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
     * Creates a JwtVerifier from a GGID base URL.
     * The JWKS URL is derived as baseURL + "/.well-known/jwks.json".
     * This enables OIDC-style auto-discovery — just pass the base URL.
     */
    public JwtVerifier(String baseURL) {
        this(baseURL, null, 0);
    }

    /**
     * Creates a JwtVerifier from a GGID base URL with issuer validation.
     */
    public JwtVerifier(String baseURL, String issuer, int leewaySeconds) {
        String jwksUrl = baseURL.endsWith("/")
                ? baseURL + ".well-known/jwks.json"
                : baseURL + "/.well-known/jwks.json";
        try {
            this.jwkProvider = new UrlJwkProvider(new URL(jwksUrl));
        } catch (Exception e) {
            throw new IllegalArgumentException("Invalid GGID base URL: " + baseURL, e);
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

        // Roles: from "roles" claim only
        String[] roles = jwt.getClaim("roles").asArray(String.class);
        if (roles != null) {
            user.roles = roles;
        }

        // Permissions: from "permissions" claim only
        String[] permissions = jwt.getClaim("permissions").asArray(String.class);
        if (permissions != null) {
            user.permissions = permissions;
        }

        // Scopes: from "scope" claim (OAuth2 standard, space-separated string)
        String scope = jwt.getClaim("scope").asString();
        if (scope != null && !scope.isEmpty()) {
            user.scopes = scope.split(" ");
        }

        return user;
    }
}
