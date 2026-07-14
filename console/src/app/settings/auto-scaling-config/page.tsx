"use client";
import { useEffect, useState } from "react";
import { Save, Loader2, AlertCircle, Check } from "lucide-react";

interface CustomMetric {
  name: string;
  target: number;
  current: number;
}

interface AutoScalingData {
  min_replicas: number;
  max_replicas: number;
  cpu_threshold_pct: number;
  memory_threshold_pct: number;
  scale_up_cooldown_s: number;
  scale_down_cooldown_s: number;
  custom_metrics: CustomMetric[];
  predictive_scaling: boolean;
  hpa_yaml: string;
  cost_estimate_monthly: number;
}

const defaultData: AutoScalingData = {
  min_replicas: 2,
  max_replicas: 10,
  cpu_threshold_pct: 70,
  memory_threshold_pct: 80,
  scale_up_cooldown_s: 60,
  scale_down_cooldown_s: 300,
  custom_metrics: [
    { name: "requests_per_sec", target: 1000, current: 420 },
    { name: "active_sessions", target: 5000, current: 1850 },
  ],
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
  const [form, setForm] = useState<AutoScalingData>(defaultData);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    setLoading(true); setError("");
    fetch("/api/v1/settings/auto-scaling", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } })
      .then(async (res) => { if (res.ok) { const data = await res.json(); if (data) setForm((prev) => ({ ...prev, ...data })); } })
      .catch(() => { /* use defaults */ })
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true); setError(""); setMsg("");
    try {
      const res = await fetch("/api/v1/settings/auto-scaling", {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(form),
      });
      if (!res.ok) throw new Error(`Save failed: HTTP ${res.status}`);
      setMsg("Auto-scaling configuration saved");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save configuration");
    } finally { setSaving(false); setTimeout(() => setMsg(""), 4000); }
  };

  const updateField = <K extends keyof AutoScalingData>(key: K, value: AutoScalingData[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const inputCls = "border rounded px-3 py-2 w-full dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100";

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">Auto-Scaling Configuration</h1>
          <p className="text-gray-600 dark:text-gray-400">Configure horizontal pod autoscaling with custom metrics and predictive scaling.</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving || loading}
          aria-label="Save auto-scaling configuration"
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" /> {error}
        </div>
      )}
      {msg && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400 flex items-center gap-2">
          <Check className="h-4 w-4" /> {msg}
        </div>
      )}
      {loading && <div className="flex items-center gap-2 text-sm text-gray-500"><Loader2 className="h-4 w-4 animate-spin" /> Loading configuration...</div>}

      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold dark:text-gray-100">Replica Bounds</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1 dark:text-gray-300">Min Replicas</label>
            <input aria-label="Minimum replicas" type="number" value={form.min_replicas} onChange={(e) => updateField("min_replicas", parseInt(e.target.value) || 0)} className={inputCls} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1 dark:text-gray-300">Max Replicas</label>
            <input aria-label="Maximum replicas" type="number" value={form.max_replicas} onChange={(e) => updateField("max_replicas", parseInt(e.target.value) || 0)} className={inputCls} />
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold dark:text-gray-100">Thresholds &amp; Cooldowns</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1 dark:text-gray-300">CPU Threshold (%)</label>
            <input aria-label="CPU threshold percentage" type="number" value={form.cpu_threshold_pct} onChange={(e) => updateField("cpu_threshold_pct", parseInt(e.target.value) || 0)} className={inputCls} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1 dark:text-gray-300">Memory Threshold (%)</label>
            <input aria-label="Memory threshold percentage" type="number" value={form.memory_threshold_pct} onChange={(e) => updateField("memory_threshold_pct", parseInt(e.target.value) || 0)} className={inputCls} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1 dark:text-gray-300">Scale-Up Cooldown (s)</label>
            <input aria-label="Scale-up cooldown seconds" type="number" value={form.scale_up_cooldown_s} onChange={(e) => updateField("scale_up_cooldown_s", parseInt(e.target.value) || 0)} className={inputCls} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1 dark:text-gray-300">Scale-Down Cooldown (s)</label>
            <input aria-label="Scale-down cooldown seconds" type="number" value={form.scale_down_cooldown_s} onChange={(e) => updateField("scale_down_cooldown_s", parseInt(e.target.value) || 0)} className={inputCls} />
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4 dark:text-gray-100">Custom Metrics</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Metric</th><th>Target</th><th>Current</th><th>Utilization</th></tr></thead>
          <tbody>
            {form.custom_metrics.map((m, i) => {
              const pct = m.target > 0 ? Math.round((m.current / m.target) * 100) : 0;
              return (
                <tr key={i} className="border-b">
                  <td className="py-2 font-medium">{m.name}</td>
                  <td>{m.target}</td>
                  <td>{m.current}</td>
                  <td>
                    <div className="flex items-center gap-2">
                      <div className="w-20 bg-gray-200 rounded-full h-2">
                        <div className={`h-2 rounded-full ${pct > 80 ? "bg-red-500" : "bg-blue-600"}`} style={{ width: `${Math.min(pct, 100)}%` }} />
                      </div>
                      <span className="text-xs">{pct}%</span>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold dark:text-gray-100">Predictive Scaling</h2>
        <div className="flex items-center gap-3">
          <input aria-label="Enable predictive scaling" id="predictive-scaling" type="checkbox" checked={form.predictive_scaling} onChange={(e) => updateField("predictive_scaling", e.target.checked)} className="w-4 h-4" />
          <label htmlFor="predictive-scaling">Enable predictive scaling based on historical patterns</label>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-2 dark:text-gray-100">HPA YAML Preview</h2>
        <pre className="bg-gray-50 dark:bg-gray-900 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap font-mono">{form.hpa_yaml}</pre>
      </div>

      <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
        <div className="text-sm text-gray-600 dark:text-gray-400">Estimated Monthly Cost</div>
        <div className="text-2xl font-bold text-blue-600">${form.cost_estimate_monthly}</div>
      </div>
    </div>
  );
}
