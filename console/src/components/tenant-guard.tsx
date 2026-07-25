"use client";

import { useEffect, useState } from "react";
import {
  getTenantSlugFromSubdomain,
  resolveTenantSlug,
} from "@/lib/api-config";

type State = "checking" | "valid" | "not-found";

/**
 * Checks whether the subdomain maps to an existing tenant.
 * - No subdomain (ggid-console.iot2.win) → always valid (default tenant)
 * - Subdomain exists and tenant resolves → valid
 * - Subdomain exists but tenant not found → renders 404
 */
export function TenantGuard({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<State>("checking");

  useEffect(() => {
    const slug = getTenantSlugFromSubdomain();
    if (!slug) {
      // Direct access (no subdomain) — default tenant
      setState("valid");
      return;
    }

    let cancelled = false;
    resolveTenantSlug(slug).then((id) => {
      if (cancelled) return;
      setState(id ? "valid" : "not-found");
    });

    return () => {
      cancelled = true;
    };
  }, []);

  if (state === "checking") {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100vh",
          background: "#0f172a",
          color: "#94a3b8",
          fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
        }}
      >
        <div style={{ textAlign: "center" }}>
          <div
            style={{
              width: 36,
              height: 36,
              border: "3px solid #1e293b",
              borderTopColor: "#6366f1",
              borderRadius: "50%",
              animation: "spin 0.7s linear infinite",
              margin: "0 auto 16px",
            }}
          />
          <span>Loading…</span>
        </div>
        <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
      </div>
    );
  }

  if (state === "not-found") {
    return <TenantNotFound />;
  }

  return <>{children}</>;
}

function TenantNotFound() {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        height: "100vh",
        background: "#0f172a",
        color: "#e2e8f0",
        fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
      }}
    >
      <div style={{ textAlign: "center", maxWidth: 480, padding: "0 24px" }}>
        <div
          style={{
            fontSize: 96,
            fontWeight: 900,
            color: "#1e293b",
            lineHeight: 1,
            marginBottom: 8,
          }}
        >
          404
        </div>
        <h1
          style={{
            fontSize: 24,
            color: "#f87171",
            marginBottom: 12,
          }}
        >
          租户不存在 / Tenant Not Found
        </h1>
        <p
          style={{
            fontSize: 15,
            color: "#94a3b8",
            lineHeight: 1.6,
            marginBottom: 24,
          }}
        >
          该子域名对应的租户不存在或已被停用。
          <br />
          The tenant for this subdomain does not exist or has been suspended.
        </p>
        <a
          href="https://ggid-console.iot2.win"
          style={{
            display: "inline-block",
            padding: "10px 24px",
            background: "#6366f1",
            color: "#fff",
            borderRadius: 8,
            textDecoration: "none",
            fontSize: 14,
            fontWeight: 600,
          }}
        >
          ← 返回主页 / Go to Console
        </a>
      </div>
    </div>
  );
}
