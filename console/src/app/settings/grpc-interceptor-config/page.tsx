"use client";
import { useEffect, useState } from "react";
import { useGrpcInterceptorConfig, GrpcInterceptorConfig, InterceptorEntry } from "@ggid/sdk-react";

export default function GrpcInterceptorConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig, testInterceptor } = useGrpcInterceptorConfig();
  const [form, setForm] = useState<GrpcInterceptorConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async (name: string) => { setTesting(true); await testInterceptor(name); setTesting(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">gRPC Interceptor Configuration</h1>
      <p className="text-gray-600">Configure interceptor chain order, toggles, and per-interceptor latency.</p>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Interceptor Chain</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Order</th><th>Name</th><th>Enabled</th><th>Latency (ms)</th><th>Action</th></tr></thead><tbody>{form.interceptor_chain.map((ic: InterceptorEntry, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono">{ic.order}</td><td className="font-medium">{ic.name}</td><td><span className={`px-2 py-1 rounded text-xs ${ic.enabled ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{ic.enabled ? "On" : "Off"}</span></td><td>{ic.latency_ms}ms</td><td><button onClick={() => handleTest(ic.name)} disabled={testing} className="text-xs text-blue-600 hover:underline">Test</button></td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
