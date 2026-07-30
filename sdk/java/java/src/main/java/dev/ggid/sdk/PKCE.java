package dev.ggid.sdk;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.SecureRandom;
import java.util.Base64;

/**
 * PKCE (Proof Key for Code Exchange) utilities — RFC 7636.
 *
 * Since the GGID OAuth server enforces PKCE S256, SDK consumers must generate
 * a code_verifier and code_challenge pair before redirecting to the authorize
 * endpoint, then send the code_verifier with the token exchange request.
 */
public class PKCE {

    private static final SecureRandom RANDOM = new SecureRandom();

    /**
     * Generate a cryptographically random code_verifier (RFC 7636 §4.1).
     * @return base64url-encoded string (no padding), 43-128 characters.
     */
    public static String generateCodeVerifier() {
        byte[] bytes = new byte[64];
        RANDOM.nextBytes(bytes);
        return base64UrlEncode(bytes);
    }

    /**
     * Derive the S256 code_challenge from a code_verifier (RFC 7636 §4.2).
     * code_challenge = BASE64URL(SHA256(code_verifier))
     */
    public static String generateCodeChallenge(String verifier) {
        try {
            MessageDigest sha256 = MessageDigest.getInstance("SHA-256");
            byte[] hash = sha256.digest(verifier.getBytes(StandardCharsets.US_ASCII));
            return base64UrlEncode(hash);
        } catch (Exception e) {
            throw new RuntimeException("SHA-256 not available", e);
        }
    }

    /**
     * Generate a complete PKCE pair for use with the authorize URL.
     * @return PKCEPair with code_verifier, code_challenge, and code_challenge_method.
     */
    public static PKCEPair generatePKCEPair() {
        String verifier = generateCodeVerifier();
        String challenge = generateCodeChallenge(verifier);
        return new PKCEPair(verifier, challenge, "S256");
    }

    private static String base64UrlEncode(byte[] bytes) {
        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }

    /**
     * Immutable holder for PKCE values.
     */
    public static class PKCEPair {
        private final String codeVerifier;
        private final String codeChallenge;
        private final String codeChallengeMethod;

        public PKCEPair(String codeVerifier, String codeChallenge, String codeChallengeMethod) {
            this.codeVerifier = codeVerifier;
            this.codeChallenge = codeChallenge;
            this.codeChallengeMethod = codeChallengeMethod;
        }

        public String getCodeVerifier() { return codeVerifier; }
        public String getCodeChallenge() { return codeChallenge; }
        public String getCodeChallengeMethod() { return codeChallengeMethod; }
    }
}