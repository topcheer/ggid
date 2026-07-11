package dev.ggid.sdk;

import com.auth0.jwt.interfaces.DecodedJWT;
import jakarta.servlet.*;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;

/**
 * Servlet filter for JWT authentication with RS256 signature verification.
 *
 * Requires a JwtVerifier configured with the GGID JWKS endpoint URL.
 *
 * Usage in web.xml:
 *   <filter>
 *     <filter-name>ggid</filter-name>
 *     <filter-class>dev.ggid.sdk.GGIDFilter</filter-class>
 *     <init-param>
 *       <param-name>jwksUrl</param-name>
 *       <param-value>https://ggid.example.com/.well-known/jwks.json</param-value>
 *     </init-param>
 *   </filter>
 *
 * Or in Spring Boot:
 *   @Bean
 *   public FilterRegistrationBean<GGIDFilter> ggidFilter() {
 *       FilterRegistrationBean<GGIDFilter> bean = new FilterRegistrationBean<>();
 *       bean.setFilter(new GGIDFilter("https://ggid.example.com/.well-known/jwks.json"));
 *       bean.addUrlPatterns("/api/*");
 *       return bean;
 *   }
 */
public class GGIDFilter implements Filter {

    private static final String PUBLIC_PREFIX = "/api/v1/auth/";

    private JwtVerifier verifier;

    /** No-arg constructor for servlet container instantiation. */
    public GGIDFilter() {}

    /** Constructor with JWKS URL — use with Spring Boot FilterRegistrationBean. */
    public GGIDFilter(String jwksUrl) {
        this.verifier = new JwtVerifier(jwksUrl);
    }

    @Override
    public void init(FilterConfig filterConfig) {
        String jwksUrl = filterConfig.getInitParameter("jwksUrl");
        if (jwksUrl != null && !jwksUrl.isEmpty()) {
            verifier = new JwtVerifier(jwksUrl);
        }
    }

    @Override
    public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain)
            throws IOException, ServletException {

        HttpServletRequest req = (HttpServletRequest) request;
        HttpServletResponse resp = (HttpServletResponse) response;

        String path = req.getRequestURI();

        // Skip public paths
        if (path.startsWith(PUBLIC_PREFIX) || path.equals("/healthz") || path.equals("/login")) {
            chain.doFilter(request, response);
            return;
        }

        // Extract Bearer token
        String authHeader = req.getHeader("Authorization");
        if (authHeader == null || !authHeader.startsWith("Bearer ")) {
            sendError(resp, 401, "missing bearer token");
            return;
        }

        String token = authHeader.substring(7);

        try {
            if (verifier == null) {
                sendError(resp, 500, "JWT verifier not configured");
                return;
            }

            // Verify RS256 signature via JWKS
            DecodedJWT jwt = verifier.verify(token);
            if (jwt == null) {
                sendError(resp, 401, "invalid or expired token");
                return;
            }

            req.setAttribute("ggid.sub", jwt.getSubject());
            req.setAttribute("ggid.email", jwt.getClaim("email").asString());
            req.setAttribute("ggid.tenant_id", jwt.getClaim("tenant_id").asString());
            chain.doFilter(request, response);
        } catch (Exception e) {
            sendError(resp, 401, "invalid token");
        }
    }

    private void sendError(HttpServletResponse resp, int status, String message) throws IOException {
        resp.setStatus(status);
        resp.setContentType("application/json");
        resp.getWriter().write("{\"error\":\"" + message + "\"}");
    }
}
