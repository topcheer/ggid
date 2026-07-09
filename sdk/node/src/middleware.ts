/**
 * GGID SDK — Express/Hono middleware for Node.js.
 *
 * Provides JWT-based authentication and permission checking
 * for Node.js backend applications.
 *
 * Usage (Express):
 *
 *   import express from "express";
 *   import { authMiddleware, requireRole } from "@ggid/sdk-node/middleware";
 *
 *   const app = express();
 *
 *   // Protect all routes except public ones
 *   app.use(authMiddleware({
 *     baseURL: "http://localhost:8080",
 *     publicPaths: ["/health", "/api/v1/auth/login"],
 *   }));
 *
 *   // Route with role check
 *   app.get("/admin", requireRole("admin"), (req, res) => {
 *     res.json({ user: req.ggidUser });
 *   });
 */

import { jwtVerify, createRemoteJWKSet } from "jose";

export interface AuthMiddlewareOptions {
  /** GGID Gateway base URL */
  baseURL: string;
  /** Paths that skip JWT verification */
  publicPaths?: string[];
  /** Default tenant ID */
  tenantID?: string;
}

export interface GGIDUser {
  userId: string;
  tenantId: string;
  username: string;
  email: string;
  roles: string[];
  scopes: string[];
  claims: Record<string, unknown>;
}

/**
 * Express middleware that verifies GGID JWT tokens.
 * Populates req.ggidUser on success.
 */
export function authMiddleware(options: AuthMiddlewareOptions) {
  let jwks: ReturnType<typeof createRemoteJWKSet> | null = null;
  const publicPaths = options.publicPaths || [];

  return async (req: any, res: any, next: any) => {
    // Check public paths
    const path = req.path || req.url || "";
    for (const p of publicPaths) {
      if (path === p || path.startsWith(p + "/") || path.startsWith(p)) {
        return next();
      }
    }

    // Extract Bearer token
    const authHeader = req.headers?.authorization || req.headers?.get?.("authorization");
    if (!authHeader || !authHeader.startsWith("Bearer ")) {
      return res.status(401).json({ error: "Missing authorization header" });
    }

    const token = authHeader.slice(7);

    try {
      // Lazy-init JWKS client
      if (!jwks) {
        jwks = createRemoteJWKSet(new URL(options.baseURL + "/oauth/jwks"));
      }

      // Verify JWT signature
      const { payload } = await jwtVerify(token, jwks, {
        issuer: "ggid-auth",
        audience: "ggid",
      });

      // Populate user info
      req.ggidUser = {
        userId: String(payload.sub || ""),
        tenantId: String(payload.tenant_id || ""),
        username: String(payload.username || ""),
        email: String(payload.email || ""),
        roles: Array.isArray(payload.roles) ? payload.roles.map(String) : [],
        scopes: typeof payload.scope === "string" ? payload.scope.split(" ") : [],
        claims: payload as Record<string, unknown>,
      };

      // Inject tenant header if configured
      if (options.tenantID) {
        req.headers["x-tenant-id"] = options.tenantID;
      }

      next();
    } catch {
      return res.status(401).json({ error: "Invalid or expired token" });
    }
  };
}

/**
 * Middleware that requires a specific role.
 * Must be used after authMiddleware.
 *
 * Usage: app.get("/admin", requireRole("admin"), handler)
 */
export function requireRole(role: string) {
  return (req: any, res: any, next: any) => {
    const user = req.ggidUser as GGIDUser | undefined;
    if (!user) {
      return res.status(401).json({ error: "Not authenticated" });
    }
    if (!user.roles.includes(role) && !user.roles.includes("admin")) {
      return res.status(403).json({ error: `Insufficient role: requires '${role}'` });
    }
    next();
  };
}

/**
 * Middleware that checks a permission via the Policy Service.
 *
 * Usage: app.get("/users", requirePermission("iam:users", "read", "http://localhost:8080"), handler)
 */
export function requirePermission(
  resource: string,
  action: string,
  gatewayURL: string,
) {
  return async (req: any, res: any, next: any) => {
    const user = req.ggidUser as GGIDUser | undefined;
    if (!user) {
      return res.status(401).json({ error: "Not authenticated" });
    }

    try {
      const authHeader = req.headers?.authorization || req.headers?.get?.("authorization");
      const resp = await fetch(`${gatewayURL}/api/v1/policies/check`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-ID": user.tenantId,
          ...(authHeader ? { Authorization: authHeader } : {}),
        },
        body: JSON.stringify({
          user_id: user.userId,
          resource,
          action,
        }),
      });

      if (!resp.ok) {
        return res.status(500).json({ error: "Permission check failed" });
      }

      const data = await resp.json();
      if (!data.allowed) {
        return res.status(403).json({ error: `Permission denied: ${resource}/${action}` });
      }

      next();
    } catch {
      return res.status(500).json({ error: "Permission check error" });
    }
  };
}

/**
 * Hono-compatible middleware (works with Hono, Elysia, etc.)
 * Returns a function that takes a context and next callback.
 */
export function honoAuth(options: AuthMiddlewareOptions) {
  const mw = authMiddleware(options);
  return async (c: any, next: () => Promise<void>) => {
    return new Promise<void>((resolve) => {
      mw(c.req.raw || c, {
        status: (code: number) => ({
          json: (body: unknown) => {
            c.status?.(code);
            return c.json?.(body) ?? c.json(body);
          },
        }),
      }, () => { next(); resolve(); });
    });
  };
}
