"use client";
import { useEffect, useState } from "react";
import { useCorsConfig, CorsConfig, AllowedOrigin } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function CorsConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useCorsConfig();
  const [form, setForm] = useState<CorsConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  const allMethods = ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"];

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">CORS Configuration</h1>
      <p className="text-gray-600">Configure Cross-Origin Resource Sharing settings.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("corsSettings.allowedOrigins")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("corsSettings.origin")}</th><th scope="col">{t("corsSettings.tenantId")}</th></tr></thead><tbody>
          {form.allowed_origins.map((o: AllowedOrigin, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 break-all">{o.origin}</td><td className="font-mono text-xs">{o.tenant_id}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("corsSettings.credentials")}</h2>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.credential_mode} onChange={(e) => setForm({ ...form, credential_mode: e.target.checked })} className="w-4 h-4" />
          <label>{t("corsSettings.credentialMode")}</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">{t("corsSettings.allowedMethods")}</label>
          <div className="flex flex-wrap gap-4">
            {allMethods.map((m) => {
              const checked = form.allowed_methods.includes(m);
              return (
                <label key={m} className="flex items-center gap-2 text-sm">
                  <input aria-label="Checked" type="checkbox" checked={checked} readOnly className="w-4 h-4" />
                  <span>{m}</span>
                </label>
              );
            })}
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("corsSettings.allowedHeaders")}</label>
          <input aria-label="form" type="text" value={form.allowed_headers.join(", ")} readOnly className="border rounded px-3 py-2 w-full bg-gray-50" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("corsSettings.preflightCache")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("corsSettings.maxAge")}</label>
          <input aria-label="form" type="number" value={form.max_age_seconds} onChange={(e) => setForm({ ...form, max_age_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" />
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.preflight_cache_enabled} onChange={(e) => setForm({ ...form, preflight_cache_enabled: e.target.checked })} className="w-4 h-4" />
          <label>{t("corsSettings.preflightEnabled")}</label>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("corsSettings.saving") : t("corsSettings.saveChanges")}</button>
    </div>
  );
}
