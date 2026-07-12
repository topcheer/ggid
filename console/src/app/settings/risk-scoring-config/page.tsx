"use client";

import { useState, useCallback } from "react";
import { Sliders, Save, AlertTriangle } from "lucide-react";

interface RiskConfig {
  weights: { geo_velocity: number; ip_reputation: number; device_familiarity: number; time_anomaly: number; failed_attempts: number };
  thresholds: { low: number; medium: number; high: number; critical: number };
  actions_per_level: { low: string; medium: string; high: string; critical: string };
  adaptive_mfa_trigger: boolean;
}

const weightLabels: { key: keyof RiskConfig["weights"]; label: string }[] = [
  { key: "geo_velocity", label: "Geo Velocity" },
  { key: "ip_reputation", label: "IP Reputation" },
  { key: "device_familiarity", label: "Device Familiarity" },
  { key: "time_anomaly", label: "Time Anomaly" },
  { key: "failed_attempts", label: "Failed Attempts" },
];

const actionOptions = ["allow", "log", "challenge", "step_up_mfa", "block"];

export default function RiskScoringConfigPage() {
  const [config, setConfig] = useState<RiskConfig>({
    weights: { geo_velocity: 25, ip_reputation: 25, device_familiarity: 20, time_anomaly: 15, failed_attempts: 15 },
    thresholds: { low: 20, medium: 40, high: 70, critical: 90 },
    actions_per_level: { low: "allow", medium: "log", high: "challenge", critical: "block" },
    adaptive_mfa_trigger: true,
  });
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const totalWeight = Object.values(config.weights).reduce((a, b) => a + b, 0);

  const save = useCallback(async () => {
    setSaving(true);
    try { await fetch("/api/v1/auth/risk-scoring-config", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); setSaved(true); setTimeout(() => setSaved(false), 2000); }
    catch { /* noop */ }
    finally { setSaving(false); }
  }, [config]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Sliders className="w-6 h-6 text-orange-500" /> Risk Scoring Config</h1><p className="text-sm text-gray-500 mt-1">Configure risk scoring weights, thresholds, and automated actions.</p></div>
        <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
      </div>

      {totalWeight !== 100 && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> Total weight is {totalWeight} (should be 100)</div>}
      {saved && <div className="text-sm text-green-600">Saved successfully!</div>}

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold mb-4">Risk Factor Weights</h3>
        <div className="space-y-4">{weightLabels.map((w) => (
          <div key={w.key} className="flex items-center gap-4">
            <span className="text-sm w-32">{w.label}</span>
            <input type="range" min={0} max={50} value={config.weights[w.key]} onChange={(e) => setConfig({ ...config, weights: { ...config.weights, [w.key]: parseInt(e.target.value) } })} className="flex-1" />
            <span className="text-sm font-bold w-10 text-right">{config.weights[w.key]}</span>
          </div>
        ))}</div>
        <div className="mt-4 text-xs text-gray-500">Total: <span className={"font-bold " + (totalWeight === 100 ? "text-green-600" : "text-red-600")}>{totalWeight}/100</span></div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold mb-4">Thresholds & Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {(["low", "medium", "high", "critical"] as const).map((level) => (
            <div key={level} className="flex items-center gap-3">
              <span className={"px-2 py-1 rounded text-xs font-medium " + (level === "low" ? "bg-gray-100 dark:bg-gray-800" : level === "medium" ? "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400" : level === "high" ? "bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400" : "bg-red-100 dark:bg-red-900/30 dark:text-red-400")}>{level}</span>
              <div><label className="text-xs text-gray-500">Score {"\u2265"}</label><input type="number" min={0} max={100} value={config.thresholds[level]} onChange={(e) => setConfig({ ...config, thresholds: { ...config.thresholds, [level]: parseInt(e.target.value) } })} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
              <select value={config.actions_per_level[level]} onChange={(e) => setConfig({ ...config, actions_per_level: { ...config.actions_per_level, [level]: e.target.value } })} className="px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm">{actionOptions.map((a) => <option key={a} value={a}>{a}</option>)}</select>
            </div>
          ))}
        </div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <label className="flex items-center gap-3 cursor-pointer"><input type="checkbox" checked={config.adaptive_mfa_trigger} onChange={(e) => setConfig({ ...config, adaptive_mfa_trigger: e.target.checked })} className="rounded" /><div><span className="text-sm font-medium">Adaptive MFA Trigger</span><p className="text-xs text-gray-500">Automatically require MFA when risk score exceeds medium threshold</p></div></label>
      </div>
    </div>
  );
}
