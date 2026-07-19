package com.example.erp.security;

import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.GGIDException;
import dev.ggid.sdk.GGIDUser;
import dev.ggid.sdk.PolicyResult;
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
 * Before the controller method executes, this aspect calls
 * GGID's POST /api/v1/policies/check to verify the user has
 * the required resource+action permission.
 *
 * If denied, returns 403 Forbidden with a descriptive reason.
 */
@Aspect
@Component
public class PermissionAspect {

    private final GGIDClient ggidClient;

    public PermissionAspect(GGIDClient ggidClient) {
        this.ggidClient = ggidClient;
    }

    @Around("@annotation(requirePermission)")
    public Object checkPermission(ProceedingJoinPoint joinPoint, RequirePermission requirePermission)
            throws Throwable {

        HttpServletRequest request = ((ServletRequestAttributes)
                RequestContextHolder.currentRequestAttributes()).getRequest();

        GGIDUser user = (GGIDUser) request.getAttribute("currentUser");
        String token = (String) request.getAttribute("accessToken");

        if (user == null || token == null) {
            throw new ResponseStatusException(HttpStatus.UNAUTHORIZED, "authentication required");
        }

        String resource = requirePermission.resource();
        String action = requirePermission.action();

        try {
            PolicyResult result = ggidClient.checkPermission(token, user.userId, resource, action);
            if (!result.isAllowed()) {
                throw new ResponseStatusException(HttpStatus.FORBIDDEN,
                        String.format("Permission denied: %s:%s — %s", resource, action, result.getReason()));
            }
        } catch (GGIDException e) {
            // If policy service is unavailable, fall back to role-based check
            if (!hasRoleFallback(user, resource, action)) {
                throw new ResponseStatusException(HttpStatus.FORBIDDEN,
                        String.format("Permission denied: %s:%s", resource, action));
            }
        }

        return joinPoint.proceed();
    }

    /**
     * Fallback: if policy API is unreachable, use the role-permission matrix
     * defined in ErpPermissions. This makes the demo resilient.
     */
    private boolean hasRoleFallback(GGIDUser user, String resource, String action) {
        if (user.roles == null) return false;
        for (String role : user.roles) {
            if (ErpPermissions.isAllowed(role, resource, action)) return true;
        }
        return false;
    }
}
