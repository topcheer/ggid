"use client";
import { useState, useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface GraphNode {
  id: string;
  label: string;
  x: number;
  y: number;
  health: "healthy" | "degraded" | "down";
  slo: number;
  error_budget: number;
  dependencies: string[];
}

const nodes: GraphNode[] = [
  { id: "gateway", label: "Gateway", x: 400, y: 50, health: "healthy", slo: 99.9, error_budget: 87, dependencies: ["auth", "identity", "policy"] },
  { id: "auth", label: "Auth", x: 200, y: 180, health: "healthy", slo: 99.9, error_budget: 92, dependencies: ["identity"] },
  { id: "identity", label: "Identity", x: 400, y: 180, health: "degraded", slo: 99.5, error_budget: 45, dependencies: ["org"] },
  { id: "policy", label: "Policy", x: 600, y: 180, health: "healthy", slo: 99.9, error_budget: 78, dependencies: [] },
  { id: "org", label: "Org", x: 300, y: 310, health: "healthy", slo: 99.9, error_budget: 90, dependencies: ["audit"] },
  { id: "audit", label: "Audit", x: 500, y: 310, health: "healthy", slo: 99.5, error_budget: 60, dependencies: [] },
  { id: "oauth", label: "OAuth", x: 100, y: 310, health: "down", slo: 99.9, error_budget: 0, dependencies: ["identity"] },
];

const edges = [
  { from: "gateway", to: "auth", protocol: "gRPC", calls_per_sec: 250, avg_latency_ms: 12 },
  { from: "gateway", to: "identity", protocol: "gRPC", calls_per_sec: 180, avg_latency_ms: 8 },
  { from: "gateway", to: "policy", protocol: "gRPC", calls_per_sec: 320, avg_latency_ms: 5 },
  { from: "auth", to: "identity", protocol: "gRPC", calls_per_sec: 150, avg_latency_ms: 9 },
  { from: "identity", to: "org", protocol: "gRPC", calls_per_sec: 90, avg_latency_ms: 15 },
  { from: "org", to: "audit", protocol: "NATS", calls_per_sec: 200, avg_latency_ms: 2 },
  { from: "oauth", to: "identity", protocol: "gRPC", calls_per_sec: 0, avg_latency_ms: 0 },
];

export default function ServiceDependencyGraphPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/healthz", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const t = useTranslations();
  const [selected, setSelected] = useState<GraphNode | null>(null);
  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  const healthColors: Record<string, string> = { healthy: "#10b981", degraded: "#f59e0b", down: "#ef4444" };
  const nodeMap = new Map(nodes.map((n: any) => [n.id, n]));

  return (
    <div className="p-8 space-y-6 max-w-5xl">
      <h1 className="text-2xl font-bold">{t("serviceDependencyGraph.title")}</h1>
      <p className="text-gray-600">{t("serviceDependencyGraph.subtitle")}</p>

      <div className="bg-yellow-50 border border-yellow-300 rounded-lg p-3">
        <span className="text-sm font-medium text-yellow-800">{"Warning: Circular dependency detected (identity -> org -> audit -> identity)"}</span>
      </div>

      <div className="flex gap-6">
        <div className="flex-1 bg-white rounded-lg p-6 shadow">
          <svg viewBox="0 0 700 400" className="w-full h-auto">
            {edges.map((e: any, i: number) => {
              const from = nodeMap.get(e.from); const to = nodeMap.get(e.to);
              if (!from || !to) return null;
              const midX = (from.x + to.x) / 2; const midY = (from.y + to.y) / 2;
              return (
                <g key={i}>
                  <line x1={from.x} y1={from.y} x2={to.x} y2={to.y} stroke="#cbd5e1" strokeWidth="2" markerEnd="url(#arrow)" />
                  <text x={midX} y={midY - 5} textAnchor="middle" className="text-[8px] fill-gray-500">{e.protocol} | {e.calls_per_sec}/s | {e.avg_latency_ms}ms</text>
                </g>
              );
            })}
            {nodes.map((n: any) => (
              <g key={n.id} onClick={() => setSelected(n)} className="cursor-pointer">
                <circle cx={n.x} cy={n.y} r="28" fill={healthColors[n.health]} fillOpacity="0.15" stroke={healthColors[n.health]} strokeWidth="2" />
                <text x={n.x} y={n.y + 4} textAnchor="middle" className="text-[10px] font-medium fill-gray-700">{n.label}</text>
              </g>
            ))}
            <defs><marker id="arrow" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0,0 L6,3 L0,6 Z" fill="#cbd5e1" /></marker></defs>
          </svg>
        </div>

        {selected && (
          <div className="w-64 bg-white rounded-lg p-6 shadow space-y-3">
            <h2 className="text-lg font-semibold">{selected.label}</h2>
            <div><span className="text-sm text-gray-500">{t("serviceDependencyGraph.health")} </span><span className={`px-2 py-0.5 rounded text-xs ${selected.health === "healthy" ? "bg-green-100 text-green-700" : selected.health === "degraded" ? "bg-yellow-100 text-yellow-700" : "bg-red-100 text-red-700"}`}>{selected.health}</span></div>
            <div className="text-sm"><span className="text-gray-500">{t("serviceDependencyGraph.slo")} </span><span className="font-medium">{selected.slo}%</span></div>
            <div className="text-sm"><span className="text-gray-500">{t("serviceDependencyGraph.errorBudget")} </span><span className="font-medium">{selected.error_budget}%</span></div>
            <div className="text-sm"><span className="text-gray-500">Dependencies: </span>{selected.dependencies.length > 0 ? selected.dependencies.join(", ") : "none"}</div>
          </div>
        )}
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Edge Details</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">From</th><th scope="col">To</th><th>Protocol</th><th>Calls/s</th><th>Avg Latency</th></tr></thead>
          <tbody>
            {edges.map((e: any, i: number) => (
              <tr key={i} className="border-b"><td className="py-2 font-medium">{e.from}</td><td>{e.to}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{e.protocol}</span></td><td>{e.calls_per_sec}</td><td>{e.avg_latency_ms}ms</td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
