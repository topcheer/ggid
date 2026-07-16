"use client";
import { useEffect, useState } from "react";
import { useOAuthIntrospectionCacheConfig, OAuthIntrospectionCacheConfig, PerClientTtl } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthIntrospectionCacheConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useOAuthIntrospectionCacheConfig();
  const [form, setForm] = useState<OAuthIntrospectionCacheConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Introspection Cache Configuration</h1>
      <p className="text-gray-600">Configure token introspection caching strategy.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Cache Settings</h2>
        <div><label className="block text-sm font-medium mb-1">Cache Key Strategy</label><select value={form.cache_key_strategy} onChange={(e) => setForm({ ...form, cache_key_strategy: e.target.value as OAuthIntrospectionCacheConfig["cache_key_strategy"] })} className="border rounded px-3 py-2"><option value="token_hash">Token Hash</option><option value="token_jti">Token JTI</option><option value="client_token">Client + Token</option></select></div>
        <div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">TTL (s)</label><input type="number" value={form.ttl_seconds} onChange={(e) => setForm({ ...form, ttl_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Max Entries</label><input type="number" value={form.max_entries} onChange={(e) => setForm({ ...form, max_entries: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.invalidation_on_revocation} onChange={(e) => setForm({ ...form, invalidation_on_revocation: e.target.checked })} className="w-4 h-4" /><label>Invalidation on Revocation</label></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.stampede_prevention} onChange={(e) => setForm({ ...form, stampede_prevention: e.target.checked })} className="w-4 h-4" /><label>Stampede Prevention</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Client TTL Override</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Client</th><th scope="col">TTL (s)</th></tr></thead><tbody>{form.per_client_ttl_override.map((c: PerClientTtl, i: number) => (<tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td><td>{c.ttl_seconds}</td></tr>))}</tbody></table></div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Cache Stats</h2><div className="grid grid-cols-4 gap-4"><div className="text-center"><div className="text-2xl font-bold text-green-600">{form.cache_stats.hits}</div><div className="text-xs text-gray-500">Hits</div></div><div className="text-center"><div className="text-2xl font-bold text-red-600">{form.cache_stats.misses}</div><div className="text-xs text-gray-500">Misses</div></div><div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.cache_stats.evictions}</div><div className="text-xs text-gray-500">Evictions</div></div><div className="text-center"><div className="text-2xl font-bold">{form.cache_stats.size}</div><div className="text-xs text-gray-500">Size</div></div></div></div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
