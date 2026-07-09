package dev.ggid.sdk;

import jakarta.servlet.Filter;
import jakarta.servlet.FilterChain;
import jakarta.servlet.FilterConfig;
import jakarta.servlet.ServletException;
import jakarta.servlet.ServletRequest;
import jakarta.servlet.ServletResponse;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

import java.io.IOException;
import java.util.Base64;
import java.util.HashSet;
import java.util.Set;

/**
 * Servlet Filter for GGID JWT authentication.
 *
 * Usage in web.xml:
 *   <filter>
 *     <filter-name>ggidAuth</filter-name>
 *     <filter-class>dev.ggid.sdk.GGIDAuthFilter</filter-class>
 *     <init-param>
 *       <param-name>publicPaths</param-name>
 *       <param-value>/health,/api/public</param-value>
 *     </init-param>
 *   </filter>
 *
 * Usage with Spring Boot:
 *   @Bean
 *   public FilterRegistrationBean<GGIDAuthFilter> ggidFilter() {
 *       FilterRegistrationBean<GGIDAuthFilter> bean = new FilterRegistrationBean<>();
 *       bean.setFilter(new GGIDAuthFilter());
 *       bean.addUrlPatterns("/api/*");
 *       bean.addInitParameter("publicPaths", "/api/auth/login,/api/auth/register");
 *       return bean;
 *   }
 *
 * After authentication, the user info is available via:
 *   GGIDUser user = (GGIDUser) request.getAttribute("ggidUser");
 */
public class GGIDAuthFilter implements Filter {

    private static final String AUTH_HEADER = "Authorization";
    private static final String BEARER_PREFIX = "Bearer ";
    private static final String USER_ATTR = "ggidUser";

    private final Set<String> publicPaths = new HashSet<>();

    @Override
    public void init(FilterConfig filterConfig) {
        String paths = filterConfig.getInitParameter("publicPaths");
        if (paths != null) {
            for (String path : paths.split(",")) {
                String trimmed = path.trim();
                if (!trimmed.isEmpty()) {
                    publicPaths.add(trimmed);
                }
            }
        }
    }

    @Override
    public void doFilter(ServletRequest request, ServletResponse response, FilterChain chain)
            throws IOException, ServletException {

        HttpServletRequest httpRequest = (HttpServletRequest) request;
        HttpServletResponse httpResponse = (HttpServletResponse) response;

        String path = httpRequest.getRequestURI();

        // Check public paths
        for (String publicPath : publicPaths) {
            if (path.equals(publicPath) || path.startsWith(publicPath + "/")) {
                chain.doFilter(request, response);
                return;
            }
        }

        // Extract Bearer token
        String authHeader = httpRequest.getHeader(AUTH_HEADER);
        if (authHeader == null || !authHeader.startsWith(BEARER_PREFIX)) {
            sendError(httpResponse, HttpServletResponse.SC_UNAUTHORIZED, "Missing authorization header");
            return;
        }

        String token = authHeader.substring(BEARER_PREFIX.length());

        // Parse JWT (offline — no signature verification for simplicity)
        // In production, verify against GGID JWKS endpoint
        GGIDUser user = parseJwt(token);
        if (user == null || user.userId == null || user.userId.isEmpty()) {
            sendError(httpResponse, HttpServletResponse.SC_UNAUTHORIZED, "Invalid or expired token");
            return;
        }

        // Inject user info into request
        httpRequest.setAttribute(USER_ATTR, user);

        chain.doFilter(request, response);
    }

    @Override
    public void destroy() {
        // noop
    }

    /**
     * Get the authenticated user from the request.
     * Returns null if not authenticated or if the filter was skipped.
     */
    public static GGIDUser getUser(HttpServletRequest request) {
        return (GGIDUser) request.getAttribute(USER_ATTR);
    }

    /**
     * Parse a JWT token and extract user info.
     * Does NOT verify signature — for production use, validate against GGID JWKS.
     */
    private GGIDUser parseJwt(String token) {
        try {
            String[] parts = token.split("\\.");
            if (parts.length != 3) return null;

            // Decode payload (part 1)
            String payload = new String(Base64.getUrlDecoder().decode(parts[1]));

            // Simple JSON parsing (avoid pulling in Jackson here)
            GGIDUser user = new GGIDUser();
            user.userId = extractJsonString(payload, "sub");
            user.tenantId = extractJsonString(payload, "tenant_id");
            user.username = extractJsonString(payload, "username");
            user.email = extractJsonString(payload, "email");

            // Extract roles array
            String rolesStr = extractJsonArray(payload, "roles");
            if (rolesStr != null && !rolesStr.isEmpty()) {
                user.roles = rolesStr.split(",");
            }

            // Extract scopes from space-delimited string
            String scopeStr = extractJsonString(payload, "scope");
            if (scopeStr != null && !scopeStr.isEmpty()) {
                user.scopes = scopeStr.split(" ");
            }

            return user;
        } catch (Exception e) {
            return null;
        }
    }

    /**
     * Extract a string value from JSON by key (simple parser, no nested objects).
     */
    private String extractJsonString(String json, String key) {
        String search = "\"" + key + "\"";
        int idx = json.indexOf(search);
        if (idx < 0) return null;

        // Find the value after the colon
        int colonIdx = json.indexOf(":", idx + search.length());
        if (colonIdx < 0) return null;

        // Find the opening quote
        int startQuote = json.indexOf("\"", colonIdx + 1);
        if (startQuote < 0) return null;

        // Find the closing quote
        int endQuote = json.indexOf("\"", startQuote + 1);
        if (endQuote < 0) return null;

        return json.substring(startQuote + 1, endQuote);
    }

    /**
     * Extract a simple string array from JSON (handles ["a","b"] format).
     */
    private String extractJsonArray(String json, String key) {
        String search = "\"" + key + "\"";
        int idx = json.indexOf(search);
        if (idx < 0) return null;

        int bracketStart = json.indexOf("[", idx);
        int bracketEnd = json.indexOf("]", bracketStart);
        if (bracketStart < 0 || bracketEnd < 0) return null;

        String arrayContent = json.substring(bracketStart + 1, bracketEnd);
        // Remove quotes and split by comma
        String cleaned = arrayContent.replaceAll("\"", "").replaceAll("\\s", "");
        if (cleaned.isEmpty()) return null;
        return cleaned;
    }

    private void sendError(HttpServletResponse response, int status, String message) throws IOException {
        response.setStatus(status);
        response.setContentType("application/json");
        response.getWriter().write("{\"error\":\"" + message + "\"}");
    }
}
