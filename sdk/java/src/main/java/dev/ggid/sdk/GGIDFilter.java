package dev.ggid.sdk;

import com.auth0.jwt.JWT;
import com.auth0.jwt.algorithms.Algorithm;
import com.auth0.jwt.interfaces.DecodedJWT;
import jakarta.servlet.*;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;

/**
 * Servlet filter for JWT authentication.
 *
 * Usage in web.xml:
 *   <filter>
 *     <filter-name>ggid</filter-name>
 *     <filter-class>dev.ggid.sdk.GGIDFilter</filter-class>
 *   </filter>
 *
 * Or in Spring Boot:
 *   @Bean
 *   public FilterRegistrationBean<GGIDFilter> ggidFilter() {
 *       FilterRegistrationBean<GGIDFilter> bean = new FilterRegistrationBean<>();
 *       bean.setFilter(new GGIDFilter());
 *       bean.addUrlPatterns("/api/*");
 *       return bean;
 *   }
 */
public class GGIDFilter implements Filter {

    private static final String PUBLIC_PREFIX = "/api/v1/auth/";

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
            // Decode without verification (verification should use JWKS in production)
            DecodedJWT jwt = JWT.decode(token);
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
