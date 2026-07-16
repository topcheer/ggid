"use client";
import { useEffect, useState } from "react";
import { useIdentityTokenPrefetchConfig, IdentityTokenPrefetchConfig, AppIntegration } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityTokenPrefetchConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useIdentityTokenPrefetchConfig();
  const [form, setForm] = useState<IdentityTokenPrefetchConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useTranslations();
  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">{t("idTokenPrefetch.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("idTokenPrefetch.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("idTokenPrefetch.title")}</h1>
      <p className="text-gray-600">{t("idTokenPrefetch.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("idTokenPrefetch.prefetchSettings")}</h2>
        <div><label className="block text-sm font-medium mb-1">{t("idTokenPrefetch.preemptiveThreshold")}: {form.preemptive_refresh_threshold_pct}%</label><input type="range" min={10} max={90} value={form.preemptive_refresh_threshold_pct} onChange={(e) => setForm({ ...form, preemptive_refresh_threshold_pct: parseInt(e.target.value) })} className="w-full" /></div>
        <div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">{t("idTokenPrefetch.backgroundRotation")}</label><input type="number" value={form.background_rotation_interval} onChange={(e) => setForm({ ...form, background_rotation_interval: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">{t("idTokenPrefetch.gracePeriod")}</label><input type="number" value={form.grace_period_seconds} onChange={(e) => setForm({ ...form, grace_period_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div>
        <div><label className="block text-sm font-medium mb-1">{t("idTokenPrefetch.clientPredictionModel")}</label><select value={form.client_prediction_model} onChange={(e) => setForm({ ...form, client_prediction_model: e.target.value as IdentityTokenPrefetchConfig["client_prediction_model"] })} className="border rounded px-3 py-2"><option value="linear">{t("idTokenPrefetch.linear")}</option><option value="exponential">{t("idTokenPrefetch.exponential")}</option><option value="ml">{t("idTokenPrefetch.mlBased")}</option></select></div>
        <div><label className="block text-sm font-medium mb-1">{t("idTokenPrefetch.offlineFallback")}</label><input type="number" value={form.offline_fallback_duration} onChange={(e) => setForm({ ...form, offline_fallback_duration: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("idTokenPrefetch.perApp")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("idTokenPrefetch.application")}</th><th scope="col">{t("idTokenPrefetch.prediction")}</th><th>{t("idTokenPrefetch.customInterval")}</th></tr></thead><tbody>{form.per_app_integration.map((a: AppIntegration, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{a.application_name}</td><td>{a.prediction_enabled ? "On" : "Off"}</td><td>{a.custom_interval}</td></tr>))}</tbody></table></div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
