"use client";
import { useEffect, useState } from "react";
import { useOAuthErrorCatalogConfig, OAuthErrorCatalogConfig, ErrorCodeEntry, CustomLocaleMessage } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthErrorCatalogConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthErrorCatalogConfig();
  const [form, setForm] = useState<OAuthErrorCatalogConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;
  const sevColors: Record<string, string> = { info: "bg-blue-100 text-blue-700", warn: "bg-yellow-100 text-yellow-700", error: "bg-orange-100 text-orange-700", critical: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Error Catalog Configuration</h1>
      <p className="text-gray-600">Configure OAuth error codes, messages, and severity.</p>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Error Codes</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Code</th><th scope="col">HTTP</th><th>Severity</th><th>User Message</th><th>Retry</th><th>Docs</th></tr></thead><tbody>{form.error_codes.map((e: ErrorCodeEntry, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono">{e.code}</td><td>{e.http_status}</td><td><span className={`px-2 py-1 rounded text-xs ${sevColors[e.severity] || ""}`}>{e.severity}</span></td><td className="text-xs">{e.user_message}</td><td className="text-xs">{e.retry_guidance}</td><td className="text-xs text-blue-600 break-all max-w-[120px] truncate">{e.developer_doc_url}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Custom Messages Per Locale</h2><div className="space-y-2">{form.custom_error_messages_per_locale.map((l: CustomLocaleMessage, i: number) => (<div key={i} className="border-b py-2"><span className="font-medium">{l.locale}</span><div className="text-xs text-gray-500">{Object.keys(l.messages).length} custom messages</div></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.troubleshooting_enabled} onChange={(e) => setForm({ ...form, troubleshooting_enabled: e.target.checked })} className="w-4 h-4" /><label>Enable Troubleshooting Mode</label></div></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
