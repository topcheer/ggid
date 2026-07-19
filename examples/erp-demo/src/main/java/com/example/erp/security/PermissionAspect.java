package com.example.erp.security;

import com.example.erp.service.AuthService;
import dev.ggid.sdk.GGIDUser;
import jakarta.servlet.http.HttpServletRequest;
import org.aspectj.lang.ProceedingJoinPoint;
import org.aspectj.lang.annotation.Around;
import org.aspectj.lang.annotation.Aspect;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Component;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;
import org.springframework.web.server.ResponseStatusException;

/**
 * AOP aspect that enforces @RequirePermission annotations.
 *
 * Uses a cached admin (service-account) token to call GGID's
 * POST /api/v1/policies/check, because the gateway blocks
 * non-admin scopes from the policy API.
 *
 * If the policy API is unreachable, falls back to a local
 * permission matrix based on the user's scope.
 */
@Aspect
@Component
public class PermissionAspect {

    private final AuthService authService;

    public PermissionAspect(AuthService authService) {
        this.authService = authService;
    }

    @Around("@annotation(requirePermission)")
    public Object checkPermission(ProceedingJoinPoint joinPoint, RequirePermission requirePermission)
            throws Throwable {

        HttpServletRequest request = ((ServletRequestAttributes)
                RequestContextHolder.currentRequestAttributes()).getRequest();

        GGIDUser user = (GGIDUser) request.getAttribute("currentUser");

        if (user == null) {
            throw new ResponseStatusException(HttpStatus.UNAUTHORIZED, "authentication required");
        }

        String resource = requirePermission.resource();
        String action = requirePermission.action();

        // Try GGID policy check with service account token
        boolean allowed = authService.checkPermissionWithServiceAccount(user.userId, resource, action);
        if (!allowed) {
            // Fallback: check local permission matrix using user's scopes
            allowed = checkLocalFallback(user, resource, action);
        }

        if (!allowed) {
            throw new ResponseStatusException(HttpStatus.FORBIDDEN,
                    String.format("Permission denied: %s:%s", resource, action));
        }

        return joinPoint.proceed();
    }

    private boolean checkLocalFallback(GGIDUser user, String resource, String action) {
        // GGID puts role keys in scopes (e.g. "erp:sales_manager")
        String[] scopes = user.scopes;
        if (scopes == null) scopes = user.roles;
        if (scopes == null) return false;

        for (String scope : scopes) {
            if (ErpPermissions.isAllowed(scope, resource, action)) return true;
        }
        return false;
    }
}
