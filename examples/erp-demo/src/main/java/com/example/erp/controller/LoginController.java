package com.example.erp.controller;

import com.example.erp.model.ApiResponse;
import com.example.erp.service.AuthService;
import dev.ggid.sdk.GGIDUser;
import dev.ggid.sdk.TokenSet;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * Login controller — handles username/password authentication via GGID.
 * Stores the access token in a cookie for stateless authentication.
 */
@Controller
public class LoginController {

    private final AuthService authService;

    public LoginController(AuthService authService) {
        this.authService = authService;
    }

    @GetMapping("/login")
    public String loginForm(@RequestParam(required = false) String error, Model model) {
        if (error != null) {
            model.addAttribute("error", switch (error) {
                case "token_expired" -> "Your session has expired. Please log in again.";
                case "invalid" -> "Invalid username or password.";
                case "failed" -> "Authentication service unavailable. Please try again.";
                default -> "Login error.";
            });
        }
        model.addAttribute("pageTitle", "Login");
        return "login";
    }

    @PostMapping("/login.do")
    public String doLogin(@RequestParam String username,
                          @RequestParam String password,
                          HttpServletResponse response) {
        try {
            TokenSet tokens = authService.login(username, password);
            // Store token in a cookie (7 day expiry, matches token refresh cycle)
            Cookie cookie = new Cookie("ggid_token", tokens.getAccessToken());
            cookie.setPath("/");
            cookie.setMaxAge(7 * 24 * 60 * 60); // 7 days
            cookie.setHttpOnly(true);
            response.addCookie(cookie);
            return "redirect:/";
        } catch (Exception e) {
            return "redirect:/login?error=invalid";
        }
    }

    @PostMapping("/logout")
    public String logout(HttpServletRequest request, HttpServletResponse response) {
        Cookie cookie = new Cookie("ggid_token", "");
        cookie.setPath("/");
        cookie.setMaxAge(0);
        response.addCookie(cookie);
        return "redirect:/login";
    }

    @GetMapping("/healthz")
    @ResponseBody
    public ApiResponse<Map<String, String>> healthz() {
        return ApiResponse.ok(Map.of("status", "ok", "app", "erp-demo", "version", "1.0.0"));
    }
}
