"use client";
import { useState, useEffect, useCallback } from "react";
import { Network, Save, Plus, X } from "lucide-react";

interface RouteLimit { route: string; requests_per_min: number; burst: number; }
interface GatewayConfig { timeout_seconds: number; max_body_size_kb: number; circuit_breaker_enabled: boolean; retry_max_attempts: number; retry_backoff_ms: number; health_check_interval_seconds: number; cors_origins: string[]; route_limits: RouteLimit[]; }

export default function ApiGatewayConfigPage() {
  const [config, setConfig] = useState<GatewayConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [newOrigin, setNewOrigin] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/admin/gateway-config", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setConfig(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try { await fetch("/api/v1/admin/gateway-config", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); setSaved(true); setTimeout(() => setSaved(false), 2000); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  if (!config) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-blue-500" /> API Gateway Config</h1><p className="text-sm text-gray-500 mt-1">Configure rate limits, timeouts, CORS, and circuit breaker.</p></div>
        <div className="flex gap-2 items-center"><button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>{saved && <span className="text-sm text-green-600">Saved!</span>}</div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Timeout (seconds)</label><input type="range" min={5} max={120} value={config.timeout_seconds} onChange={(e) => setConfig({ ...config, timeout_seconds: parseInt(e.target.value) })} className="w-full mt-2" /><span className="text-sm font-bold">{config.timeout_seconds}s</span></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Max Body Size (KB)</label><input type="number" value={config.max_body_size_kb} onChange={(e) => setConfig({ ...config, max_body_size_kb: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Health Check Interval</label><input type="number" value={config.health_check_interval_seconds} onChange={(e) => setConfig({ ...config, health_check_interval_seconds: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><h3 className="text-sm font-semibold">Circuit Breaker & Retry</h3><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.circuit_breaker_enabled} onChange={(e) => setConfig({ ...config, circuit_breaker_enabled: e.target.checked })} className="rounded" /> Enable Circuit Breaker</label><div className="grid grid-cols-2 gap-3"><div><label className="text-xs font-medium text-gray-500">Max Retry Attempts</label><input type="number" value={config.retry_max_attempts} onChange={(e) => setConfig({ ...config, retry_max_attempts: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-xs font-medium text-gray-500">Retry Backoff (ms)</label><input type="number" value={config.retry_backoff_ms} onChange={(e) => setConfig({ ...config, retry_backoff_ms: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div></div></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">CORS Origins</h3><div className="flex flex-wrap gap-2 mb-2">{config.cors_origins.map((o) => (<span key={o} className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{o}<button onClick={() => setConfig({ ...config, cors_origins: config.cors_origins.filter((x) => x !== o) })}><X className="w-3 h-3" /></button></span>))}</div><div className="flex gap-2"><input type="text" value={newOrigin} onChange={(e) => setNewOrigin(e.target.value)} placeholder="https://app.example.com" className="flex-1 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /><button onClick={() => { if (newOrigin) { setConfig({ ...config, cors_origins: [...config.cors_origins, newOrigin] }); setNewOrigin(""); } }} className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-sm"><Plus className="w-4 h-4" /></button></div></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Per-Route Rate Limits</h3><div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Route</th><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Req/Min</th><th className="px-3 py-2 text-left text-xs font-medium text-gray-500">Burst</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{config.route_limits.map((r, i) => (<tr key={i}><td className="px-3 py-2 font-mono text-xs">{r.route}</td><td className="px-3 py-2"><input type="number" value={r.requests_per_min} onChange={(e) => { const next = [...config.route_limits]; next[i] = { ...r, requests_per_min: parseInt(e.target.value) || 0 }; setConfig({ ...config, route_limits: next }); }} className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td><td className="px-3 py-2"><input type="number" value={r.burst} onChange={(e) => { const next = [...config.route_limits]; next[i] = { ...r, burst: parseInt(e.target.value) || 0 }; setConfig({ ...config, route_limits: next }); }} className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td></tr>))}</tbody></table></div></div>
    </div>
  );
}
