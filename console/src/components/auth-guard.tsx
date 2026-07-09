"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

const PUBLIC_PATHS = ["/login"];

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(p));
    if (isPublic) return;

    const token = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") : null;
    if (!token) {
      router.push("/login");
    }
  }, [pathname, router]);

  return <>{children}</>;
}
