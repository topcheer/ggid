"use client";
import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface WebhookSubscription {
  id: string;
  url: string;
  events: string[];
  enabled: boolean;
  last_delivery: string;
  status: "delivered" | "failed" | "pending";
}

interface DeliveryRecord {
  timestamp: string;
  event: string;
  status_code: number;
  latency_ms: number;
  success: boolean;
}

export default function WebhookSubscriptionsPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/webhooks", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : t("webhookSubs.failedToLoad"));
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [t]);

  const [showAdd, setShowAdd] = useState(false);const [subs] = useState<WebhookSubscription[]>([
    { id: "wh-001", url: "https://hooks.example.com/users", events: ["user.created", "user.updated"], enabled: true, last_delivery: "2025-01-15 16:01", status: "delivered" },
    { id: "wh-002", url: "https://api.slack.com/hooks/xyz", events: ["auth.login_failed", "policy.violation"], enabled: true, last_delivery: "2025-01-15 15:45", status: "delivered" },
    { id: "wh-003", url: "https://legacy.internal/api/audit", events: ["audit.*"], enabled: false, last_delivery: "2025-01-14 09:00", status: "failed" },
  ]);

  const [history] = useState<DeliveryRecord[]>([
    { timestamp: "16:01:23", event: "user.created", status_code: 200, latency_ms: 145, success: true },
    { timestamp: "15:58:01", event: "user.updated", status_code: 200, latency_ms: 89, success: true },
    { timestamp: "15:45:15", event: "auth.login_failed", status_code: 500, latency_ms: 3021, success: false },
    { timestamp: "15:30:00", event: "user.created", status_code: 200, latency_ms: 167, success: true },
  ]);

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">{t("common.error")}: {error}</div>;

  const eventCatalog = ["user.created", "user.updated", "user.deleted", "auth.login", "auth.login_failed", "auth.logout", "policy.violation", "role.assigned", "audit.*"];

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">{t("webhookSubs.title")}</h1>
      <p className="text-gray-600">{t("webhookSubs.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4"><h2 className="text-lg font-semibold">{t("webhookSubs.subscriptions")}</h2><button onClick={() => setShowAdd(!showAdd)} className="px-4 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700">{t("webhookSubs.addSubscription")}</button></div>
        {showAdd && (<div className="mb-4 border rounded p-4 space-y-3 bg-gray-50"><div><label className="block text-sm font-medium mb-1">{t("webhookSubs.url")}</label><input aria-label="Toggle option" autoComplete="current-password" type="text" placeholder={t("webhookSubs.urlPlaceholder")} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">{t("webhookSubs.eventTypes")}</label><div className="flex flex-wrap gap-2">{eventCatalog.map((e) => (<label key={e} className="flex items-center gap-1 text-sm"><input type="checkbox" className="w-4 h-4" />{e}</label>))}</div></div><div><label className="block text-sm font-medium mb-1">{t("webhookSubs.secret")}</label><input type="password" placeholder={t("webhookSubs.secretPlaceholder")} className="border rounded px-3 py-2 w-full" /></div><button className="px-4 py-2 bg-green-600 text-white rounded text-sm" aria-label="Action">{t("webhookSubs.create")}</button></div>)}
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("webhookSubs.url")}</th><th scope="col">{t("webhookSubs.eventTypes")}</th><th>{t("common.enabled")}</th><th>{t("webhookSubs.lastDelivery")}</th><th>{t("common.status")}</th><th>{t("webhookSubs.actions")}</th></tr></thead><tbody>
          {subs.map((s: WebhookSubscription, i: number) => (<tr key={i} className="border-b hover:bg-gray-50"><td className="py-2 font-mono text-xs break-all max-w-xs">{s.url}</td><td><div className="flex flex-wrap gap-1">{s.events.map((e) => <span key={e} className="px-1.5 py-0.5 bg-purple-100 text-purple-700 rounded text-xs">{e}</span>)}</div></td><td>{s.enabled ? t("webhookSubs.yes") : t("webhookSubs.no")}</td><td className="text-xs text-gray-500">{s.last_delivery}</td><td><span className={`px-2 py-1 rounded text-xs ${s.status === "delivered" ? "bg-green-100 text-green-700" : s.status === "failed" ? "bg-red-100 text-red-700" : "bg-yellow-100 text-yellow-700"}`}>{s.status}</span></td><td className="flex gap-2"><button aria-label="action" className="text-xs text-blue-600 hover:underline">{t("common.test")}</button><button className="text-xs text-red-600 hover:underline" aria-label="Action">{t("common.delete")}</button></td></tr>))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("webhookSubs.deliveryHistory")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("webhookSubs.timestamp")}</th><th scope="col">{t("webhookSubs.event")}</th><th>{t("webhookSubs.statusCode")}</th><th>{t("webhookSubs.latency")}</th><th>{t("webhookSubs.result")}</th></tr></thead><tbody>
          {history.map((h: DeliveryRecord, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono text-xs text-gray-500">{h.timestamp}</td><td className="font-mono text-xs">{h.event}</td><td>{h.status_code}</td><td>{h.latency_ms}ms</td><td><span className={`px-2 py-1 rounded text-xs ${h.success ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{h.success ? t("webhookSubs.ok") : t("webhookSubs.failed")}</span></td></tr>))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("webhookSubs.eventTypeCatalog")}</h2>
        <div className="flex flex-wrap gap-2">{eventCatalog.map((e) => <span key={e} className="px-2 py-1 bg-gray-100 rounded text-xs font-mono">{e}</span>)}</div>
      </div>
    </div>
  );
}
