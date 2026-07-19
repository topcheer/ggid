package com.example.erp.security;

import com.example.erp.service.AuthService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

/**
 * Authentication interceptor — extracts Bearer token from Authorization header,
 * verifies it via GGID JWT, and stores the GGIDUser in request attributes.
 *
 * Public paths (login, callback, static resources) are excluded.
 */
@Component
public class AuthInterceptor implements HandlerInterceptor {

    private final AuthService authService;

    public AuthInterceptor(AuthService authService) {
        this.authService = authService;
    }

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler)
            throws Exception {

        String path = request.getRequestURI();

        // Skip public paths
        if (isPublicPath(path)) {
            return true;
        }

        // Check session for token
        String token = (String) request.getSession().getAttribute("access_token");
        if (token == null) {
            // Try Authorization header (for API calls)
            String authHeader = request.getHeader("Authorization");
            if (authHeader != null && authHeader.startsWith("Bearer ")) {
                token = authHeader.substring(7);
            }
        }

        if (token == null) {
            response.sendRedirect("/login");
            return false;
        }

        // Verify token and attach user info
        var user = authService.verifyToken(token);
        if (user == null) {
            request.getSession().invalidate();
            response.sendRedirect("/login?error=token_expired");
            return false;
        }

        request.setAttribute("currentUser", user);
        request.setAttribute("accessToken", token);
        return true;
    }

    private boolean isPublicPath(String path) {
        return path.equals("/") || path.equals("/login") || path.equals("/login.do")
                || path.equals("/callback") || path.startsWith("/static/")
                || path.startsWith("/css/") || path.startsWith("/js/")
                || path.equals("/favicon.ico") || path.equals("/healthz");
    }
}
