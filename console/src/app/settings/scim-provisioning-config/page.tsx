"use client";
import { useEffect, useState } from "react";
import { useScimProvisioningConfig, ScimProvisioningConfig, ScimMappingRule } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function ScimProvisioningConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig, testConnection } = useScimProvisioningConfig();
  const [form, setForm] = useState<ScimProvisioningConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async () => { setTesting(true); await testConnection(); setTesting(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">SCIM Provisioning Configuration</h1>
      <p className="text-gray-600">Configure SCIM 2.0 provisioning endpoint, mappings, and sync.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Endpoint</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Endpoint URL</label>
          <input aria-label="form" type="text" value={form.endpoint_url}
            onChange={(e) => setForm({ ...form, endpoint_url: e.target.value })}
            className="border rounded px-3 py-2 w-full" />
        </div>
        <div className="flex items-center gap-4">
          <span className={`px-3 py-1 rounded text-sm ${form.test_connection_status === "connected" ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`} >{form.test_connection_status}</span>
          <button onClick={handleTest} disabled={testing} className="px-4 py-1 border rounded text-sm hover:bg-gray-50">{testing ? "Testing..." : "Test Connection"}</button>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Mapping Rules</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Source Field</th><th scope="col">Target Field</th><th>Required</th></tr></thead>
          <tbody>
            {form.mapping_rules.map((r: ScimMappingRule, i: number) => (
              <tr key={i} className="border-b"><td className="py-2">{r.source_field}</td><td>{r.target_field}</td><td>{r.required ? "Yes" : "No"}</td></tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Provisioning Triggers</h2>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.provisioning_triggers.create}
            onChange={(e) => setForm({ ...form, provisioning_triggers: { ...form.provisioning_triggers, create: e.target.checked } })}
            className="w-4 h-4" />
          <label>Create</label>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.provisioning_triggers.update}
            onChange={(e) => setForm({ ...form, provisioning_triggers: { ...form.provisioning_triggers, update: e.target.checked } })}
            className="w-4 h-4" />
          <label>Update</label>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.provisioning_triggers.deactivate}
            onChange={(e) => setForm({ ...form, provisioning_triggers: { ...form.provisioning_triggers, deactivate: e.target.checked } })}
            className="w-4 h-4" />
          <label>Deactivate</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Sync Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Sync Direction</label>
          <select aria-label="form" value={form.sync_direction}
            onChange={(e) => setForm({ ...form, sync_direction: e.target.value as ScimProvisioningConfig["sync_direction"] })}
            className="border rounded px-3 py-2">
            <option value="inbound">Inbound</option>
            <option value="outbound">Outbound</option>
            <option value="bidirectional">Bidirectional</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.deprovision_on_disable}
            onChange={(e) => setForm({ ...form, deprovision_on_disable: e.target.checked })}
            className="w-4 h-4" />
          <label>Deprovision on Disable</label>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
