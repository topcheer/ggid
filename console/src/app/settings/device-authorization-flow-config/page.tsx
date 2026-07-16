"use client";
import { useEffect, useState } from "react";
import { useDeviceAuthorizationFlowConfig, DeviceAuthorizationFlowConfig, DeviceClientEntry } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function DeviceAuthorizationFlowConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useDeviceAuthorizationFlowConfig();
  const [form, setForm] = useState<DeviceAuthorizationFlowConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">{t("big1.deviceAuthorizationFlowConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("big1.deviceAuthorizationFlowConfig.error")}{error}</div>;
  if (!form) return <div className="p-8">{t("big1.deviceAuthorizationFlowConfig.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("big1.deviceAuthorizationFlowConfig.title")}</h1>
      <p className="text-gray-600">{t("big1.deviceAuthorizationFlowConfig.configureOAuth20DeviceAuthorizationGrant")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.deviceAuthorizationFlowConfig.deviceCodeSettings")}</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">{t("big1.deviceAuthorizationFlowConfig.codeLifetimeS")}</label><input aria-label="form" type="number" value={form.device_code_lifetime} onChange={(e) => setForm({ ...form, device_code_lifetime: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">{t("big1.deviceAuthorizationFlowConfig.pollingIntervalS")}</label><input aria-label="form" type="number" value={form.polling_interval} onChange={(e) => setForm({ ...form, polling_interval: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">{t("big1.deviceAuthorizationFlowConfig.userCodeFormat")}</label><select aria-label="form" value={form.user_code_format} onChange={(e) => setForm({ ...form, user_code_format: e.target.value as DeviceAuthorizationFlowConfig["user_code_format"] })} className="border rounded px-3 py-2"><option value="numeric">{t("big1.deviceAuthorizationFlowConfig.numeric")}</option><option value="alphanumeric">{t("big1.deviceAuthorizationFlowConfig.alphanumeric")}</option></select></div>
        </div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.qr_code_enabled} onChange={(e) => setForm({ ...form, qr_code_enabled: e.target.checked })} className="w-4 h-4" /><label>{t("big1.deviceAuthorizationFlowConfig.qrCodeEnabled")}</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.deviceAuthorizationFlowConfig.perClientEnabled")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("big1.deviceAuthorizationFlowConfig.client")}</th><th scope="col">{t("big1.deviceAuthorizationFlowConfig.enabled")}</th></tr></thead><tbody>
          {form.per_client_enabled.map((c: DeviceClientEntry, i: number) => (
            <tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td><td>{c.enabled ? "Yes" : "No"}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.deviceAuthorizationFlowConfig.flowStatistics")}</h2>
        <div className="grid grid-cols-3 gap-4">
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.stats.completed}</div><div className="text-xs text-gray-500">{t("big1.deviceAuthorizationFlowConfig.completed")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.stats.expired}</div><div className="text-xs text-gray-500">{t("big1.deviceAuthorizationFlowConfig.expired")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-red-600">{form.stats.rejected}</div><div className="text-xs text-gray-500">{t("big1.deviceAuthorizationFlowConfig.rejected")}</div></div>
        </div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("big1.deviceAuthorizationFlowConfig.saving") : t("big1.deviceAuthorizationFlowConfig.saveChanges")}</button>
    </div>
  );
}
