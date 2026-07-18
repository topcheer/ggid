"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldAlert, Send, Loader2, AlertCircle, X, Check, Bell,
  Users, Database,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface BreachStatus {
  hibp_enabled: boolean;
  last_check: string;
  total_breaches: number;
  affected_users: number;
  notified_users: number;
  breaches: { name: string; date: string; affected_count: number; notified: boolean }[];
}

export default function PasswordBreachPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [status, setStatus] = useState<BreachStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notifying, setNotifying] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setStatus(await apiFetch<BreachStatus>("/api/v1/auth/password-breach/status").catch(() => null)); }
      catch { setError("Failed to load breach status"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleNotify = async (breachName: string) => {
    setNotifying(breachName);
    try {
      await apiFetch("/api/v1/auth/password-breach/notify", { method: "POST", body: JSON.stringify({ breach_name: breachName }) });
      setStatus(await apiFetch<BreachStatus>("/api/v1/auth/password-breach/status").catch(() => status));
    } catch { setError("Notification failed"); }
    finally { setNotifying(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const enabledCls = status?.hibp_enabled ? "bg-green-100 dark:bg-green-900/30" : "bg-gray-100 dark:bg-gray-700";
  const enabledIcon = status?.hibp_enabled ? "text-green-600" : "text-gray-400";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <ShieldAlert className="h-6 w-6 text-red-600" /> Password Breach Monitoring
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">HIBP integration for detecting compromised passwords and notifying affected users.</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : status ? (
        <>
          {/* HIBP status */}
          <div className={cardCls}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`rounded-lg p-2 ${enabledCls}`}>
                  <Database className={`h-5 w-5 ${enabledIcon}`} />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-800 dark:text-gray-200">HIBP Integration</h3>
                  <p className="text-sm text-gray-400">Last check: {new Date(status.last_check).toLocaleString()}</p>
                </div>
              </div>
              <span className={`rounded-full px-3 py-1 text-xs font-medium ${status.hibp_enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>
                {status.hibp_enabled ? "Active" : "Inactive"}
              </span>
            </div>
          </div>

          {/* Summary */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}>
              <div className="flex items-center gap-2"><ShieldAlert className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Breaches</span></div>
              <p className="mt-2 text-2xl font-bold text-red-600">{status.total_breaches}</p>
            </div>
            <div className={cardCls}>
              <div className="flex items-center gap-2"><Users className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">Affected Users</span></div>
              <p className="mt-2 text-2xl font-bold text-orange-600">{status.affected_users}</p>
            </div>
            <div className={cardCls}>
              <div className="flex items-center gap-2"><Bell className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Notified</span></div>
              <p className="mt-2 text-2xl font-bold text-green-600">{status.notified_users}</p>
            </div>
          </div>

          {/* Breach list */}
          {status.breaches.length > 0 && (
            <div className="space-y-3">
              {status.breaches.map((b: any) => (
                <div key={b.name} className={cardCls}>
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-800 dark:text-gray-200">{b.name}</span>
                        <span className="text-xs text-gray-400">{new Date(b.date).toLocaleDateString()}</span>
                      </div>
                      <p className="mt-1 flex items-center gap-1 text-sm text-orange-500"><Users className="h-3 w-3" />{b.affected_count} users affected</p>
                    </div>
                    {b.notified ? (
                      <span className="flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-700 dark:bg-green-900/30 dark:text-green-400"><Check className="h-3 w-3" />Notified</span>
                    ) : (
                      <button onClick={() => handleNotify(b.name)} disabled={notifying === b.name} className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                        {notifying === b.name ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Send className="h-3.5 w-3.5" />}Notify Users
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      ) : (
        <div className={cardCls}>
          <div className="py-12 text-center">
            <ShieldAlert className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-4 text-sm text-gray-400">No breach data available.</p>
          </div>
        </div>
      )}
    </div>
  );
}
