package com.example.erp.security;

import com.example.erp.service.AuthService;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

/**
 * Authentication interceptor — extracts the GGID JWT from a cookie
 * or Authorization header, verifies it, and attaches user info.
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
        if (isPublicPath(path)) {
            return true;
        }

        // Extract token from cookie first, then Authorization header
        String token = null;
        if (request.getCookies() != null) {
            for (Cookie c : request.getCookies()) {
                if ("ggid_token".equals(c.getName())) {
                    token = c.getValue();
                    break;
                }
            }
        }
        if (token == null) {
            String authHeader = request.getHeader("Authorization");
            if (authHeader != null && authHeader.startsWith("Bearer ")) {
                token = authHeader.substring(7);
            }
        }

        if (token == null) {
            response.sendRedirect("/login");
            return false;
        }

        var user = authService.verifyToken(token);
        if (user == null) {
            // Clear cookie and redirect to login
            Cookie clear = new Cookie("ggid_token", "");
            clear.setPath("/");
            clear.setMaxAge(0);
            response.addCookie(clear);
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
