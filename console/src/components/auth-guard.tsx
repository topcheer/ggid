"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Sidebar } from "@/components/sidebar";

const PUBLIC_PATHS = ["/login", "/register", "/forgot-password", "/reset-password", "/setup"];

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();

  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    const token = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") : null;
    const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p));

    // If on /setup page, clear any stale tokens to ensure clean wizard
    if (pathname === "/setup") {
      localStorage.removeItem("ggid_access_token");
      localStorage.removeItem("ggid_refresh_token");
      localStorage.removeItem("ggid_user_scopes");
      localStorage.removeItem("ggid_user_id");
      localStorage.removeItem("ggid_tenant_id");
      setChecked(true);
      return;
    }

    // If no token and not on a public path → check system status
    if (!token && !isPublic) {
      fetch("/api/v1/system/status")
        .then((resp) => resp.json())
        .then((data) => {
          if (data.initialized === false) {
            router.replace("/setup");
          } else {
            router.push(`/login?redirect_to=${encodeURIComponent(pathname)}`);
          }
          setChecked(true);
        })
        .catch(() => {
          router.push(`/login?redirect_to=${encodeURIComponent(window.location.pathname)}`);
          setChecked(true);
        });
      return;
    }

    // If on /setup but already logged in → go to dashboard
    if (pathname === "/setup" && token) {
      router.replace("/dashboard");
      setChecked(true);
      return;
    }

    if (token) {
      // Validate token before trusting it — prevents stale/expired tokens
      // from rendering the dashboard and flooding it with 401 errors.
      (async () => {
        try {
          const verifyResp = await fetch("/api/v1/users/me", {
            headers: { Authorization: `Bearer ${token}` },
          });
          if (verifyResp.status === 401) {
            // Token is invalid/expired — clear and redirect to login
            localStorage.removeItem("ggid_access_token");
            localStorage.removeItem("ggid_refresh_token");
            localStorage.removeItem("ggid_session_id");
            localStorage.removeItem("ggid_user_scopes");
            localStorage.removeItem("ggid_user_id");
            setIsAuthenticated(false);
            setChecked(true);
            if (!isPublic) {
              router.push(`/login?redirect_to=${encodeURIComponent(pathname)}`);
            }
            return;
          }
        } catch {
          // Network error — fail open, let the 401 handler deal with it later
        }

        setIsAuthenticated(true);

        // Route-level permission check: block direct URL access to admin pages
        const ADMIN_PREFIXES: Record<string, string> = {
          "/users": "tenant", "/roles": "tenant", "/audit": "tenant",
          "/organizations": "tenant", "/sessions": "user",
          "/settings": "tenant", "/api-keys": "tenant", "/oauth-clients": "tenant",
          "/webhooks": "tenant", "/policies": "tenant", "/security/": "tenant",
          "/access-requests": "user", "/analytics/": "tenant", "/monitoring/": "tenant",
          "/admin/tenants": "platform",
          "/admin/audit": "platform",
          "/admin/threats": "platform",
          "/admin/impersonate": "tenant",
          "/admin/secrets": "tenant",
          "/admin/key-rotation": "tenant",
          "/admin/backup": "tenant",
          "/admin/settings": "tenant",
          "/admin/feature-flags": "tenant",
          "/admin/health": "tenant",
          "/devices": "tenant",
          "/access-reviews": "tenant",
          "/activity": "user",
          "/exports": "tenant",
          "/providers/config": "tenant",
        };
        const userScopes = JSON.parse(localStorage.getItem("ggid_user_scopes") || '["user:self"]');
        const isPlatform = userScopes.some((s: string) => {
          const ls = s.toLowerCase();
          return ls === "platform:admin" || ls === "platform administrator" || ls === "platform_admin";
        });
        const isTenant = userScopes.some((s: string) => {
          const ls = s.toLowerCase();
          return ls === "tenant:admin" || ls === "manager" || ls === "tenant administrator" || ls === "tenant_admin";
        });
        for (const [prefix, scope] of Object.entries(ADMIN_PREFIXES)) {
          if (pathname.startsWith(prefix)) {
            if (scope === "tenant" && !isTenant) {
              router.replace("/dashboard");
              setChecked(true);
              return;
            }
            if (scope === "platform" && !isPlatform) {
              router.replace("/dashboard");
              setChecked(true);
              return;
            }
            break;
          }
        }
        setChecked(true);
      })();
    } else {
      setIsAuthenticated(false);
      setChecked(true);
    }
  }, [pathname, router]);

  // Listen for 401 events from api.ts to force logout without page reload
  useEffect(() => {
    const handleUnauthorized = () => {
      // If already on a public page (login/register/setup), don't redirect — prevents loop
      const currentPath = typeof window !== "undefined" ? window.location.pathname : "";
      const isPublicPath = PUBLIC_PATHS.some((p) => currentPath === p || currentPath.startsWith(p));
      if (isPublicPath) return;

      // Clear all auth state
      ["ggid_access_token","ggid_refresh_token","ggid_session_id","ggid_tenant_id","ggid_user_id","ggid_user_name","ggid_user_email","ggid_user_scopes","ggid_user_permissions"].forEach(k => localStorage.removeItem(k));
      setIsAuthenticated(false);
      // Use full page reload to login to ensure clean state
      window.location.replace(`/login?redirect_to=${encodeURIComponent(currentPath)}`);
    };
    window.addEventListener("ggid:unauthorized", handleUnauthorized);
    return () => window.removeEventListener("ggid:unauthorized", handleUnauthorized);
  }, []);

  if (!checked) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p));

  // Public pages (login, register, etc.) render full-screen without sidebar
  if (isPublic) {
    return <main className="min-h-screen dark:bg-gray-950">{children}</main>;
  }

  // Not authenticated and not on a public path — show spinner while
  // redirecting to login. Do NOT render children (avoids 401 flood).
  if (!isAuthenticated) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  // Authenticated pages render with sidebar layout
  return (
    <div className="flex h-screen dark:bg-gray-950">
      <Sidebar />
      <main id="main-content" className="flex-1 overflow-auto">
        <div className="p-4 md:p-6">{children}</div>
      </main>
    </div>
  );
}
