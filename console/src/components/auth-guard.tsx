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
            router.push("/login");
          }
          setChecked(true);
        })
        .catch(() => {
          // Fetch failed — still go to login as fallback
          router.push("/login");
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
      setIsAuthenticated(true);

      // Route-level permission check: block direct URL access to admin pages
      const ADMIN_PREFIXES: Record<string, string> = {
        "/users": "manager", "/roles": "manager", "/audit": "manager",
        "/organizations": "manager", "/sessions": "manager",
        "/settings": "manager", "/api-keys": "admin", "/oauth-clients": "manager",
        "/webhooks": "admin", "/policies": "manager", "/security/": "manager",
        "/access-requests": "user", "/analytics/": "manager", "/monitoring/": "manager",
      };
      for (const [prefix, scope] of Object.entries(ADMIN_PREFIXES)) {
        if (pathname.startsWith(prefix)) {
          const userScopes = JSON.parse(localStorage.getItem("ggid_user_scopes") || '["user"]');
          const hasAdmin = userScopes.includes("admin");
          const hasManager = userScopes.includes("manager") || hasAdmin;
          if (scope === "manager" && !hasManager) {
            router.replace("/dashboard");
            return;
          }
          if (scope === "admin" && !hasAdmin) {
            router.replace("/dashboard");
            return;
          }
          break;
        }
      }
    } else {
      setIsAuthenticated(false);
      if (!isPublic) {
        router.push("/login");
      }
    }
    setChecked(true);
  }, [pathname, router]);

  // Listen for 401 events from api.ts to force logout without page reload
  useEffect(() => {
    const handleUnauthorized = () => {
      localStorage.removeItem("ggid_access_token");
      localStorage.removeItem("ggid_refresh_token");
      localStorage.removeItem("ggid_session_id");
      localStorage.removeItem("ggid_tenant_id");
      localStorage.removeItem("ggid_user_id");
      localStorage.removeItem("ggid_user_name");
      localStorage.removeItem("ggid_user_email");
      setIsAuthenticated(false);
      router.push("/login");
    };
    window.addEventListener("ggid:unauthorized", handleUnauthorized);
    return () => window.removeEventListener("ggid:unauthorized", handleUnauthorized);
  }, [router]);

  if (!checked) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-50 dark:bg-gray-950">
        <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p));

  // Public pages (login, register, etc.) render full-screen without sidebar
  if (isPublic || !isAuthenticated) {
    return <main className="min-h-screen dark:bg-gray-950">{children}</main>;
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
