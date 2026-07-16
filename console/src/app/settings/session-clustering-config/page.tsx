"use client";
import { useEffect, useState } from "react";
import { useSessionClusteringConfig, SessionClusteringConfig, RedisNode } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function SessionClusteringConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useSessionClusteringConfig();
  const [form, setForm] = useState<SessionClusteringConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Session Clustering Configuration</h1>
      <p className="text-gray-600">Configure session clustering topology, partitioning, and eviction.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Topology</h2>
        <div><label className="block text-sm font-medium mb-1">Cluster Topology</label><select aria-label="Select option" value={form.cluster_topology} onChange={(e) => setForm({ ...form, cluster_topology: e.target.value as SessionClusteringConfig["cluster_topology"] })} className="border rounded px-3 py-2"><option value="single">Single</option><option value="HA">High Availability</option><option value="cluster">Cluster</option></select></div>
        <div><label className="block text-sm font-medium mb-1">Failover Mode</label><select aria-label="form" value={form.failover_mode} onChange={(e) => setForm({ ...form, failover_mode: e.target.value as SessionClusteringConfig["failover_mode"] })} className="border rounded px-3 py-2"><option value="automatic">Automatic</option><option value="manual">Manual</option></select></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Redis Nodes</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Host</th><th scope="col">Port</th><th>Role</th></tr></thead><tbody>
          {form.redis_nodes.map((n: RedisNode, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono">{n.host}</td><td>{n.port}</td><td><span className={`px-2 py-1 rounded text-xs ${n.role === "master" ? "bg-blue-100 text-blue-700" : "bg-gray-100 text-gray-500"}`}>{n.role}</span></td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Strategy</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">Partition</label><select aria-label="Select option" value={form.partition_strategy} onChange={(e) => setForm({ ...form, partition_strategy: e.target.value as SessionClusteringConfig["partition_strategy"] })} className="border rounded px-3 py-2"><option value="by_tenant">By Tenant</option><option value="by_user">By User</option></select></div>
          <div><label className="block text-sm font-medium mb-1">Eviction</label><select aria-label="form" value={form.eviction_policy} onChange={(e) => setForm({ ...form, eviction_policy: e.target.value as SessionClusteringConfig["eviction_policy"] })} className="border rounded px-3 py-2"><option value="lru">LRU</option><option value="lfu">LFU</option><option value="ttl">TTL</option></select></div>
          <div><label className="block text-sm font-medium mb-1">Serialization</label><select aria-label="form" value={form.serialization_format} onChange={(e) => setForm({ ...form, serialization_format: e.target.value as SessionClusteringConfig["serialization_format"] })} className="border rounded px-3 py-2"><option value="json">JSON</option><option value="msgpack">MsgPack</option><option value="protobuf">Protobuf</option></select></div>
        </div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
