"use client";
import { useEffect, useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useOidcBackchannelLogoutConfig, OidcBackchannelLogoutConfig, BackchannelLogoutClient } from "@ggid/sdk-react";

export default function OidcBackchannelLogoutConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig, testLogout } = useOidcBackchannelLogoutConfig();
  const [form, setForm] = useState<OidcBackchannelLogoutConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const t = useTranslations();

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async (clientId: string) => { setTesting(true); await testLogout(clientId); setTesting(false); };

  if (loading && !form) return <div className="p-8">{t("oidcBackchannelLogout.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("common.error")}: {error}</div>;
  if (!form) return <div className="p-8">{t("oidcBackchannelLogout.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("oidcBackchannelLogout.title")}</h1>
      <p className="text-gray-600">{t("oidcBackchannelLogout.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("oidcBackchannelLogout.globalSettings")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("oidcBackchannelLogout.sessionLifetime")}</label>
          <input aria-label="form" type="number" value={form.session_lifetime_after_logout} onChange={(e) => setForm({ ...form, session_lifetime_after_logout: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" />
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.token_revocation_on_logout} onChange={(e) => setForm({ ...form, token_revocation_on_logout: e.target.checked })} className="w-4 h-4" />
          <label>{t("oidcBackchannelLogout.tokenRevocation")}</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("oidcBackchannelLogout.perClientEndpoints")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("oidcBackchannelLogout.client")}</th><th scope="col">{t("oidcBackchannelLogout.logoutEndpoint")}</th><th>{t("oidcBackchannelLogout.test")}</th></tr></thead><tbody>
          {form.per_client_endpoints.map((c: BackchannelLogoutClient, i: number) => (
            <tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td><td className="break-all">{c.logout_endpoint_url}</td><td><button onClick={() => handleTest(c.client_id)} disabled={testing} className="text-blue-600 hover:text-blue-800 text-xs">{t("oidcBackchannelLogout.test")}</button></td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("oidcBackchannelLogout.logoutTokenPreview")}</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.logout_token_preview || "No token preview available"}</pre>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">{t("oidcBackchannelLogout.errorHandling")}</h2>
        <div className="grid grid-cols-3 gap-4 text-sm">
          <div><span className="text-gray-500">{t("oidcBackchannelLogout.retryAttempts")}</span> <span className="font-medium">{form.error_handling.retry_attempts}</span></div>
          <div><span className="text-gray-500">{t("oidcBackchannelLogout.timeout")}</span> <span className="font-medium">{form.error_handling.timeout_seconds}</span></div>
          <div><span className="text-gray-500">{t("oidcBackchannelLogout.failed24h")}</span> <span className="font-medium text-red-600">{form.error_handling.failed_24h}</span></div>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("common.loading") : t("common.save")}</button>
    </div>
  );
}
