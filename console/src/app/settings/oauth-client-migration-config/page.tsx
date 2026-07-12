"use client";
import { useEffect, useState } from "react";
import { useOAuthClientMigrationConfig, OAuthClientMigrationConfig, MappingPreview, CutoverPhase } from "@ggid/sdk-react";

export default function OAuthClientMigrationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthClientMigrationConfig();
  const [form, setForm] = useState<OAuthClientMigrationConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Client Migration Configuration</h1>
      <p className="text-gray-600">Migrate OAuth clients from external providers to this system.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Source System</h2>
        <select value={form.source_system} onChange={(e) => setForm({ ...form, source_system: e.target.value as OAuthClientMigrationConfig["source_system"] })} className="border rounded px-3 py-2">
          <option value="Auth0">Auth0</option><option value="Okta">Okta</option><option value="Keycloak">Keycloak</option><option value="Ping">Ping</option>
        </select>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Migration Scope</h2>
        {(["clients", "users", "policies", "custom_claims"] as const).map((k) => (
          <div key={k} className="flex items-center gap-3">
            <input type="checkbox" checked={form.migration_scope[k]} onChange={(e) => setForm({ ...form, migration_scope: { ...form.migration_scope, [k]: e.target.checked } })} className="w-4 h-4" />
            <label className="capitalize">{k.replace("_", " ")}</label>
          </div>
        ))}
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Field Mapping Preview</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Source Field</th><th>Target Field</th><th>Transform</th></tr></thead><tbody>
          {form.mapping_preview.map((m: MappingPreview, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono">{m.source_field}</td><td className="font-mono">{m.target_field}</td><td className="text-xs">{m.transform}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Validation Report</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="text-center"><div className="text-2xl font-bold">{form.validation_report.total_clients}</div><div className="text-xs text-gray-500">Total Clients</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.validation_report.mapped}</div><div className="text-xs text-gray-500">Mapped</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.validation_report.warnings}</div><div className="text-xs text-gray-500">Warnings</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-red-600">{form.validation_report.errors}</div><div className="text-xs text-gray-500">Errors</div></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Phased Cutover Timeline</h2>
        <div className="space-y-2">
          {form.phased_cutover_timeline.map((p: CutoverPhase, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{p.phase}</span><span className="ml-2 text-sm text-gray-500">{p.description}</span></div>
              <span className="text-sm text-gray-400">{p.duration_days} days</span>
            </div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
