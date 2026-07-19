package com.example.erp.service;

import com.example.erp.config.GgidConfig;
import dev.ggid.sdk.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.io.IOException;

/**
 * Authentication service wrapping GGID SDK calls.
 *
 * Manages a cached service-account token for policy checks.
 * The token is refreshed when it expires or on first use.
 */
@Service
public class AuthService {

    private static final Logger log = LoggerFactory.getLogger(AuthService.class);

    private final GGIDClient ggidClient;
    private final JwtVerifier jwtVerifier;
    private final GgidConfig ggidConfig;

    private volatile String cachedServiceToken;
    private volatile long tokenExpiresAt;

    public AuthService(GGIDClient ggidClient, JwtVerifier jwtVerifier, GgidConfig ggidConfig) {
        this.ggidClient = ggidClient;
        this.jwtVerifier = jwtVerifier;
        this.ggidConfig = ggidConfig;
    }

    public TokenSet login(String username, String password) throws GGIDException, IOException {
        return ggidClient.login(username, password);
    }

    public GGIDUser verifyToken(String token) {
        return jwtVerifier.verifyUser(token);
    }

    /**
     * Check permission using a service-account (admin) token.
     * The gateway blocks non-admin scopes from /api/v1/policies/check,
     * so external apps must use a service account for policy evaluation.
     */
    public boolean checkPermissionWithServiceAccount(String userId, String resource, String action) {
        try {
            String svcToken = getServiceToken();
            if (svcToken == null) return false;

            PolicyResult result = ggidClient.checkPermission(svcToken, userId, resource, action);
            log.debug("Policy check: {}:{} for {} -> {} ({})", resource, action, userId,
                    result.isAllowed(), result.getReason());
            return result.isAllowed();
        } catch (Exception e) {
            log.warn("Policy check failed for {}:{} - {}", resource, action, e.getMessage());
            return false;
        }
    }

    /**
     * Get a cached service-account token, refreshing if needed.
     */
    private String getServiceToken() {
        long now = System.currentTimeMillis();
        if (cachedServiceToken != null && now < tokenExpiresAt) {
            return cachedServiceToken;
        }

        try {
            String pass = ggidConfig.getSvcPass();
            if (pass == null || pass.isEmpty()) {
                log.warn("No service-account password configured (ggid.service-account-pass)");
                return null;
            }

            TokenSet tokens = ggidClient.login(ggidConfig.getSvcUser(), pass);
            cachedServiceToken = tokens.getAccessToken();
            // Cache for 10 minutes (GGID tokens last 15)
            tokenExpiresAt = now + 10 * 60 * 1000;
            log.info("Service-account token refreshed");
            return cachedServiceToken;
        } catch (Exception e) {
            log.error("Failed to get service-account token: {}", e.getMessage());
            return null;
        }
    }
}
