"use client";
import { useEffect, useState } from "react";

interface CircuitBreakerState {
  service: string;
  state: "closed" | "open" | "half_open";
  failure_rate: number;
  recovery_attempts: number;
}

interface ThresholdConfig {
  failure_rate_threshold: number;
  min_calls: number;
  open_timeout_s: number;
}

interface TimelineEvent {
  timestamp: string;
  event: string;
  severity: "info" | "warn" | "error";
}

const defaultData = {
  services: [
    { service: "auth-service", state: "closed" as const, failure_rate: 0.2, recovery_attempts: 0 },
    { service: "identity-service", state: "half_open" as const, failure_rate: 15.5, recovery_attempts: 3 },
    { service: "policy-service", state: "closed" as const, failure_rate: 1.1, recovery_attempts: 0 },
    { service: "org-service", state: "open" as const, failure_rate: 52.3, recovery_attempts: 5 },
  ] as CircuitBreakerState[],
  threshold_config: { failure_rate_threshold: 50, min_calls: 10, open_timeout_s: 30 } as ThresholdConfig,
  timeline: [
    { timestamp: "16:42:01", event: "org-service circuit opened (failure rate 52.3%)", severity: "error" as const },
    { timestamp: "16:41:30", event: "identity-service entered half-open state", severity: "warn" as const },
    { timestamp: "16:40:15", event: "auth-service circuit recovered", severity: "info" as const },
  ] as TimelineEvent[],
};

export default function CircuitBreakerDashboardPage() {
  const [data] = useState(defaultData);
  const [thresholds, setThresholds] = useState(defaultData.threshold_config);

  const stateColors: Record<string, string> = { closed: "bg-green-100 text-green-700", open: "bg-red-100 text-red-700", half_open: "bg-yellow-100 text-yellow-700" };
  const sevColors: Record<string, string> = { info: "text-gray-500", warn: "text-yellow-600", error: "text-red-600" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Circuit Breaker Dashboard</h1>
      <p className="text-gray-600">Monitor per-service circuit states, thresholds, and recovery.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Service Circuit States</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Service</th><th>State</th><th>Failure Rate</th><th>Recovery Attempts</th><th>Action</th></tr></thead>
          <tbody>
            {data.services.map((s: CircuitBreakerState, i: number) => (
              <tr key={i} className="border-b"><td className="py-2 font-medium">{s.service}</td><td><span className={`px-2 py-1 rounded text-xs ${stateColors[s.state] || ""}`}>{s.state}</span></td><td><span className={s.failure_rate > thresholds.failure_rate_threshold ? "text-red-600 font-medium" : ""}>{s.failure_rate}%</span></td><td>{s.recovery_attempts}</td><td>{s.state !== "closed" && <button className="text-xs text-blue-600 hover:underline">Reset</button>}</td></tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Threshold Configuration</h2>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">Failure Rate Threshold (%)</label><input type="number" value={thresholds.failure_rate_threshold} onChange={(e) => setThresholds({ ...thresholds, failure_rate_threshold: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Min Calls</label><input type="number" value={thresholds.min_calls} onChange={(e) => setThresholds({ ...thresholds, min_calls: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Open Timeout (s)</label><input type="number" value={thresholds.open_timeout_s} onChange={(e) => setThresholds({ ...thresholds, open_timeout_s: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Real-Time Event Timeline</h2>
        <div className="space-y-2">
          {data.timeline.map((ev: TimelineEvent, i: number) => (
            <div key={i} className="flex items-center gap-3 border-b py-2"><span className="text-xs font-mono text-gray-400">{ev.timestamp}</span><span className={`text-sm ${sevColors[ev.severity] || ""}`}>{ev.event}</span></div>
          ))}
        </div>
      </div>
    </div>
  );
}
