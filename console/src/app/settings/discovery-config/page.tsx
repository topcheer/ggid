"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Globe, RefreshCw, CheckCircle, XCircle } from "lucide-react";

interface DiscoveryInfo {
  issuer: string;
  well_known_url: string;
  supported_scopes: { name: string; description: string }[];
  supported_grants: string[];
  signing_algs: string[];
  userinfo_endpoint: string;
  userinfo_status: "up" | "down";
  jwks_uri: string;
  jwks_last_refresh: string;
  jwks_key_count: number;
}

export default function DiscoveryConfigPage() {
  const t = useTranslations();
  const [data, setData] = useState<DiscoveryInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/discovery-config", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const refreshJwks = async () => {
    setRefreshing(true);
    try { await fetch("/api/v1/oauth/discovery-config/jwks-refresh", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
    finally { setRefreshing(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-blue-500" /> {t("backend.discoveryConfig.title")}</h1><p className="text-sm text-gray-500 mt-1">OIDC discovery endpoint configuration and JWKS management.</p></div>

      {data && (
        <>
          <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div><span className="text-sm text-gray-500">{t("backend.discoveryConfig.issuer")}</span><p className="font-mono text-sm font-medium">{data.issuer}</p></div>
            <div><span className="text-sm text-gray-500">Well-Known URL</span><p className="font-mono text-xs text-blue-600">{data.well_known_url}</p></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("backend.discoveryConfig.supportedScopes")}</h3><div className="space-y-1">{data.supported_scopes.map((s) => <div key={s.name} className="flex items-center gap-2"><span className="font-mono text-xs font-medium">{s.name}</span><span className="text-xs text-gray-400">{s.description}</span></div>)}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("backend.discoveryConfig.supportedGrants")}</h3><div className="flex flex-wrap gap-2">{data.supported_grants.map((g) => <span key={g} className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{g}</span>)}</div><h3 className="text-sm font-semibold mt-4 mb-3">{t("backend.discoveryConfig.signingAlgorithms")}</h3><div className="flex flex-wrap gap-2">{data.signing_algs.map((a) => <span key={a} className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{a}</span>)}</div></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between"><div><span className="text-sm text-gray-500">{t("backend.discoveryConfig.userinfoEndpoint")}</span><p className="font-mono text-xs mt-1">{data.userinfo_endpoint}</p></div>{data.userinfo_status === "up" ? <CheckCircle className="w-5 h-5 text-green-500" /> : <XCircle className="w-5 h-5 text-red-500" />}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between"><div><span className="text-sm text-gray-500">{t("backend.discoveryConfig.jwksUri")}</span><p className="font-mono text-xs mt-1">{data.jwks_uri}</p><p className="text-xs text-gray-400 mt-1">{data.jwks_key_count} keys - refreshed {data.jwks_last_refresh}</p></div><button onClick={refreshJwks} disabled={refreshing} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-xs flex items-center gap-1"><RefreshCw className={"w-3 h-3 " + (refreshing ? "animate-spin" : "")} /> Refresh</button></div></div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
