"use client";
import { useState, useEffect, useCallback } from "react";
import { Database, Zap, Trash2, Activity } from "lucide-react";

interface CacheInstance { id: string; name: string; type: "Redis" | "Memory"; status: "healthy" | "degraded" | "down"; hit_rate_pct: number; memory_used_mb: number; memory_max_mb: number; keys: number; evictions_per_min: number; latency_ms: number; top_keys: { key: string; hits: number; ttl: number }[]; }

export default function CacheHealthPage() {
  const [instances, setInstances] = useState<CacheInstance[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/admin/cache-health", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setInstances(d.instances || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const statusColors: Record<string, string> = { healthy: "text-green-600", degraded: "text-yellow-600", down: "text-red-600" };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Database className="w-6 h-6 text-red-500" /> Cache Health</h1><p className="text-sm text-gray-500 mt-1">Monitor cache instances, hit rates, and memory usage.</p></div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {instances.map((inst) => {
          const memPct = (inst.memory_used_mb / inst.memory_max_mb) * 100;
          return (<div key={inst.id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><div className="flex items-center justify-between"><div><span className="font-semibold">{inst.name}</span><p className="text-xs text-gray-400 font-mono">{inst.type}</p></div><span className={"text-sm font-medium " + statusColors[inst.status]}>{inst.status}</span></div>
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="flex items-center gap-1"><Activity className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Hit Rate</span><span className={"font-bold ml-auto " + (inst.hit_rate_pct > 80 ? "text-green-600" : "text-yellow-600")}>{inst.hit_rate_pct}%</span></div>
              <div className="flex items-center gap-1"><Zap className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Latency</span><span className="font-bold ml-auto">{inst.latency_ms}ms</span></div>
              <div><span className="text-gray-500">Keys:</span> <span className="font-bold">{inst.keys.toLocaleString()}</span></div>
              <div><span className="text-gray-500">Evictions:</span> <span className={"font-bold " + (inst.evictions_per_min > 100 ? "text-red-600" : "")}>{inst.evictions_per_min}/min</span></div>
            </div>
            <div><div className="flex items-center justify-between text-xs mb-1"><span className="text-gray-500">Memory</span><span className="font-medium">{inst.memory_used_mb} / {inst.memory_max_mb} MB ({memPct.toFixed(0)}%)</span></div><div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-3"><div className={"h-full rounded-full " + (memPct > 90 ? "bg-red-500" : memPct > 70 ? "bg-yellow-500" : "bg-green-500")} style={{ width: memPct + "%" }} /></div></div>
            {inst.top_keys.length > 0 && (<div><h4 className="text-xs font-semibold text-gray-500 mb-1">Top Keys</h4><div className="space-y-0.5">{inst.top_keys.slice(0, 5).map((k) => (<div key={k.key} className="flex items-center gap-2 text-xs"><span className="font-mono text-gray-500 flex-1 truncate">{k.key}</span><span className="text-gray-400">{k.hits} hits</span><span className="text-gray-400">TTL {k.ttl}s</span></div>))}</div></div>)}
          </div>);
        })}
        {instances.length === 0 && !loading && <div className="col-span-full text-center text-gray-500 py-8">No cache instances.</div>}
      </div>
    </div>
  );
}
