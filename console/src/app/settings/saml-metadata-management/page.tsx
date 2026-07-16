"use client";
import { useEffect, useState } from "react";
import { useSamlMetadataManagement, SamlMetadataManagement, IdpMetadata } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function SamlMetadataManagementPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useSamlMetadataManagement();
  const [form, setForm] = useState<SamlMetadataManagement | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">SAML Metadata Management</h1>
      <p className="text-gray-600">Manage SP and IdP metadata, refresh schedules, and federation aggregation.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">SP Metadata Preview</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.sp_metadata_preview || "No metadata available"}</pre>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">IdP Metadata</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Entity ID</th><th scope="col">URL</th><th>Last Refresh</th><th>Sig Valid</th></tr></thead><tbody>
          {form.idp_metadata_list.map((m: IdpMetadata, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono text-xs">{m.entity_id}</td><td className="break-all text-xs">{m.url}</td><td className="text-xs text-gray-500">{m.last_refresh}</td><td><span className={`px-2 py-1 rounded text-xs ${m.signature_valid ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{m.signature_valid ? "Yes" : "No"}</span></td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Settings</h2>
        <div><label className="block text-sm font-medium mb-1">Refresh Schedule (cron)</label><input aria-label="form" type="text" value={form.refresh_schedule_cron} onChange={(e) => setForm({ ...form, refresh_schedule_cron: e.target.value })} className="border rounded px-3 py-2 w-full font-mono text-sm" /></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.federation_aggregation} onChange={(e) => setForm({ ...form, federation_aggregation: e.target.checked })} className="w-4 h-4" /><label>Federation Aggregation</label></div>
        <div className="text-sm text-gray-500">Entity Categories: {form.entity_categories.join(", ")}</div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
