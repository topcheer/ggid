# SDK Authorization Integration Guide

## Overview

GGID IAM provides a two-layer authorization model for application integration:

### Layer 1: JWT Claim-based Authorization (Simple Roles)

The JWT access token contains a `roles` claim (e.g., `["admin", "developer"]`) and a `scopes` claim (e.g., `["users:read", "users:write"]`). This layer is evaluated locally by the SDK middleware without any network call.

**Use when**:
- Simple role checks (e.g., "is this user an admin?")
- Scope-based access control (e.g., "does the token have `users:write` scope?")
- High-frequency checks where latency matters (no API call needed)
- Frontend menu visibility toggles

### Layer 2: Policy Engine Authorization (Fine-grained RBAC/ABAC)

The GGID Policy Service evaluates access decisions against configured RBAC roles, ABAC attribute conditions, and SoD rules. This layer requires an API call to the policy service.

**Use when**:
- Resource-level permission checks (e.g., "can user X delete resource Y?")
- Attribute-based access control (e.g., "can a warehouse manager transfer stock from warehouse A?")
- Segregation of duties enforcement
- Dynamic policy changes without token re-issuance
- Audit trail of authorization decisions

### Decision Matrix

| Requirement | Layer | Method |
|-------------|-------|--------|
| Is user admin? | JWT Claim | Check `roles` claim |
| Does token have `users:write`? | JWT Claim | Check `scopes` claim |
| Can user delete product X? | Policy Engine | `CheckPermission(resource="product", action="delete")` |
| Can warehouse manager transfer from WH-A? | Policy Engine | `EvaluateABAC(attributes + conditions)` |
| Full ABAC evaluation with context | Policy Engine | `CheckPolicy(subject, resource, action, context)` |

## Go SDK

**Import**: `github.com/ggid/ggid/sdk/go/ggid`

### Setup

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ggid/ggid/sdk/go/ggid"
)

func main() {
    client, err := ggid.NewClient(&ggid.Config{
        BaseURL:   "https://ggid.example.com",
        JWKSURL:   "https://ggid.example.com/.well-known/jwks.json",
        ClientID:  "your-client-id",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    token := "eyJhbGciOiJSUzI1NiIs..." // JWT access token

    // --- Scenario 1: CheckPermission ---
    // Check if user can delete a product
    result, err := client.CheckPermission(ctx, token, "product", "delete")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Can delete product: %v (reason: %s)\n", result.Allowed, result.Reason)

    // --- Scenario 2: EvaluateABAC ---
    // Check if warehouse manager can transfer stock from specific warehouse
    abacResult, err := client.EvaluateABAC(ctx, token, ggid.ABACEvalRequest{
        Attributes: map[string]string{
            "user_role":       "warehouse_manager",
            "warehouse_id":    "WH-001",
            "resource_type":   "inventory",
            "action":          "transfer",
        },
        Conditions: []ggid.ABACCondition{
            {Field: "user_role", Operator: "eq", Value: "warehouse_manager"},
            {Field: "warehouse_id", Operator: "eq", Value: "WH-001"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("ABAC matched: %v, rules: %v\n", abacResult.Matched, abacResult.MatchedRules)

    // --- Scenario 3: AssignRole ---
    // Assign admin role to a user
    err = client.AssignRole(ctx, token, "user-uuid-123", "role-uuid-admin")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Role assigned successfully")

    // --- Scenario 4: GetUserRoles ---
    // Query all roles for a user
    roles, err := client.GetUserRoles(ctx, token, "user-uuid-123")
    if err != nil {
        log.Fatal(err)
    }
    for _, role := range roles {
        fmt.Printf("Role: %s (key: %s, system: %v)\n", role.Name, role.Key, role.SystemRole)
    }

    // --- Scenario 5: CheckPolicy (full ABAC) ---
    // Full ABAC evaluation with subject, resource, action, and context
    policyResult, err := client.CheckPolicy(ctx, token, &ggid.PolicyCheckRequest{
        Subject:  "user-uuid-123",
        Resource: "inventory:WH-001",
        Action:   "transfer",
        Context: map[string]string{
            "department":    "logistics",
            "warehouse_id":  "WH-001",
            "time_of_day":   "business_hours",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Policy decision: allowed=%v, reason=%s\n", policyResult.Allowed, policyResult.Reason)
}
```

### HTTP Middleware (Gin)

```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/ggid/ggid/sdk/go/ggid"
)

func AuthMiddleware(client *ggid.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }
        token := strings.TrimPrefix(auth, "Bearer ")

        // Verify JWT
        claims, err := client.VerifyToken(c.Request.Context(), token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }

        c.Set("user_id", claims["sub"])
        c.Set("tenant_id", claims["tenant_id"])
        c.Set("roles", claims["roles"])
        c.Set("token", token)
        c.Next()
    }
}

func RequirePermission(client *ggid.Client, resource, action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetString("token")
        result, err := client.CheckPermission(c.Request.Context(), token, resource, action)
        if err != nil || !result.Allowed {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
            return
        }
        c.Next()
    }
}

// Usage in routes:
// router.POST("/api/products/:id", AuthMiddleware(client), RequirePermission(client, "product", "delete"), deleteProduct)
```

### HTTP Middleware (Echo)

```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/ggid/ggid/sdk/go/ggid"
)

func GGIDAuth(client *ggid.Client) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            auth := c.Request().Header.Get("Authorization")
            if !strings.HasPrefix(auth, "Bearer ") {
                return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
            }
            token := strings.TrimPrefix(auth, "Bearer ")
            claims, err := client.VerifyToken(c.Request().Context(), token)
            if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
            }
            c.Set("user_id", claims["sub"])
            c.Set("token", token)
            return next(c)
        }
    }
}

func RequirePerm(client *ggid.Client, resource, action string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            token := c.Get("token").(string)
            result, err := client.CheckPermission(c.Request().Context(), token, resource, action)
            if err != nil || !result.Allowed {
                return echo.NewHTTPError(http.StatusForbidden, "permission denied")
            }
            return next(c)
        }
    }
}
```

## Node.js SDK

**Import**: `@ggid/sdk-node`

### Setup and Scenarios

```typescript
import { GGIDClient } from '@ggid/sdk-node';

const client = new GGIDClient({
  baseURL: 'https://ggid.example.com',
  jwksURL: 'https://ggid.example.com/.well-known/jwks.json',
  clientID: 'your-client-id',
});

const token = 'eyJhbGciOiJSUzI1NiIs...';

async function main() {
  // --- Scenario 1: CheckPermission ---
  const canDelete = await client.checkPermission(token, 'product', 'delete');
  console.log(`Can delete product: ${canDelete.allowed} (${canDelete.reason})`);

  // --- Scenario 2: EvaluateABAC ---
  const abacResult = await client.evaluateABAC(token, {
    attributes: {
      user_role: 'warehouse_manager',
      warehouse_id: 'WH-001',
      resource_type: 'inventory',
      action: 'transfer',
    },
    conditions: [
      { field: 'user_role', operator: 'eq', value: 'warehouse_manager' },
      { field: 'warehouse_id', operator: 'eq', value: 'WH-001' },
    ],
  });
  console.log(`ABAC matched: ${abacResult.matched}, rules: ${abacResult.matchedRules}`);

  // --- Scenario 3: AssignRole ---
  await client.assignRole(token, 'user-uuid-123', 'role-uuid-admin');
  console.log('Role assigned');

  // --- Scenario 4: GetUserRoles ---
  const roles = await client.getUserRoles(token, 'user-uuid-123');
  roles.forEach(r => console.log(`Role: ${r.name} (key: ${r.key})`));

  // --- Scenario 5: CheckPolicy ---
  const policyResult = await client.checkPolicy(token, {
    subject: 'user-uuid-123',
    resource: 'inventory:WH-001',
    action: 'transfer',
    context: {
      department: 'logistics',
      warehouse_id: 'WH-001',
      time_of_day: 'business_hours',
    },
  });
  console.log(`Policy: allowed=${policyResult.allowed}, reason=${policyResult.reason}`);
}

main().catch(console.error);
```

### Express Middleware

```typescript
import { Request, Response, NextFunction } from 'express';
import { GGIDClient } from '@ggid/sdk-node';

export function authMiddleware(client: GGIDClient) {
  return async (req: Request, res: Response, next: NextFunction) => {
    const auth = req.headers.authorization;
    if (!auth?.startsWith('Bearer ')) {
      return res.status(401).json({ error: 'missing token' });
    }
    const token = auth.slice(7);
    try {
      const claims = await client.verifyToken(token);
      (req as any).userId = claims.sub;
      (req as any).tenantId = claims.tenant_id;
      (req as any).token = token;
      next();
    } catch {
      return res.status(401).json({ error: 'invalid token' });
    }
  };
}

export function requirePermission(client: GGIDClient, resource: string, action: string) {
  return async (req: Request, res: Response, next: NextFunction) => {
    const token = (req as any).token;
    try {
      const result = await client.checkPermission(token, resource, action);
      if (!result.allowed) {
        return res.status(403).json({ error: 'permission denied' });
      }
      next();
    } catch {
      return res.status(500).json({ error: 'authorization check failed' });
    }
  };
}

// Usage:
// app.post('/api/products/:id', authMiddleware(client), requirePermission(client, 'product', 'delete'), deleteProduct);
```

## Java SDK

**Import**: `dev.ggid.sdk.GGIDClient`

### Setup and Scenarios

```java
import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.model.*;
import java.util.Map;
import java.util.List;

public class AuthorizationExample {
    public static void main(String[] args) {
        GGIDClient client = GGIDClient.builder()
            .baseURL("https://ggid.example.com")
            .jwksURL("https://ggid.example.com/.well-known/jwks.json")
            .clientID("your-client-id")
            .build();

        String token = "eyJhbGciOiJSUzI1NiIs...";

        // --- Scenario 1: CheckPermission ---
        PolicyResult canDelete = client.checkPermission(token, "product", "delete");
        System.out.printf("Can delete product: %b (reason: %s)%n", canDelete.isAllowed(), canDelete.getReason());

        // --- Scenario 2: EvaluateABAC ---
        ABACEvalRequest abacReq = ABACEvalRequest.builder()
            .attribute("user_role", "warehouse_manager")
            .attribute("warehouse_id", "WH-001")
            .attribute("resource_type", "inventory")
            .attribute("action", "transfer")
            .condition(ABACCondition.builder().field("user_role").operator("eq").value("warehouse_manager").build())
            .condition(ABACCondition.builder().field("warehouse_id").operator("eq").value("WH-001").build())
            .build();
        ABACEvalResult abacResult = client.evaluateABAC(token, abacReq);
        System.out.printf("ABAC matched: %b, rules: %s%n", abacResult.isMatched(), abacResult.getMatchedRules());

        // --- Scenario 3: AssignRole ---
        client.assignRole(token, "user-uuid-123", "role-uuid-admin");
        System.out.println("Role assigned");

        // --- Scenario 4: GetUserRoles ---
        List<Role> roles = client.getUserRoles(token, "user-uuid-123");
        for (Role role : roles) {
            System.out.printf("Role: %s (key: %s)%n", role.getName(), role.getKey());
        }

        // --- Scenario 5: CheckPolicy ---
        PolicyCheckRequest policyReq = PolicyCheckRequest.builder()
            .subject("user-uuid-123")
            .resource("inventory:WH-001")
            .action("transfer")
            .context("department", "logistics")
            .context("warehouse_id", "WH-001")
            .context("time_of_day", "business_hours")
            .build();
        PolicyResult policyResult = client.checkPolicy(token, policyReq);
        System.out.printf("Policy: allowed=%b, reason=%s%n", policyResult.isAllowed(), policyResult.getReason());
    }
}
```

### Spring Boot Interceptor

```java
import org.springframework.web.servlet.HandlerInterceptor;
import dev.ggid.sdk.GGIDClient;
import dev.ggid.sdk.model.PolicyResult;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;

public class GGIDAuthInterceptor implements HandlerInterceptor {
    private final GGIDClient client;

    public GGIDAuthInterceptor(GGIDClient client) {
        this.client = client;
    }

    @Override
    public boolean preHandle(HttpServletRequest req, HttpServletResponse resp, Object handler) throws Exception {
        String auth = req.getHeader("Authorization");
        if (auth == null || !auth.startsWith("Bearer ")) {
            resp.setStatus(401);
            resp.getWriter().write("{\"error\":\"missing token\"}");
            return false;
        }
        String token = auth.substring(7);
        try {
            Map<String, Object> claims = client.verifyToken(token);
            req.setAttribute("userId", claims.get("sub"));
            req.setAttribute("token", token);
            return true;
        } catch (Exception e) {
            resp.setStatus(401);
            resp.getWriter().write("{\"error\":\"invalid token\"}");
            return false;
        }
    }
}

public class PermissionInterceptor implements HandlerInterceptor {
    private final GGIDClient client;
    private final String resource;
    private final String action;

    public PermissionInterceptor(GGIDClient client, String resource, String action) {
        this.client = client;
        this.resource = resource;
        this.action = action;
    }

    @Override
    public boolean preHandle(HttpServletRequest req, HttpServletResponse resp, Object handler) throws Exception {
        String token = (String) req.getAttribute("token");
        PolicyResult result = client.checkPermission(token, resource, action);
        if (!result.isAllowed()) {
            resp.setStatus(403);
            resp.getWriter().write("{\"error\":\"permission denied\"}");
            return false;
        }
        return true;
    }
}

// Registration in WebMvcConfigurer:
// registry.addInterceptor(new GGIDAuthInterceptor(client)).addPathPatterns("/api/**");
// registry.addInterceptor(new PermissionInterceptor(client, "product", "delete")).addPathPatterns("/api/products/*/delete");
```

## Python SDK

**Import**: `from ggid import GGIDClient`

### Setup and Scenarios

```python
from ggid import GGIDClient

client = GGIDClient(
    base_url="https://ggid.example.com",
    jwks_url="https://ggid.example.com/.well-known/jwks.json",
    client_id="your-client-id",
)

token = "eyJhbGciOiJSUzI1NiIs..."

# --- Scenario 1: CheckPermission ---
can_delete = client.check_permission(token, resource="product", action="delete")
print(f"Can delete product: {can_delete.allowed} ({can_delete.reason})")

# --- Scenario 2: EvaluateABAC ---
abac_result = client.evaluate_abac(token, attributes={
    "user_role": "warehouse_manager",
    "warehouse_id": "WH-001",
    "resource_type": "inventory",
    "action": "transfer",
}, conditions=[
    {"field": "user_role", "operator": "eq", "value": "warehouse_manager"},
    {"field": "warehouse_id", "operator": "eq", "value": "WH-001"},
])
print(f"ABAC matched: {abac_result.matched}, rules: {abac_result.matched_rules}")

# --- Scenario 3: AssignRole ---
client.assign_role(token, user_id="user-uuid-123", role_id="role-uuid-admin")
print("Role assigned")

# --- Scenario 4: GetUserRoles ---
roles = client.get_user_roles(token, user_id="user-uuid-123")
for role in roles:
    print(f"Role: {role.name} (key: {role.key})")

# --- Scenario 5: CheckPolicy ---
policy_result = client.check_policy(token, subject="user-uuid-123",
    resource="inventory:WH-001", action="transfer",
    context={
        "department": "logistics",
        "warehouse_id": "WH-001",
        "time_of_day": "business_hours",
    })
print(f"Policy: allowed={policy_result.allowed}, reason={policy_result.reason}")
```

### Flask Decorator

```python
from functools import wraps
from flask import request, jsonify, g
from ggid import GGIDClient

client = GGIDClient(
    base_url="https://ggid.example.com",
    jwks_url="https://ggid.example.com/.well-known/jwks.json",
    client_id="your-client-id",
)

def auth_required(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        auth = request.headers.get("Authorization", "")
        if not auth.startswith("Bearer "):
            return jsonify({"error": "missing token"}), 401
        token = auth[7:]
        try:
            claims = client.verify_token(token)
            g.user_id = claims.get("sub")
            g.tenant_id = claims.get("tenant_id")
            g.token = token
        except Exception:
            return jsonify({"error": "invalid token"}), 401
        return f(*args, **kwargs)
    return decorated

def permission_required(resource: str, action: str):
    def decorator(f):
        @wraps(f)
        def decorated(*args, **kwargs):
            token = g.token
            try:
                result = client.check_permission(token, resource=resource, action=action)
                if not result.allowed:
                    return jsonify({"error": "permission denied"}), 403
            except Exception:
                return jsonify({"error": "authorization check failed"}), 500
            return f(*args, **kwargs)
        return decorated
    return decorator

# Usage:
# @app.route("/api/products/<id>", methods=["DELETE"])
# @auth_required
# @permission_required("product", "delete")
# def delete_product(id):
#     ...
```

### FastAPI Dependency

```python
from fastapi import Depends, HTTPException, Request
from ggid import GGIDClient

client = GGIDClient(
    base_url="https://ggid.example.com",
    jwks_url="https://ggid.example.com/.well-known/jwks.json",
    client_id="your-client-id",
)

async def get_current_user(request: Request):
    auth = request.headers.get("Authorization", "")
    if not auth.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="missing token")
    token = auth[7:]
    try:
        claims = client.verify_token(token)
        return {"user_id": claims.get("sub"), "tenant_id": claims.get("tenant_id"), "token": token}
    except Exception:
        raise HTTPException(status_code=401, detail="invalid token")

def require_permission(resource: str, action: str):
    async def dependency(user: dict = Depends(get_current_user)):
        result = client.check_permission(user["token"], resource=resource, action=action)
        if not result.allowed:
            raise HTTPException(status_code=403, detail="permission denied")
        return user
    return dependency

# Usage:
# @app.delete("/api/products/{id}")
# async def delete_product(id: str, user: dict = Depends(require_permission("product", "delete"))):
#     ...
```

## ERP Application Integration

This section demonstrates how to integrate GGID authorization into a cross-border e-commerce ERP system.

### Architecture Overview

```
Frontend (React/Next.js)
  |
  |--- JWT token in localStorage/cookie
  |--- Menu visibility from JWT roles claim
  |--- API calls with Authorization: Bearer <token>
  |
  v
Backend API (Go/Node/Python)
  |
  |--- JWT verification (Layer 1: local, no network)
  |--- Permission check via SDK (Layer 2: policy engine)
  |--- ABAC evaluation for complex rules
  |
  v
GGID Policy Service
  |
  |--- RBAC: role -> permission mappings
  |--- ABAC: attribute conditions
  |--- SoD: conflict detection
```

### 1. Frontend Menu Control

The ERP frontend uses JWT claims to control menu visibility without API calls:

```typescript
// frontend/lib/auth.ts
import { jwtDecode } from 'jwt-decode';

interface GGIDClaims {
  sub: string;
  tenant_id: string;
  roles: string[];
  scopes: string[];
}

export function getMenuItems(token: string): MenuItem[] {
  const claims = jwtDecode<GGIDClaims>(token);
  const allMenus: MenuItem[] = [
    { path: '/dashboard', label: 'Dashboard', roles: ['admin', 'manager', 'staff'] },
    { path: '/products', label: 'Products', roles: ['admin', 'manager', 'staff'] },
    { path: '/orders', label: 'Orders', roles: ['admin', 'manager', 'staff'] },
    { path: '/inventory', label: 'Inventory', roles: ['admin', 'warehouse_manager'] },
    { path: '/finance', label: 'Finance', roles: ['admin', 'finance'] },
    { path: '/logistics', label: 'Logistics', roles: ['admin', 'logistics'] },
    { path: '/settings', label: 'Settings', roles: ['admin'] },
  ];
  return allMenus.filter(m => m.roles.some(r => claims.roles.includes(r)));
}
```

### 2. Backend API Permission Enforcement

The ERP backend enforces permissions on every API endpoint:

```go
// Go example: ERP product service
router := gin.New()
router.Use(middleware.AuthMiddleware(ggidClient))

// Simple RBAC: product management
router.GET("/api/products", middleware.RequirePermission(ggidClient, "product", "read"), listProducts)
router.POST("/api/products", middleware.RequirePermission(ggidClient, "product", "create"), createProduct)
router.DELETE("/api/products/:id", middleware.RequirePermission(ggidClient, "product", "delete"), deleteProduct)

// ABAC: warehouse-scoped inventory transfer
router.POST("/api/inventory/transfer", func(c *gin.Context) {
    token := c.GetString("token")
    warehouseID := c.Query("warehouse_id")

    // ABAC check: can this user transfer from this specific warehouse?
    result, err := ggidClient.CheckPolicy(c.Request.Context(), token, &ggid.PolicyCheckRequest{
        Subject:  c.GetString("user_id"),
        Resource: fmt.Sprintf("inventory:%s", warehouseID),
        Action:   "transfer",
        Context: map[string]string{
            "warehouse_id": warehouseID,
            "department":   c.GetString("department"),
        },
    })
    if err != nil || !result.Allowed {
        c.JSON(403, gin.H{"error": "not authorized for this warehouse"})
        return
    }
    // Proceed with transfer...
})
```

### 3. ABAC Policy Configuration

Configure ABAC policies in GGID for warehouse-scoped access:

```json
// POST /api/v1/policies/abac/groups
{
  "name": "warehouse-transfer-policy",
  "description": "Warehouse managers can transfer inventory only from their assigned warehouse",
  "conditions": [
    {
      "logic": "AND",
      "rules": [
        { "field": "user_role", "operator": "eq", "value": "warehouse_manager" },
        { "field": "warehouse_id", "operator": "eq", "value": "{{context.warehouse_id}}" },
        { "field": "action", "operator": "eq", "value": "transfer" }
      ]
    }
  ],
  "effect": "allow",
  "priority": 100
}
```

### 4. Role Assignment Workflow

```go
// Onboard new warehouse manager
func onboardWarehouseManager(client *ggid.Client, adminToken, userID, warehouseID string) error {
    ctx := context.Background()

    // Assign warehouse_manager role
    err := client.AssignRole(ctx, adminToken, userID, "role-warehouse-manager")
    if err != nil {
        return fmt.Errorf("assign role: %w", err)
    }

    // Verify role assignment
    roles, err := client.GetUserRoles(ctx, adminToken, userID)
    if err != nil {
        return fmt.Errorf("verify roles: %w", err)
    }

    for _, role := range roles {
        if role.Key == "warehouse_manager" {
            log.Printf("User %s assigned warehouse_manager role for warehouse %s", userID, warehouseID)
            return nil
        }
    }

    return fmt.Errorf("role assignment verification failed")
}
```

### 5. Complete Authorization Flow

```
1. User logs in via GGID OAuth2 -> receives JWT with roles + scopes
2. Frontend reads JWT roles -> shows/hides menu items (Layer 1)
3. User clicks "Delete Product" -> frontend calls DELETE /api/products/:id
4. Backend middleware verifies JWT (Layer 1)
5. Backend calls CheckPermission("product", "delete") via SDK (Layer 2)
6. Policy service evaluates: does user's role include product:delete?
7. If allowed -> proceed; if denied -> return 403
8. For warehouse transfer: backend calls CheckPolicy with ABAC context
9. Policy service evaluates attribute conditions (warehouse_id match)
10. All decisions logged to audit service for compliance
```

## API Reference Summary

| Method | Go SDK | Node SDK | Java SDK | Python SDK | Endpoint |
|--------|--------|----------|----------|------------|----------|
| CheckPermission | `CheckPermission(ctx, token, resource, action)` | `checkPermission(token, resource, action)` | `checkPermission(token, resource, action)` | `check_permission(token, resource, action)` | POST /api/v1/policies/check |
| EvaluateABAC | `EvaluateABAC(ctx, token, req)` | `evaluateABAC(token, req)` | `evaluateABAC(token, req)` | `evaluate_abac(token, ...)` | POST /api/v1/policies/abac/evaluate |
| CheckPolicy | `CheckPolicy(ctx, token, req)` | `checkPolicy(token, req)` | `checkPolicy(token, req)` | `check_policy(token, ...)` | POST /api/v1/policies/check |
| AssignRole | `AssignRole(ctx, token, userID, roleID)` | `assignRole(token, userID, roleID)` | `assignRole(token, userID, roleID)` | `assign_role(token, user_id, role_id)` | POST /api/v1/roles/assign |
| GetUserRoles | `GetUserRoles(ctx, token, userID)` | `getUserRoles(token, userID)` | `getUserRoles(token, userID)` | `get_user_roles(token, user_id)` | GET /api/v1/users/{id}/roles |
| VerifyToken | `VerifyToken(ctx, token)` | `verifyToken(token)` | `verifyToken(token)` | `verify_token(token)` | Local (JWKS) |

## See Also

- [Go SDK Documentation](./go-sdk-documentation.md)
- [RBAC Implementation Guide](./rbac-implementation.md)
- [ABAC Policy Guide](./abac-policy-guide.md)
- [OAuth 2.1 Implementation](./oauth-2-1-implementation.md)
- [API Rate Limiting Strategy](./api-rate-limiting-strategy.md)
- [Identity Provisioning Standards](./identity-provisioning-standards.md)
