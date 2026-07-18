"use client";

import { useState, useCallback, useEffect } from "react";
import { Code2, ChevronRight, ChevronDown, Copy, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface DecodedToken {
  header: Record<string, unknown>;
  payload: {
    iss: string;
    sub: string;
    aud: string | string[];
    exp: number;
    iat: number;
    scope: string;
    [key: string]: unknown;
  };
  signature: string;
}

function JsonNode({ label, value, depth }: { label: string; value: unknown; depth: number }) {
  const [expanded, setExpanded] = useState(depth < 2);
  const isObject = value !== null && typeof value === "object";
  const isArray = Array.isArray(value);

  if (!isObject) {
    return (
      <div className="flex items-start gap-1 py-0.5" style={{ paddingLeft: `${depth * 16}px` }}>
        <span className="text-gray-500">{label}:</span>
        <span className={typeof value === "string" ? "text-green-600" : typeof value === "number" ? "text-blue-600" : "text-purple-600"}>{JSON.stringify(value)}</span>
      </div>
    );
  }

  const entries = Object.entries(value as Record<string, unknown>);
  return (
    <div>
      <button onClick={() => setExpanded(!expanded)} aria-label="Toggle node" className="flex items-center gap-1 py-0.5 hover:bg-gray-50 dark:hover:bg-gray-900/30 rounded px-1" style={{ paddingLeft: `${depth * 16}px` }}>
        {expanded ? <ChevronDown className="w-3 h-3 text-gray-400" /> : <ChevronRight className="w-3 h-3 text-gray-400" />}
        <span className="text-gray-500">{label}:</span>
        <span className="text-gray-400 text-xs">{isArray ? `[${entries.length}]` : `{${entries.length}}`}</span>
      </button>
      {expanded && (
        <div>
          {entries.map(([k, v]) => (
            <JsonNode key={k} label={k} value={v} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}

export default function TokenClaimsPage() {
  const t = useTranslations();
  const [token, setToken] = useState("");
  const [decoded, setDecoded] = useState<DecodedToken | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/v1/auth/sessions/anomaly-score", {
      headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(() => setLoading(false))
      .catch(() => setLoading(false));
  }, []);

  const decode = useCallback(() => {
    setError("");
    setDecoded(null);
    if (!token.trim()) return;
    try {
      const parts = token.trim().split(".");
      if (parts.length !== 3) throw new Error("Invalid token format (expected 3 parts)");
      const decodeB64 = (s: string) => {
        const padded = s.replace(/-/g, "+").replace(/_/g, "/");
        return JSON.parse(atob(padded));
      };
      setDecoded({
        header: decodeB64(parts[0]),
        payload: decodeB64(parts[1]),
        signature: parts[2],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to decode token");
    }
  }, [token]);

  const standardClaims = decoded ? ["iss", "sub", "aud", "exp", "iat", "scope"].filter((k) => k in decoded.payload) : [];
  const customClaims = decoded ? Object.keys(decoded.payload).filter((k) => !["iss", "sub", "aud", "exp", "iat", "scope"].includes(k)) : [];
  const expDate = decoded?.payload.exp ? new Date(decoded.payload.exp * 1000).toLocaleString() : "";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Code2 className="w-6 h-6 text-blue-500" /> {t("tokenClaims.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Decode and inspect JWT access tokens.</p>
      </div>

      {/* Token input */}
      <div className="space-y-2">
        <textarea aria-label="Paste a JWT token here..." value={token} onChange={(e) => setToken(e.target.value)} placeholder="Paste a JWT token here..." rows={4} spellCheck={false} className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono resize-y" />
        <div className="flex items-center gap-2">
          <button onClick={decode} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700">Decode Token</button>
          <button onClick={() => { setToken(""); setDecoded(null); setError(""); }} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Clear</button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2 text-sm text-red-600">
          <AlertCircle className="w-5 h-5" /> {error}
        </div>
      )}

      {decoded && (
        <div className="space-y-4">
          {/* Standard claims grid */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3">Standard Claims</h3>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {standardClaims.map((key) => (
                <div key={key} className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3">
                  <span className="text-xs text-gray-400 font-mono">{key}</span>
                  <p className="text-sm font-medium mt-0.5 break-all">
                    {key === "exp" ? expDate : key === "scope" ? String(decoded.payload[key]).split(" ").join(", ") : String(decoded.payload[key])}
                  </p>
                </div>
              ))}
            </div>
          </div>

          {/* Scope badges */}
          {decoded.payload.scope && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-2">Scopes</h3>
              <div className="flex flex-wrap gap-1">
                {String(decoded.payload.scope).split(" ").map((s: any, i: number) => (
                  <span key={i} className="px-2 py-0.5 rounded text-xs font-mono bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400">{s}</span>
                ))}
              </div>
            </div>
          )}

          {/* JSON tree view */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Code2 className="w-4 h-4" /> Full Payload (JSON Tree)</h3>
            </div>
            <div className="p-4 font-mono text-xs overflow-x-auto">
              <JsonNode label="payload" value={decoded.payload} depth={0} />
            </div>
          </div>

          {/* Custom claims */}
          {customClaims.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-3">Custom Claims ({customClaims.length})</h3>
              <div className="space-y-1">
                {customClaims.map((key) => (
                  <div key={key} className="flex items-start gap-2 text-sm">
                    <span className="font-mono text-gray-400">{key}:</span>
                    <span className="font-mono text-green-600">{JSON.stringify(decoded.payload[key])}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Header + Signature */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-2 text-sm">Header</h3>
              <pre className="text-xs font-mono bg-gray-50 dark:bg-gray-900/50 rounded p-3 overflow-x-auto">{JSON.stringify(decoded.header, null, 2)}</pre>
            </div>
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-2 text-sm">Signature</h3>
              <p className="text-xs font-mono break-all text-gray-400">{decoded.signature.substring(0, 40)}...</p>
            </div>
          </div>
        </div>
      )}

      {!decoded && !error && <p className="text-sm text-gray-500 text-center py-8">Paste a JWT token above and click Decode.</p>}
    </div>
  );
}
