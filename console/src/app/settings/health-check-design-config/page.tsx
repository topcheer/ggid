"use client";
import { useEffect, useState } from "react";
import { useHealthCheckDesignConfig, HealthCheckDesignConfig, DependencyCheck } from "@ggid/sdk-react";

interface LocalDegradationRule {
  condition: string;
  action: string;
}

interface LocalHealthCheckDesignConfig extends HealthCheckDesignConfig {
  degradation_rules: LocalDegradationRule[];
}

export default function HealthCheckDesignConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useHealthCheckDesignConfig();
  const [form, setForm] = useState<LocalHealthCheckDesignConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config as unknown as LocalHealthCheckDesignConfig); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;
  const statusColors: Record<string, string> = { healthy: "bg-green-100 text-green-700", degraded: "bg-yellow-100 text-yellow-700", down: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Health Check Design</h1>
      <p className="text-gray-600">Configure liveness, readiness, dependency checks, and auto-healing.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Check Types & Integration</h2><div className="text-sm text-gray-600">{form.check_types.join(", ")}</div><div className="flex items-center gap-3"><input type="checkbox" checked={form.circuit_breaker_integration} onChange={(e) => setForm({ ...form, circuit_breaker_integration: e.target.checked })} className="w-4 h-4" /><label>Circuit Breaker Integration</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.auto_healing} onChange={(e) => setForm({ ...form, auto_healing: e.target.checked })} className="w-4 h-4" /><label>Auto Healing</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.lb_integration} onChange={(e) => setForm({ ...form, lb_integration: e.target.checked })} className="w-4 h-4" /><label>Load Balancer Integration</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Dependency Checks</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Dependency</th><th>Status</th><th>Latency</th></tr></thead><tbody>{form.dependency_checks.map((d: DependencyCheck, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{d.name}</td><td><span className={`px-2 py-1 rounded text-xs ${statusColors[d.status] || ""}`}>{d.status}</span></td><td>{d.latency_ms}ms</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Degradation Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Condition</th><th>Action</th></tr></thead><tbody>{form.degradation_rules.map((r, i) => (<tr key={i} className="border-b"><td className="py-2">{r.condition}</td><td className="text-xs">{r.action}</td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} aria-label="Save health check design" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
