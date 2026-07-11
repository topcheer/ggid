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
import java.util.HashSet;
import java.util.Set;

/**
 * Servlet Filter for GGID JWT authentication with RS256 signature verification.
 *
 * Requires a JwtVerifier configured with the GGID JWKS endpoint URL.
 * Set the "jwksUrl" init parameter in web.xml or pass it to the constructor.
 *
 * Usage in web.xml:
 *   <filter>
 *     <filter-name>ggidAuth</filter-name>
 *     <filter-class>dev.ggid.sdk.GGIDAuthFilter</filter-class>
 *     <init-param>
 *       <param-name>jwksUrl</param-name>
 *       <param-value>https://ggid.example.com/.well-known/jwks.json</param-value>
 *     </init-param>
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
 *       bean.setFilter(new GGIDAuthFilter("https://ggid.example.com/.well-known/jwks.json"));
 *       bean.addUrlPatterns("/api/*");
 *       bean.addInitParameter("publicPaths", "/api/auth/login,/api/auth/register");
 *       return bean;
 *   }
 *
 * After authentication, the verified user is available via:
 *   GGIDUser user = (GGIDUser) request.getAttribute("ggidUser");
 */
public class GGIDAuthFilter implements Filter {

    private static final String AUTH_HEADER = "Authorization";
    private static final String BEARER_PREFIX = "Bearer ";
    private static final String USER_ATTR = "ggidUser";

    private final Set<String> publicPaths = new HashSet<>();
    private JwtVerifier verifier;

    /** No-arg constructor for servlet container instantiation. Set jwksUrl via init-param. */
    public GGIDAuthFilter() {}

    /** Constructor with JWKS URL — use with Spring Boot FilterRegistrationBean. */
    public GGIDAuthFilter(String jwksUrl) {
        this.verifier = new JwtVerifier(jwksUrl);
    }

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

        // Initialize verifier from init-param if not set via constructor
        if (verifier == null) {
            String jwksUrl = filterConfig.getInitParameter("jwksUrl");
            if (jwksUrl != null && !jwksUrl.isEmpty()) {
                verifier = new JwtVerifier(jwksUrl);
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

        // Verify JWT signature via JWKS (RS256)
        GGIDUser user;
        if (verifier != null) {
            user = verifier.verifyUser(token);
        } else {
            // Fallback: no JWKS configured — reject in production
            sendError(httpResponse, HttpServletResponse.SC_INTERNAL_SERVER_ERROR,
                    "JWT verifier not configured");
            return;
        }

        if (user == null || user.userId == null || user.userId.isEmpty()) {
            sendError(httpResponse, HttpServletResponse.SC_UNAUTHORIZED,
                    "Invalid or expired token");
            return;
        }

        // Inject verified user info into request
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

    private void sendError(HttpServletResponse response, int status, String message) throws IOException {
        response.setStatus(status);
        response.setContentType("application/json");
        response.getWriter().write("{\"error\":\"" + message + "\"}");
    }
}
