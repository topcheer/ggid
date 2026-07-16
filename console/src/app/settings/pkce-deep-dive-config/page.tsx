"use client";
import { useEffect, useState } from "react";
import { usePkceDeepDiveConfig, PkceDeepDiveConfig, PkceClientEntry } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function PkceDeepDiveConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = usePkceDeepDiveConfig();
  const [form, setForm] = useState<PkceDeepDiveConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">PKCE Deep Dive Configuration</h1>
      <p className="text-gray-600">Configure Proof Key for Code Exchange (RFC 7636) enforcement.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Global Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Code Challenge Method</label>
          <select aria-label="form" value={form.code_challenge_method} onChange={(e) => setForm({ ...form, code_challenge_method: e.target.value as PkceDeepDiveConfig["code_challenge_method"] })} className="border rounded px-3 py-2">
            <option value="S256">S256</option><option value="plain">Plain</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.migrate_non_pkce_clients} onChange={(e) => setForm({ ...form, migrate_non_pkce_clients: e.target.checked })} className="w-4 h-4" />
          <label>Migrate Non-PKCE Clients</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-2">Compliance</h2>
        <div className="flex items-center gap-4">
          <div className="text-3xl font-bold text-blue-600">{form.compliance_pct}%</div>
          <div className="flex-1 bg-gray-200 rounded-full h-4"><div className="bg-blue-600 h-4 rounded-full" style={{ width: `${form.compliance_pct}%` }} /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Client Enforcement</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Client</th><th scope="col">Required</th><th>Method</th></tr></thead><tbody>
          {form.per_client_enforcement.map((c: PkceClientEntry, i: number) => (
            <tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td><td>{c.required ? "Yes" : "No"}</td><td>{c.method}</td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
