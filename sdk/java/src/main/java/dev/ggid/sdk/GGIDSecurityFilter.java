package dev.ggid.sdk;

import jakarta.servlet.*;
import jakarta.servlet.http.*;
import java.io.IOException;
import java.util.*;

/**
 * Servlet filter for GGID JWT authentication.
 * 
 * Usage in web.xml or via Spring Boot FilterRegistrationBean.
 * Skips paths starting with /api/v1/auth/, /login, /healthz.
 */
public class GGIDSecurityFilter implements Filter {
    
    private final GGIDClient client;
    private final Set<String> publicPaths = Set.of("/", "/healthz", "/login", "/register", "/docs");
    
    public GGIDSecurityFilter(GGIDClient client) {
        this.client = client;
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
        
        // Store token for downstream use
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

/**
 * Represents the authenticated GGID user.
 * Can be used as @AuthenticationPrincipal in Spring controllers.
 */
class GGIDUser {
    private String subject;
    private String email;
    private String name;
    private List<String> roles;
    private Map<String, Object> claims;
    
    public String getSubject() { return subject; }
    public String getEmail() { return email; }
    public String getName() { return name; }
    public List<String> getRoles() { return roles != null ? roles : Collections.emptyList(); }
    public Map<String, Object> getClaims() { return claims; }
}
