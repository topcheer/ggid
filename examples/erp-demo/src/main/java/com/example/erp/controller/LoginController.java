package com.example.erp.controller;

import com.example.erp.model.ApiResponse;
import com.example.erp.service.AuthService;
import dev.ggid.sdk.GGIDUser;
import dev.ggid.sdk.TokenSet;
import jakarta.servlet.http.HttpSession;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * Login controller — handles username/password authentication via GGID.
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
                          HttpSession session) {
        try {
            TokenSet tokens = authService.login(username, password);
            session.setAttribute("access_token", tokens.getAccessToken());
            session.setAttribute("refresh_token", tokens.getRefreshToken());

            // Verify and store user info
            GGIDUser user = authService.verifyToken(tokens.getAccessToken());
            if (user != null) {
                session.setAttribute("user", user);
            }

            return "redirect:/";
        } catch (Exception e) {
            return "redirect:/login?error=invalid";
        }
    }

    @PostMapping("/logout")
    public String logout(HttpSession session) {
        session.invalidate();
        return "redirect:/login";
    }

    /**
     * Health check endpoint.
     */
    @GetMapping("/healthz")
    @ResponseBody
    public ApiResponse<Map<String, String>> healthz() {
        return ApiResponse.ok(Map.of("status", "ok", "app", "erp-demo", "version", "1.0.0"));
    }
}
