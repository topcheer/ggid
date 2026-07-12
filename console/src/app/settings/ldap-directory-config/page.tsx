"use client";
import { useEffect, useState } from "react";
import { useLdapDirectoryConfig, LdapDirectoryConfig, DirectoryFederation } from "@ggid/sdk-react";

export default function LdapDirectoryConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useLdapDirectoryConfig();
  const [form, setForm] = useState<LdapDirectoryConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">LDAP Directory Configuration</h1>
      <p className="text-gray-600">Configure LDAP connection pooling, search optimization, and federation.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Connection Pool</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">Min Connections</label><input type="number" value={form.connection_pool.min} onChange={(e) => setForm({ ...form, connection_pool: { ...form.connection_pool, min: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Connections</label><input type="number" value={form.connection_pool.max} onChange={(e) => setForm({ ...form, connection_pool: { ...form.connection_pool, max: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Idle Timeout (s)</label><input type="number" value={form.connection_pool.idle_timeout_seconds} onChange={(e) => setForm({ ...form, connection_pool: { ...form.connection_pool, idle_timeout_seconds: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Search Optimization</h2>
        <div className="text-sm text-gray-500">Indexed Attributes: {form.search_optimization.indexed_attributes.join(", ")}</div>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.search_optimization.query_cache_enabled} onChange={(e) => setForm({ ...form, search_optimization: { ...form.search_optimization, query_cache_enabled: e.target.checked } })} className="w-4 h-4" />
          <label>Query Cache Enabled</label>
        </div>
        <div><label className="block text-sm font-medium mb-1">Query Cache TTL (s)</label><input type="number" value={form.search_optimization.query_cache_ttl} onChange={(e) => setForm({ ...form, search_optimization: { ...form.search_optimization, query_cache_ttl: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Group Membership Resolution</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Nested Depth</label><input type="number" value={form.group_membership_resolution.nested_depth} onChange={(e) => setForm({ ...form, group_membership_resolution: { ...form.group_membership_resolution, nested_depth: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Cache TTL (s)</label><input type="number" value={form.group_membership_resolution.cache_ttl} onChange={(e) => setForm({ ...form, group_membership_resolution: { ...form.group_membership_resolution, cache_ttl: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Multi-Directory Federation</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Base DN</th><th>Bind DN</th><th>Priority</th></tr></thead><tbody>
          {form.multi_directory_federation.map((d: DirectoryFederation, i: number) => (
            <tr key={i} className="border-b"><td className="py-2">{d.name}</td><td className="font-mono text-xs">{d.base_dn}</td><td className="font-mono text-xs">{d.bind_dn}</td><td>{d.priority}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Sync Tuning</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Batch Size</label><input type="number" value={form.sync_tuning.batch_size} onChange={(e) => setForm({ ...form, sync_tuning: { ...form.sync_tuning, batch_size: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Interval (s)</label><input type="number" value={form.sync_tuning.interval_seconds} onChange={(e) => setForm({ ...form, sync_tuning: { ...form.sync_tuning, interval_seconds: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
