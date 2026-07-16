"use client";
import { useEffect, useState } from "react";
import { useJarConfig, JarConfig, JarPerClient } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function JarConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useJarConfig();
  const [form, setForm] = useState<JarConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => {
    if (!form) return;
    setSaving(true);
    await updateConfig(form);
    setSaving(false);
  };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">JAR Configuration</h1>
      <p className="text-gray-600">Configure JWT-Secured Authorization Request (JAR / RFC 9101).</p>

      {/* Require JAR */}
      <div className="flex items-center gap-3 bg-white rounded-lg p-4 shadow">
        <input
          type="checkbox"
          checked={form.require_jar}
          onChange={(e) => setForm({ ...form, require_jar: e.target.checked })}
          className="w-5 h-5"
        />
        <label className="font-medium">Require JAR for all clients</label>
      </div>

      {/* JAR Lifetime & Signing */}
      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">JAR Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">JAR Lifetime (seconds)</label>
          <input
            type="number"
            value={form.jar_lifetime_seconds}
            onChange={(e) => setForm({ ...form, jar_lifetime_seconds: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-48"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Signing Algorithm</label>
          <select
            value={form.signing_alg}
            onChange={(e) => setForm({ ...form, signing_alg: e.target.value as JarConfig["signing_alg"] })}
            className="border rounded px-3 py-2"
          >
            <option value="RS256">RS256</option>
            <option value="ES256">ES256</option>
            <option value="PS256">PS256</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            checked={form.encryption_optional}
            onChange={(e) => setForm({ ...form, encryption_optional: e.target.checked })}
            className="w-4 h-4"
          />
          <label>Encryption is optional</label>
        </div>
      </div>

      {/* Request Object Preview */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Request Object Preview</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.request_object_preview || "No request object available"}</pre>
      </div>

      {/* Per-Client Override Table */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Client Overrides</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th scope="col" className="py-2">Client ID</th>
              <th scope="col">Client Name</th>
              <th scope="col">Signing Alg</th>
              <th scope="col">Require JAR</th>
            </tr>
          </thead>
          <tbody>
            {form.per_client_override.map((c: JarPerClient, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2">{c.client_id}</td>
                <td>{c.client_name}</td>
                <td>{c.signing_alg}</td>
                <td>{c.require_jar ? "Yes" : "No"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Usage Stats */}
      {form.usage_stats && (
        <div className="bg-white rounded-lg p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Usage Statistics</h2>
          <div className="grid grid-cols-4 gap-4">
            <div className="text-center"><div className="text-2xl font-bold">{form.usage_stats.total_requests}</div><div className="text-xs text-gray-500">Total</div></div>
            <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.usage_stats.with_jar}</div><div className="text-xs text-gray-500">With JAR</div></div>
            <div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.usage_stats.without_jar}</div><div className="text-xs text-gray-500">Without JAR</div></div>
            <div className="text-center"><div className="text-2xl font-bold text-red-600">{form.usage_stats.rejected}</div><div className="text-xs text-gray-500">Rejected</div></div>
          </div>
        </div>
      )}

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
        {saving ? "Saving..." : "Save Changes"}
      </button>
    </div>
  );
}
