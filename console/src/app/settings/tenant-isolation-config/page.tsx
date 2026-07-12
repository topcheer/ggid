"use client";
import { useEffect, useState } from "react";
import { useTenantIsolationConfig, TenantIsolationConfig, IsolationTestResult } from "@ggid/sdk-react";

export default function TenantIsolationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig, testIsolation } = useTenantIsolationConfig();
  const [form, setForm] = useState<TenantIsolationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async () => { setTesting(true); await testIsolation(); setTesting(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Tenant Isolation Configuration</h1>
      <p className="text-gray-600">Configure multi-tenant isolation for database, cache, and storage.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Isolation Settings</h2><div><label className="block text-sm font-medium mb-1">Isolation Mode</label><select value={form.isolation_mode} onChange={(e) => setForm({ ...form, isolation_mode: e.target.value as TenantIsolationConfig["isolation_mode"] })} className="border rounded px-3 py-2"><option value="rls">Row-Level Security</option><option value="schema">Schema per Tenant</option><option value="hybrid">Hybrid</option></select></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.per_tenant_redis_namespace} onChange={(e) => setForm({ ...form, per_tenant_redis_namespace: e.target.checked })} className="w-4 h-4" /><label>Per-Tenant Redis Namespace</label></div><div><label className="block text-sm font-medium mb-1">NATS Subject Prefix</label><input type="text" value={form.nats_subject_prefix} onChange={(e) => setForm({ ...form, nats_subject_prefix: e.target.value })} className="border rounded px-3 py-2 w-full" /></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.file_storage_isolation} onChange={(e) => setForm({ ...form, file_storage_isolation: e.target.checked })} className="w-4 h-4" /><label>File Storage Isolation</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Isolation Test Results</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Test</th><th>Passed</th><th>Detail</th></tr></thead><tbody>{form.isolation_test_results.map((r: IsolationTestResult, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{r.test}</td><td><span className={`px-2 py-1 rounded text-xs ${r.passed ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{r.passed ? "Pass" : "Fail"}</span></td><td className="text-xs">{r.detail}</td></tr>))}</tbody></table></div>
      <div className="flex gap-4"><button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button><button onClick={handleTest} disabled={testing} className="px-6 py-2 border rounded-lg hover:bg-gray-50 disabled:opacity-50">{testing ? "Testing..." : "Test Isolation"}</button></div>
    </div>
  );
}
