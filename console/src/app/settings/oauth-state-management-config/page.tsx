"use client";
import { useEffect, useState } from "react";
import { useOAuthStateManagementConfig, OAuthStateManagementConfig, PerFlowEncoding } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthStateManagementConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthStateManagementConfig();
  const [form, setForm] = useState<OAuthStateManagementConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth State Parameter Management</h1>
      <p className="text-gray-600">Configure state parameter generation, binding, and validation.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">State Configuration</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">State Length (bytes)</label><input aria-label="form" type="number" value={form.state_length_bytes} onChange={(e) => setForm({ ...form, state_length_bytes: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">State TTL (seconds)</label><input aria-label="form" type="number" value={form.state_ttl_seconds} onChange={(e) => setForm({ ...form, state_ttl_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
        <div><label className="block text-sm font-medium mb-1">Binding Method</label><select aria-label="form" value={form.binding_method} onChange={(e) => setForm({ ...form, binding_method: e.target.value as OAuthStateManagementConfig["binding_method"] })} className="border rounded px-3 py-2"><option value="session">Session</option><option value="cookie">Cookie</option><option value="jwt">JWT</option></select></div>
        <div><label className="block text-sm font-medium mb-1">Validation Strictness</label><select aria-label="form" value={form.validation_strictness} onChange={(e) => setForm({ ...form, validation_strictness: e.target.value as OAuthStateManagementConfig["validation_strictness"] })} className="border rounded px-3 py-2"><option value="strict">Strict</option><option value="standard">Standard</option><option value="lenient">Lenient</option></select></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Flow Encoding</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Flow</th><th scope="col">Encoding</th></tr></thead><tbody>
          {form.per_flow_encoding.map((f: PerFlowEncoding, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{f.flow}</td><td className="font-mono text-xs">{f.encoding}</td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
