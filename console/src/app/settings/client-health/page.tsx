"use client";

import { useState, useEffect, useCallback } from "react";
import { Heart, Activity, AlertTriangle, KeyRound, ShieldCheck } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ClientHealth {
  client_id: string;
  client_name: string;
  status: "healthy" | "warning" | "critical";
  active_tokens: number;
  error_rate: number;
  secret_expires: string | null;
  cert_expires: string | null;
  last_error: string | null;
}

const statusConfig: Record<string, { color: string; bg: string; icon: string }> = {
  healthy: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30 dark:text-green-400", icon: "text-green-500" },
  warning: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400", icon: "text-yellow-500" },
  critical: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30 dark:text-red-400", icon: "text-red-500" },
};

export default function ClientHealthPage() {
  const t = useTranslations();

  const [clients, setClients] = useState<ClientHealth[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/client-health", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setClients(data.clients || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const summary = { healthy: clients.filter((c) => c.status === "healthy").length, warning: clients.filter((c) => c.status === "warning").length, critical: clients.filter((c) => c.status === "critical").length };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Heart className="w-6 h-6 text-pink-500" /> {t("clientHealth.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Monitor OAuth client health, token usage, and credential expiry.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><ShieldCheck className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Healthy</span><p className="text-xl font-bold text-green-600">{summary.healthy}</p></div></div>
        <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><AlertTriangle className="w-8 h-8 text-yellow-500" /><div><span className="text-sm text-gray-500">Warning</span><p className="text-xl font-bold text-yellow-600">{summary.warning}</p></div></div>
        <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><AlertTriangle className="w-8 h-8 text-red-500" /><div><span className="text-sm text-gray-500">Critical</span><p className="text-xl font-bold text-red-600">{summary.critical}</p></div></div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {clients.map((c) => {
          const cfg = statusConfig[c.status];
          return (
            <div key={c.client_id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <div><span className="font-semibold">{c.client_name}</span><p className="text-xs text-gray-400 font-mono">{c.client_id}</p></div>
                <span className={`px-2 py-1 rounded text-xs font-medium ${cfg.bg}`}>{c.status}</span>
              </div>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div className="flex items-center gap-1"><Activity className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Tokens</span><span className="font-medium ml-auto">{c.active_tokens}</span></div>
                <div className="flex items-center gap-1"><AlertTriangle className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Errors</span><span className={`font-medium ml-auto ${c.error_rate > 5 ? "text-red-600" : ""}`}>{c.error_rate}%</span></div>
                <div className="flex items-center gap-1"><KeyRound className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Secret</span><span className={`text-xs ml-auto ${c.secret_expires ? "text-yellow-600" : "text-green-600"}`}>{c.secret_expires || "OK"}</span></div>
                <div className="flex items-center gap-1"><ShieldCheck className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Cert</span><span className={`text-xs ml-auto ${c.cert_expires ? "text-yellow-600" : "text-green-600"}`}>{c.cert_expires || "OK"}</span></div>
              </div>
              {c.last_error && <div className="text-xs text-red-500 border-t dark:border-gray-800 pt-2">{c.last_error}</div>}
            </div>
          );
        })}
        {clients.length === 0 && !loading && <div className="col-span-full text-center text-gray-500 py-8">No clients found.</div>}
      </div>
    </div>
  );
}
