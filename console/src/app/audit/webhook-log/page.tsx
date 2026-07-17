"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Webhook, Loader2, AlertCircle, X, Check, Clock, Zap,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface DeliveryEntry {
  id: string;
  webhook_url: string;
  event_type: string;
  status: "delivered" | "failed" | "retrying" | "pending";
  http_status: number;
  latency_ms: number;
  timestamp: string;
  attempt: number;
  error?: string;
}

const STATUS_CONFIG = {
  delivered: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: Check },
  failed: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: X },
  retrying: { color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30", icon: Zap },
  pending: { color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-700", icon: Clock },
};

export default function WebhookLogPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [entries, setEntries] = useState<DeliveryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const data = await apiFetch<{ entries?: DeliveryEntry[]; items?: DeliveryEntry[] }>("/api/v1/settings/webhooks/delivery-log?limit=50").catch(() => null);
        setEntries(data?.entries ?? data?.items ?? []);
      } catch { setError("Failed to load delivery log"); }
      finally { setLoading(false); }
    })();
  }, []);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const successCount = entries.filter((e) => e.status === "delivered").length;
  const failCount = entries.filter((e) => e.status === "failed").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Webhook className="h-6 w-6 text-indigo-600" /> {t("auditWebhookLog.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Timeline of all webhook delivery attempts with HTTP status and latency.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Total</p><p className="mt-1 text-2xl font-bold text-indigo-600">{entries.length}</p></div>
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Success Rate</p><p className="mt-1 text-2xl font-bold text-green-600">{entries.length > 0 ? Math.round((successCount / entries.length) * 100) : 0}%</p></div>
        <div className={cardCls}><p className="text-xs font-semibold uppercase text-gray-400">Failed</p><p className="mt-1 text-2xl font-bold text-red-600">{failCount}</p></div>
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : entries.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Webhook className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No webhook deliveries recorded.</p></div></div>
      : (
        <div className="space-y-2">
          {entries.map((e) => {
            const cfg = STATUS_CONFIG[e.status] ?? STATUS_CONFIG.pending;
            const StatusIcon = cfg.icon;
            return (
              <div key={e.id} className={`${cardCls} flex items-center gap-4 py-3`}>
                <div className={`rounded-lg ${cfg.bg} p-1.5`}><StatusIcon className={`h-4 w-4 ${cfg.color}`} /></div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm text-gray-700 dark:text-gray-300">{e.event_type}</span>
                    <span className="text-xs text-gray-400">attempt #{e.attempt}</span>
                  </div>
                  <p className="truncate font-mono text-xs text-gray-400">→ {e.webhook_url}</p>
                </div>
                <div className="hidden items-center gap-4 text-xs sm:flex">
                  <span className={`rounded px-1.5 py-0.5 font-mono font-medium ${e.http_status >= 200 && e.http_status < 300 ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : e.http_status > 0 ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 text-gray-400"}`}>{e.http_status || "—"}</span>
                  <span className="text-gray-400">{e.latency_ms}ms</span>
                </div>
                <div className="text-right">
                  <span className={`rounded-full ${cfg.bg} px-2 py-0.5 text-xs font-medium ${cfg.color}`}>{e.status}</span>
                  <p className="mt-0.5 text-xs text-gray-400">{new Date(e.timestamp).toLocaleTimeString()}</p>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
