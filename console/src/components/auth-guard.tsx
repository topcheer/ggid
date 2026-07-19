"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Sidebar } from "@/components/sidebar";

const PUBLIC_PATHS = ["/login", "/register", "/forgot-password", "/reset-password"];

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();

  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    const token = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") : null;
    const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p));

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
