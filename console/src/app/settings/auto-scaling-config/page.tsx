"use client";
import { useEffect, useState } from "react";

interface CustomMetric {
  name: string;
  target: number;
  current: number;
}

const defaultData = {
  min_replicas: 2,
  max_replicas: 10,
  cpu_threshold_pct: 70,
  memory_threshold_pct: 80,
  scale_up_cooldown_s: 60,
  scale_down_cooldown_s: 300,
  custom_metrics: [
    { name: "requests_per_sec", target: 1000, current: 420 },
    { name: "active_sessions", target: 5000, current: 1850 },
  ] as CustomMetric[],
  predictive_scaling: true,
  hpa_yaml: `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ggid-services
spec:
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70`,
  cost_estimate_monthly: 480,
};

export default function AutoScalingConfigPage() {
  const [form, setForm] = useState(defaultData);

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Auto-Scaling Configuration</h1>
      <p className="text-gray-600">Configure horizontal pod autoscaling with custom metrics and predictive scaling.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Replica Bounds</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Min Replicas</label><input type="number" value={form.min_replicas} onChange={(e) => setForm({ ...form, min_replicas: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Replicas</label><input type="number" value={form.max_replicas} onChange={(e) => setForm({ ...form, max_replicas: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Thresholds &amp; Cooldowns</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">CPU Threshold (%)</label><input type="number" value={form.cpu_threshold_pct} onChange={(e) => setForm({ ...form, cpu_threshold_pct: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Memory Threshold (%)</label><input type="number" value={form.memory_threshold_pct} onChange={(e) => setForm({ ...form, memory_threshold_pct: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Scale-Up Cooldown (s)</label><input type="number" value={form.scale_up_cooldown_s} onChange={(e) => setForm({ ...form, scale_up_cooldown_s: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Scale-Down Cooldown (s)</label><input type="number" value={form.scale_down_cooldown_s} onChange={(e) => setForm({ ...form, scale_down_cooldown_s: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Custom Metrics</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Metric</th><th>Target</th><th>Current</th><th>Utilization</th></tr></thead>
          <tbody>
            {form.custom_metrics.map((m: CustomMetric, i: number) => {
              const pct = m.target > 0 ? Math.round((m.current / m.target) * 100) : 0;
              return (
                <tr key={i} className="border-b"><td className="py-2 font-medium">{m.name}</td><td>{m.target}</td><td>{m.current}</td><td><div className="flex items-center gap-2"><div className="w-20 bg-gray-200 rounded-full h-2"><div className={`h-2 rounded-full ${pct > 80 ? "bg-red-500" : "bg-blue-600"}`} style={{ width: `${Math.min(pct, 100)}%` }} /></div><span className="text-xs">{pct}%</span></div></td></tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Predictive Scaling</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.predictive_scaling} onChange={(e) => setForm({ ...form, predictive_scaling: e.target.checked })} className="w-4 h-4" /><label>Enable predictive scaling based on historical patterns</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-2">HPA YAML Preview</h2>
        <pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap font-mono">{form.hpa_yaml}</pre>
      </div>

      <div className="bg-blue-50 rounded-lg p-4">
        <div className="text-sm text-gray-600">Estimated Monthly Cost</div>
        <div className="text-2xl font-bold text-blue-600">${form.cost_estimate_monthly}</div>
      </div>
    </div>
  );
}
