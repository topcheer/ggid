import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

// Avoid checking tenant on these paths
const SKIP_PATHS = [
  "/_next/",
  "/favicon.ico",
  "/manifest.json",
  "/icon.svg",
  "/apple-icon.png",
  "/sw.js",
  "/workbox-",
];

// In-memory cache: slug → { id, ts }
// Avoids hitting the API on every request for the same subdomain.
const cache = new Map<string, { valid: boolean; ts: number }>();
const CACHE_TTL = 60_000; // 1 minute

function getTenantSlugFromHost(host: string): string | null {
  // host format: slug.ggid-console.iot2.win or ggid-console.iot2.win
  // Also handle localhost / IP (no subdomain)
  if (!host || host.includes("localhost") || /^\d+\.\d+\.\d+\.\d+/.test(host)) {
    return null;
  }
  const parts = host.split(".");
  // Expected: slug.ggid-console.iot2.win → parts[0] = slug
  // If parts[0] === "ggid-console" → no subdomain
  if (parts.length >= 4 && parts[1] === "ggid-console") {
    if (parts[0] === "ggid-console") return null;
    return parts[0];
  }
  return null;
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const host = request.headers.get("host") || "";

  // Skip static assets
  if (SKIP_PATHS.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  const slug = getTenantSlugFromHost(host);
  if (!slug) {
    // No subdomain → default tenant, always allow
    return NextResponse.next();
  }

  // Check cache
  const cached = cache.get(slug);
  if (cached && Date.now() - cached.ts < CACHE_TTL) {
    if (!cached.valid) {
      return tenantNotFound();
    }
    return NextResponse.next();
  }

  // Resolve tenant via API
  try {
    const gatewayUrl = process.env.GATEWAY_URL || "http://localhost:8080";
    const resp = await fetch(
      `${gatewayUrl}/api/v1/tenants/resolve?slug=${encodeURIComponent(slug)}`,
      { signal: AbortSignal.timeout(3000) }
    );
    const valid = resp.ok;
    cache.set(slug, { valid, ts: Date.now() });
    if (!valid) {
      return tenantNotFound();
    }
  } catch {
    // API unavailable — fail open (let client-side handle it)
    // Don't cache failure to connect
  }

  return NextResponse.next();
}

function tenantNotFound() {
  const html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>404 — 租户不存在 / Tenant Not Found</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#0f172a;color:#e2e8f0;display:flex;align-items:center;justify-content:center;height:100vh}
.inner{text-align:center;max-width:480px;padding:0 24px}
.code{font-size:96px;font-weight:900;color:#1e293b;line-height:1;margin-bottom:8px}
h1{font-size:24px;color:#f87171;margin-bottom:12px}
p{font-size:15px;color:#94a3b8;line-height:1.6;margin-bottom:24px}
a{display:inline-block;padding:10px 24px;background:#6366f1;color:#fff;border-radius:8px;text-decoration:none;font-size:14px;font-weight:600}
a:hover{background:#4f46e5}
</style>
</head>
<body>
<div class="inner">
<div class="code">404</div>
<h1>租户不存在 / Tenant Not Found</h1>
<p>该子域名对应的租户不存在或已被停用。<br>The tenant for this subdomain does not exist or has been suspended.</p>
<a href="https://ggid-console.iot2.win">← 返回主页 / Go to Console</a>
</div>
</body>
</html>`;

  return new NextResponse(html, {
    status: 404,
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static, _next/image (static files)
     * - favicon.ico, icon.svg, manifest.json, etc (already in SKIP_PATHS)
     */
    "/((?!_next/static|_next/image).*)",
  ],
};
