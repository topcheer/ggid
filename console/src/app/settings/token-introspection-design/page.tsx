"use client";
import { useEffect, useState } from "react";
import { useTokenIntrospectionDesign, TokenIntrospectionDesign, ResourceServerAuth } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function TokenIntrospectionDesignPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useTokenIntrospectionDesign();
  const [form, setForm] = useState<TokenIntrospectionDesign | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Token Introspection Design (RFC 7662)</h1>
      <p className="text-gray-600">Configure introspection endpoint caching, filtering, and rate limiting.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Caching Strategy</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.caching.enabled} onChange={(e) => setForm({ ...form, caching: { ...form.caching, enabled: e.target.checked } })} className="w-4 h-4" /><label>Enabled</label></div>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">TTL (seconds)</label><input aria-label="form" type="number" value={form.caching.ttl_seconds} onChange={(e) => setForm({ ...form, caching: { ...form.caching, ttl_seconds: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Entries</label><input aria-label="form" type="number" value={form.caching.max_entries} onChange={(e) => setForm({ ...form, caching: { ...form.caching, max_entries: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Options</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.scope_filtering} onChange={(e) => setForm({ ...form, scope_filtering: e.target.checked })} className="w-4 h-4" /><label>Scope Filtering</label></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.privacy_mode} onChange={(e) => setForm({ ...form, privacy_mode: e.target.checked })} className="w-4 h-4" /><label>Privacy Mode (minimal claims in response)</label></div>
        <div><label className="block text-sm font-medium mb-1">Rate Limit Per Client (req/min)</label><input aria-label="form" type="number" value={form.rate_limit_per_client} onChange={(e) => setForm({ ...form, rate_limit_per_client: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per Resource Server Auth</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Resource Server</th><th scope="col">Auth Required</th><th>Scope</th></tr></thead><tbody>
          {form.per_resource_server_auth.map((r: ResourceServerAuth, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.resource_server}</td><td>{r.auth_required ? "Yes" : "No"}</td><td className="font-mono text-xs">{r.scope}</td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
