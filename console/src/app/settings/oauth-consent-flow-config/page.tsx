"use client";
import { useEffect, useState } from "react";
import { useOauthConsentFlowConfig, OauthConsentFlowConfig, ScopeDescription, PreApprovedApp } from "@ggid/sdk-react";

export default function OauthConsentFlowConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOauthConsentFlowConfig();
  const [form, setForm] = useState<OauthConsentFlowConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Consent Flow Configuration</h1>
      <p className="text-gray-600">Configure consent screen, scope descriptions, and pre-approved apps.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Consent Screen Preview</h2>
        <div className="flex items-center gap-4">
          {form.consent_screen.logo_url && <img src={form.consent_screen.logo_url} alt="Logo" className="w-16 h-16 rounded" />}
          <div className="text-sm">
            <div><span className="text-gray-500">Privacy URL:</span> {form.consent_screen.privacy_url}</div>
            <div><span className="text-gray-500">Terms URL:</span> {form.consent_screen.tos_url}</div>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Scope Descriptions</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Scope</th><th>Description</th></tr></thead><tbody>
          {form.per_scope_description.map((s: ScopeDescription, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono">{s.scope}</td><td>{s.description}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Consent Behavior</h2>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.show_skip_consent} onChange={(e) => setForm({ ...form, show_skip_consent: e.target.checked })} className="w-4 h-4" />
          <label>Show Skip Consent Option</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Remember Duration: {form.remember_duration_days} days</label>
          <input type="range" min={0} max={365} value={form.remember_duration_days} onChange={(e) => setForm({ ...form, remember_duration_days: parseInt(e.target.value) })} className="w-full" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Pre-Approved Apps</h2>
        <div className="space-y-2">
          {form.pre_approved_apps.map((a: PreApprovedApp, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{a.client_name}</span><span className="ml-2 text-xs text-gray-400">{a.client_id}</span></div>
              <span className="text-sm text-gray-500">{a.scopes.join(", ")}</span>
            </div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
