package dev.ggid.sdk;

import jakarta.servlet.*;
import jakarta.servlet.http.*;
import java.io.IOException;
import java.util.*;

/**
 * Servlet filter for GGID JWT authentication with RS256 signature verification.
 *
 * Requires a JwtVerifier configured with the GGID JWKS endpoint URL.
 *
 * Usage in web.xml or via Spring Boot FilterRegistrationBean.
 * Skips paths starting with /api/v1/auth/, /login, /healthz.
 */
public class GGIDSecurityFilter implements Filter {

    private final JwtVerifier verifier;
    private final Set<String> publicPaths = Set.of("/", "/healthz", "/login", "/register", "/docs");

    public GGIDSecurityFilter(GGIDClient client, String jwksUrl) {
        this.verifier = new JwtVerifier(jwksUrl);
    }

    public GGIDSecurityFilter(String jwksUrl) {
        this.verifier = new JwtVerifier(jwksUrl);
    }

    @Override
    public void doFilter(ServletRequest req, ServletResponse res, FilterChain chain)
            throws IOException, ServletException {

        HttpServletRequest request = (HttpServletRequest) req;
        HttpServletResponse response = (HttpServletResponse) res;
        String path = request.getRequestURI();

        // Skip public paths
        if (publicPaths.contains(path) || path.startsWith("/api/v1/auth/") || path.startsWith("/oauth/")) {
            chain.doFilter(req, res);
            return;
        }

        // Extract token
        String authHeader = request.getHeader("Authorization");
        if (authHeader == null || !authHeader.startsWith("Bearer ")) {
            response.setStatus(401);
            response.setContentType("application/json");
            response.getWriter().write("{\"error\":\"missing bearer token\"}");
            return;
        }

        String token = authHeader.substring(7);

        // Verify JWT signature via JWKS
        GGIDUser user = verifier.verifyUser(token);
        if (user == null || user.userId == null || user.userId.isEmpty()) {
            response.setStatus(401);
            response.setContentType("application/json");
            response.getWriter().write("{\"error\":\"invalid or expired token\"}");
            return;
        }

        // Store verified user and token for downstream use
        request.setAttribute("ggid.user", user);
        request.setAttribute("ggid.token", token);

        chain.doFilter(req, res);
    }
}

/**
 * Convenience annotation for permission-based access control.
 * Used with Spring AOP to check permissions before method execution.
 */
@interface RequiresPermission {
    String resource();
    String action();
}
