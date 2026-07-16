"use client";
import { useEffect, useState } from "react";
import { useWebauthnServerConfig, WebauthnServerConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function WebauthnServerConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useWebauthnServerConfig();
  const [form, setForm] = useState<WebauthnServerConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">WebAuthn Server Configuration</h1>
      <p className="text-gray-600">Configure WebAuthn / Passkey server settings.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Ceremony Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Ceremony Timeout (seconds)</label>
          <input aria-label="form" type="number" value={form.ceremony_timeout}
            onChange={(e) => setForm({ ...form, ceremony_timeout: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Attestation Trust Path</label>
          <select aria-label="form" value={form.attestation_trust_path}
            onChange={(e) => setForm({ ...form, attestation_trust_path: e.target.value as WebauthnServerConfig["attestation_trust_path"] })}
            className="border rounded px-3 py-2">
            <option value="none">None</option>
            <option value="indirect">Indirect</option>
            <option value="direct">Direct</option>
            <option value="enterprise">Enterprise</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Relying Party Entity</h2>
        <div>
          <label className="block text-sm font-medium mb-1">RP ID</label>
          <input aria-label="form" type="text" value={form.rp_entity.id}
            onChange={(e) => setForm({ ...form, rp_entity: { ...form.rp_entity, id: e.target.value } })}
            className="border rounded px-3 py-2 w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">RP Name</label>
          <input aria-label="form" type="text" value={form.rp_entity.name}
            onChange={(e) => setForm({ ...form, rp_entity: { ...form.rp_entity, name: e.target.value } })}
            className="border rounded px-3 py-2 w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Origins</label>
          <input aria-label="form" type="text" value={form.rp_entity.origins.join(", ")} readOnly className="border rounded px-3 py-2 w-full bg-gray-50" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Credential & Security</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Credential Storage Policy</label>
          <select aria-label="form" value={form.credential_storage_policy}
            onChange={(e) => setForm({ ...form, credential_storage_policy: e.target.value as WebauthnServerConfig["credential_storage_policy"] })}
            className="border rounded px-3 py-2">
            <option value="database">Database</option>
            <option value="memory">Memory</option>
            <option value="hybrid">Hybrid</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Counter Enforcement</label>
          <select aria-label="form" value={form.counter_enforcement}
            onChange={(e) => setForm({ ...form, counter_enforcement: e.target.value as WebauthnServerConfig["counter_enforcement"] })}
            className="border rounded px-3 py-2">
            <option value="strict">Strict</option>
            <option value="report">Report</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.uv_preferred}
            onChange={(e) => setForm({ ...form, uv_preferred: e.target.checked })}
            className="w-4 h-4" />
          <label>User Verification Preferred</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">AAGUID Allowlist</h2>
        <div className="space-y-1">
          {form.aaguid_allowlist.map((aaguid: string, i: number) => (
            <div key={i} className="font-mono text-sm border-b py-1">{aaguid}</div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
