"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Activity, CheckCircle, XCircle, AlertTriangle } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface EndpointStatus {
  name: string;
  url: string;
  status: "up" | "down" | "degraded";
  response_time_ms: number;
  last_check: string;
}

interface HealthData {
  endpoints: EndpointStatus[];
  cert_expiry_days: number;
  last_check: string;
  failover_status: "active" | "standby" | "failed";
}

const statusConfig: Record<string, { color: string; icon: typeof CheckCircle }> = {
  up: { color: "text-green-600", icon: CheckCircle },
  degraded: { color: "text-yellow-600", icon: AlertTriangle },
  down: { color: "text-red-600", icon: XCircle },
};

export default function OAuthHealthCheckPage() {
  const t = useTranslations();
  const [data, setData] = useState<HealthData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/health-check", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-green-500" />{t("oauthHealthCheck.title")}</h1><p className="text-sm text-gray-500 mt-1">Monitor all OAuth endpoint health with response times and cert status.</p></div>

      {data && (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Cert Expiry</span><p className={"text-xl font-bold mt-1 " + (data.cert_expiry_days <= 30 ? "text-red-600" : data.cert_expiry_days <= 90 ? "text-yellow-600" : "text-green-600")}>{data.cert_expiry_days} days</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Failover Status</span><p className={"text-xl font-bold mt-1 " + (data.failover_status === "active" ? "text-green-600" : data.failover_status === "standby" ? "text-yellow-600" : "text-red-600")}>{data.failover_status}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Last Check</span><p className="text-sm font-bold mt-1">{data.last_check}</p></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.endpoints.map((e: any) => { const cfg = statusConfig[e.status]; const Icon = cfg.icon; return (
              <div key={e.name} className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between"><span className="font-medium text-sm capitalize">{e.name}</span><Icon className={"w-5 h-5 " + cfg.color} /></div><p className="font-mono text-xs text-gray-400 mt-1 truncate">{e.url}</p><div className="flex items-center justify-between mt-2"><span className={"text-xs font-bold " + cfg.color}>{e.status}</span><span className={"text-xs " + (e.response_time_ms > 500 ? "text-red-500" : e.response_time_ms > 200 ? "text-yellow-500" : "text-green-500")}>{e.response_time_ms}ms</span></div></div>
            ); })}
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
