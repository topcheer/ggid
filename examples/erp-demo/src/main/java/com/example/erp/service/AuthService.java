package com.example.erp.service;

import dev.ggid.sdk.*;
import org.springframework.stereotype.Service;

import java.io.IOException;

/**
 * Authentication service wrapping GGID SDK calls.
 */
@Service
public class AuthService {

    private final GGIDClient ggidClient;
    private final JwtVerifier jwtVerifier;

    public AuthService(GGIDClient ggidClient, JwtVerifier jwtVerifier) {
        this.ggidClient = ggidClient;
        this.jwtVerifier = jwtVerifier;
    }

    /**
     * Login with username/password via GGID.
     */
    public TokenSet login(String username, String password) throws GGIDException, IOException {
        return ggidClient.login(username, password);
    }

    /**
     * Verify a JWT token and return user info.
     */
    public GGIDUser verifyToken(String token) {
        return jwtVerifier.verifyUser(token);
    }

    /**
     * Check if user has permission for resource+action.
     */
    public boolean checkPermission(String token, String userId, String resource, String action) {
        try {
            PolicyResult result = ggidClient.checkPermission(token, userId, resource, action);
            return result.isAllowed();
        } catch (Exception e) {
            return false;
        }
    }
}
