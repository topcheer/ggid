"use client";
import { useEffect, useState } from "react";
import { useOAuthBackpressureConfig, OAuthBackpressureConfig, DegradationRule } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthBackpressureConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthBackpressureConfig();
  const [form, setForm] = useState<OAuthBackpressureConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useTranslations();
  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">{t("oauthBackpressure.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("oauthBackpressure.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("oauthBackpressure.title")}</h1>
      <p className="text-gray-600">{t("oauthBackpressure.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("oauthBackpressure.queueConcurrency")}</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.per_client_fair_queueing} onChange={(e) => setForm({ ...form, per_client_fair_queueing: e.target.checked })} className="w-4 h-4" /><label>{t("oauthBackpressure.perClientFairQueue")}</label></div>
        <div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">{t("oauthBackpressure.maxConcurrent")}</label><input type="number" value={form.max_concurrent_token_requests} onChange={(e) => setForm({ ...form, max_concurrent_token_requests: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">{t("oauthBackpressure.circuitBreakerThreshold")}</label><input type="number" value={form.circuit_breaker_threshold} onChange={(e) => setForm({ ...form, circuit_breaker_threshold: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div>
        <div><label className="block text-sm font-medium mb-1">{t("oauthBackpressure.queueOverflow")}</label><select value={form.queue_overflow_action} onChange={(e) => setForm({ ...form, queue_overflow_action: e.target.value as OAuthBackpressureConfig["queue_overflow_action"] })} className="border rounded px-3 py-2"><option value="reject">{t("oauthBackpressure.reject")}</option><option value="defer">{t("oauthBackpressure.defer")}</option></select></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.rate_limit_headers} onChange={(e) => setForm({ ...form, rate_limit_headers: e.target.checked })} className="w-4 h-4" /><label>{t("oauthBackpressure.rateLimitHeaders")}</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("oauthBackpressure.gracefulDegradation")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("oauthBackpressure.metric")}</th><th scope="col">{t("oauthBackpressure.threshold")}</th><th>{t("oauthBackpressure.action")}</th></tr></thead><tbody>{form.graceful_degradation_rules.map((r: DegradationRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{r.metric}</td><td>{r.threshold}</td><td className="text-xs">{r.action}</td></tr>))}</tbody></table></div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
