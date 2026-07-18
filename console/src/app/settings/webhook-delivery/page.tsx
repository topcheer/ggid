"use client";

import { useState, useEffect } from "react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import {
  Webhook, RefreshCw, AlertCircle, Loader2, X, RotateCcw,
} from "lucide-react";

interface FailedDelivery {
  id: string;
  webhook_url: string;
  event_type: string;
  attempts: number;
  last_error: string;
  last_attempt: string;
  status: "pending_retry" | "exhausted" | "retrying";
}

export default function WebhookDeliveryPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [deliveries, setDeliveries] = useState<FailedDelivery[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retrying, setRetrying] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const data = await apiFetch<{ entries?: FailedDelivery[]; items?: FailedDelivery[] }>("/api/v1/settings/webhooks/delivery-log?status=failed").catch(() => null);
        setDeliveries(data?.entries ?? data?.items ?? []);
      } catch { setError(t("webhookDelivery.failedToLoad")); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleRetry = async (id: string) => {
    setRetrying(id);
    try { await apiFetch(`/api/v1/settings/webhooks/deliveries/${id}/retry`, { method: "POST" }); setDeliveries((prev) => prev.filter((d: any) => d.id !== id)); }
    catch { setError(t("webhookDelivery.retryFailed")); }
    finally { setRetrying(null); }
  };

  const handleRetryAll = async () => {
    setRetrying("all");
    try { await apiFetch("/api/v1/settings/webhooks/deliveries/retry-all", { method: "POST" }); setDeliveries([]); }
    catch { setError(t("webhookDelivery.bulkRetryFailed")); }
    finally { setRetrying(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Webhook className="h-6 w-6 text-indigo-600" /> {t("webhookDelivery.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("webhookDelivery.subtitle")}</p>
        </div>
        {deliveries.length > 0 && <button onClick={handleRetryAll} disabled={retrying === "all"} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
          {retrying === "all" ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />}
          {t("webhookDelivery.retryAll").replace("{count}", String(deliveries.length))}
        </button>}
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
        <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
      </div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : deliveries.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Webhook className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("webhookDelivery.empty")}</p></div></div>
      : (
        <div className="space-y-3">
          {deliveries.map((d: any) => (
            <div key={d.id} className={`${cardCls} ${d.status === "exhausted" ? "border-red-200 dark:border-red-800" : ""}`}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-sm text-gray-700 dark:text-gray-300">{d.event_type}</span>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${d.status === "exhausted" ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400"}`}>
                      {d.status === "exhausted" ? t("webhookDelivery.exhausted") : t("webhookDelivery.pendingRetry")}
                    </span>
                  </div>
                  <p className="mt-1 truncate font-mono text-xs text-gray-400">→ {d.webhook_url}</p>
                  <div className="mt-2 flex items-center gap-3 text-xs text-gray-400">
                    <span className="rounded bg-red-50 px-1.5 py-0.5 text-red-600 dark:bg-red-900/20">{t("webhookDelivery.attempts").replace("{count}", String(d.attempts))}</span>
                    <span>{t("webhookDelivery.last")}: {new Date(d.last_attempt).toLocaleString()}</span>
                  </div>
                  <p className="mt-1 text-xs text-red-400">{t("common.error")}: {d.last_error}</p>
                </div>
                {d.status !== "exhausted" && <button onClick={() => handleRetry(d.id)} disabled={retrying === d.id} className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                  {retrying === d.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                  {t("common.retry")}
                </button>}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
