"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Sidebar } from "@/components/sidebar";

const PUBLIC_PATHS = ["/login", "/register", "/forgot-password", "/reset-password", "/setup"];

// Check if system has been initialized (has any users).
// Returns null = unknown, true = initialized, false = needs setup.
async function checkSystemInitialized(): Promise<boolean | null> {
  try {
    const resp = await fetch("/api/v1/system/status", { method: "GET" });
    if (resp.ok) {
      const data = await resp.json();
      return data.initialized === true;
    }
    // Fallback: if status endpoint doesn't exist, assume initialized
    return true;
  } catch {
    return null;
  }
}

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();

  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    const token = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") : null;
    const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p));

    // If no token and not on a public path → check if system needs setup
    if (!token && !isPublic) {
      checkSystemInitialized().then((initialized) => {
        if (initialized === false) {
          // System not initialized → redirect to setup wizard
          router.replace("/setup");
        } else {
          // System initialized but not logged in → redirect to login
          setIsAuthenticated(false);
          router.push("/login");
        }
        setChecked(true);
      });
      return;
    }

    // If on /setup but system is already initialized → redirect to login
    if (pathname === "/setup" && token) {
      router.replace("/dashboard");
      setChecked(true);
      return;
    }

    if (token) {
      setIsAuthenticated(true);
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
