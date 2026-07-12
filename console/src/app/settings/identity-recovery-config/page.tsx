"use client";
import { useEffect, useState } from "react";
import { useIdentityRecoveryConfig, IdentityRecoveryConfig } from "@ggid/sdk-react";

export default function IdentityRecoveryConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useIdentityRecoveryConfig();
  const [form, setForm] = useState<IdentityRecoveryConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Identity Recovery Configuration</h1>
      <p className="text-gray-600">Configure account takeover response, mass reset, and forensics collection.</p>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Takeover Response Checklist</h2><div className="space-y-1">{form.takeover_response_checklist.map((item: string, i: number) => (<div key={i} className="flex items-center gap-3 border-b py-1"><input type="checkbox" readOnly className="w-4 h-4" /><span className="text-sm">{item}</span></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Recovery Actions</h2><div><label className="block text-sm font-medium mb-1">Session Invalidation Scope</label><select value={form.session_invalidation_scope} onChange={(e) => setForm({ ...form, session_invalidation_scope: e.target.value as IdentityRecoveryConfig["session_invalidation_scope"] })} className="border rounded px-3 py-2"><option value="affected">Affected Users</option><option value="tenant">Entire Tenant</option><option value="global">Global</option></select></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.forensics_auto_collect} onChange={(e) => setForm({ ...form, forensics_auto_collect: e.target.checked })} className="w-4 h-4" /><label>Forensics Auto-Collection</label></div><div><label className="block text-sm font-medium mb-1">Notification Channels</label><div className="text-sm text-gray-600">{form.notification_channels.join(", ")}</div></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Mass Reset Template</h2><pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.mass_reset_template}</pre></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">User Communication Template</h2><pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.user_communication_template}</pre></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
