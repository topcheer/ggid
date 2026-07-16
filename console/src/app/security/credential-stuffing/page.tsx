"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldBan, Loader2, AlertCircle, X, Check, Ban, ToggleLeft, ToggleRight,
  Globe, Activity,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface StuffingData {
  detection_enabled: boolean;
  total_attempts: number;
  success_rate: number;
  blocked_ips: { ip: string; attempts: number; blocked_at: string }[];
  ip_spread: { ip: string; attempts: number }[];
}

export default function CredentialStuffingPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [data, setData] = useState<StuffingData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggling, setToggling] = useState(false);

  useState(() => {
    (async () => {
      try { setData(await apiFetch<StuffingData>("/api/v1/auth/credential-stuffing/status").catch(() => null)); }
      catch { setError("Failed to load credential stuffing data"); }
      finally { setLoading(false); }
    })();
  });

  const handleToggle = async () => {
    if (!data) return;
    setToggling(true);
    try { await apiFetch("/api/v1/auth/credential-stuffing/toggle", { method: "POST", body: JSON.stringify({ enabled: !data.detection_enabled }) }); setData((p) => p ? { ...p, detection_enabled: !p.detection_enabled } : null); }
    catch { setError("Toggle failed"); }
    finally { setToggling(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const maxAttempts = Math.max(...(data?.ip_spread ?? []).map((s) => s.attempts), 1);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldBan className="h-6 w-6 text-red-600" /> {t("securityCredentialStuffing.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Automated detection of credential stuffing attacks across login endpoints.</p>
        </div>
        {data && <button onClick={handleToggle} disabled={toggling} className="flex items-center gap-2">{toggling ? <Loader2 className="h-5 w-5 animate-spin" /> : data.detection_enabled ? <ToggleRight className="h-6 w-6 text-green-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}<span className="text-sm font-medium">{data.detection_enabled ? "Detection On" : "Detection Off"}</span></button>}
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : data ? (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Attempts</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{data.total_attempts}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Check className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Success Rate</span></div><p className={`mt-2 text-2xl font-bold ${data.success_rate > 5 ? "text-red-600" : "text-green-600"}`}>{data.success_rate}%</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Ban className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Blocked IPs</span></div><p className="mt-2 text-2xl font-bold text-red-600">{data.blocked_ips.length}</p></div>
          </div>

          {/* IP spread chart */}
          {data.ip_spread.length > 0 && (
            <div className={cardCls}>
              <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><Globe className="h-4 w-4" /> IP Distribution (Top 10)</h3>
              <div className="space-y-2">
                {data.ip_spread.slice(0, 10).map((s) => (
                  <div key={s.ip}>
                    <div className="flex items-center justify-between text-sm"><span className="font-mono text-gray-600 dark:text-gray-300">{s.ip}</span><span className="font-bold text-red-500">{s.attempts}</span></div>
                    <div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-red-400" style={{ width: `${(s.attempts / maxAttempts) * 100}%` }} /></div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Blocked IPs */}
          {data.blocked_ips.length > 0 && (
            <div>
              <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Blocked IPs</h2>
              <div className="space-y-2">
                {data.blocked_ips.map((b) => (
                  <div key={b.ip} className={`${cardCls} flex items-center justify-between py-3`}><div><span className="font-mono text-sm text-gray-700 dark:text-gray-300">{b.ip}</span><p className="text-xs text-gray-400">Blocked {new Date(b.blocked_at).toLocaleString()}</p></div><span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400">{b.attempts} attempts</span></div>
                ))}
              </div>
            </div>
          )}
        </>
      ) : <div className={cardCls}><div className="py-12 text-center"><ShieldBan className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No credential stuffing data.</p></div></div>}
    </div>
  );
}
