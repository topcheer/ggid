"use client";
import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { useServiceMeshConfig, ServiceMeshConfig, TrafficPolicy } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function ServiceMeshConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useServiceMeshConfig();
  const [form, setForm] = useState<ServiceMeshConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (loading) return <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!form) return <div className="p-8 text-gray-400">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("serviceMesh.title")}</h1>
      <p className="text-gray-600">{t("serviceMesh.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Mesh Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Mesh Type</label>
          <select aria-label="form" value={form.mesh_type} onChange={(e) => setForm({ ...form, mesh_type: e.target.value as ServiceMeshConfig["mesh_type"] })} className="border rounded px-3 py-2">
            <option value="none">None</option><option value="istio">Istio</option><option value="linkerd">Linkerd</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">mTLS Mode</label>
          <select aria-label="form" value={form.mtls_mode} onChange={(e) => setForm({ ...form, mtls_mode: e.target.value as ServiceMeshConfig["mtls_mode"] })} className="border rounded px-3 py-2">
            <option value="disable">Disable</option><option value="permissive">Permissive</option><option value="strict">Strict</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Traffic Policies</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Source</th><th scope="col">Destination</th><th>Policy</th></tr></thead><tbody>
          {form.traffic_policies.map((p: TrafficPolicy, i: number) => (
            <tr key={i} className="border-b"><td className="py-2">{p.source}</td><td>{p.destination}</td><td><span className={`px-2 py-1 rounded text-xs ${p.policy === "allow" ? "bg-green-100 text-green-700" : p.policy === "deny" ? "bg-red-100 text-red-700" : "bg-yellow-100 text-yellow-700"}`}>{p.policy}</span></td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Identity Propagation</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Header Name</label>
          <input aria-label="form" type="text" value={form.identity_propagation.header_name} onChange={(e) => setForm({ ...form, identity_propagation: { ...form.identity_propagation, header_name: e.target.value } })} className="border rounded px-3 py-2 w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Format</label>
          <select aria-label="form" value={form.identity_propagation.format} onChange={(e) => setForm({ ...form, identity_propagation: { ...form.identity_propagation, format: e.target.value as "jwt" | "plain" } })} className="border rounded px-3 py-2">
            <option value="jwt">JWT</option><option value="plain">Plain</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.identity_propagation.propagate} onChange={(e) => setForm({ ...form, identity_propagation: { ...form.identity_propagation, propagate: e.target.checked } })} className="w-4 h-4" />
          <label>Propagate Identity</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Circuit Breaking</h2>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.circuit_breaking.enabled} onChange={(e) => setForm({ ...form, circuit_breaking: { ...form.circuit_breaking, enabled: e.target.checked } })} className="w-4 h-4" />
          <label>Enabled</label>
        </div>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">Max Connections</label><input aria-label="form" type="number" value={form.circuit_breaking.max_connections} onChange={(e) => setForm({ ...form, circuit_breaking: { ...form.circuit_breaking, max_connections: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Pending</label><input aria-label="form" type="number" value={form.circuit_breaking.max_pending_requests} onChange={(e) => setForm({ ...form, circuit_breaking: { ...form.circuit_breaking, max_pending_requests: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Retries</label><input aria-label="form" type="number" value={form.circuit_breaking.max_retries} onChange={(e) => setForm({ ...form, circuit_breaking: { ...form.circuit_breaking, max_retries: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Observability Export</h2>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.observability_export.enabled} onChange={(e) => setForm({ ...form, observability_export: { ...form.observability_export, enabled: e.target.checked } })} className="w-4 h-4" />
          <label>Enabled</label>
        </div>
        <div><label className="block text-sm font-medium mb-1">Endpoint</label><input aria-label="form" type="text" value={form.observability_export.endpoint} onChange={(e) => setForm({ ...form, observability_export: { ...form.observability_export, endpoint: e.target.value } })} className="border rounded px-3 py-2 w-full" /></div>
        <div><label className="block text-sm font-medium mb-1">Interval (s)</label><input aria-label="form" type="number" value={form.observability_export.interval_seconds} onChange={(e) => setForm({ ...form, observability_export: { ...form.observability_export, interval_seconds: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
