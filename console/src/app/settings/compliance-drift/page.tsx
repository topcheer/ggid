"use client";

import { useState, useEffect, useCallback } from "react";
import { GitCompare, Gauge, TrendingDown } from "lucide-react";

interface ChangedControl {
  control_id: string;
  name: string;
  was_status: string;
  now_status: string;
  drift_score: number;
  risk_level: "low" | "medium" | "high";
}

interface DriftData {
  framework: string;
  drift_score: number;
  changed_controls: ChangedControl[];
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

const frameworks = ["SOC 2", "ISO 27001", "GDPR", "HIPAA", "PCI DSS", "NIST CSF"];

export default function ComplianceDriftPage() {
  const [framework, setFramework] = useState("SOC 2");
  const [data, setData] = useState<DriftData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/compliance-drift?framework=${encodeURIComponent(framework)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [framework]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const scoreColor = data ? (data.drift_score >= 50 ? "#ef4444" : data.drift_score >= 25 ? "#f59e0b" : "#10b981") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitCompare className="w-6 h-6 text-orange-500" /> Compliance Drift</h1>
        <p className="text-sm text-gray-500 mt-1">Track compliance posture changes between assessments.</p>
      </div>

      <select value={framework} onChange={(e) => setFramework(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        {frameworks.map((f) => <option key={f} value={f}>{f}</option>)}
      </select>

      {data && (
        <>
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-6">
            <div className="relative w-24 h-24">
              <svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${data.drift_score * 1.76} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.drift_score}</span><span className="text-[10px] text-gray-400">drift</span></div>
            </div>
            <div><h3 className="font-semibold">{framework}</h3><p className="text-sm text-gray-500 mt-1">{data.changed_controls.length} controls changed since last assessment</p></div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Control</th><th className="px-4 py-3 text-left font-medium">Was</th><th className="px-4 py-3 text-left font-medium">Now</th><th className="px-4 py-3 text-left font-medium">Drift</th><th className="px-4 py-3 text-left font-medium">Risk</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {data.changed_controls.map((c, i) => (
                  <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-3"><span className="font-mono text-xs">{c.control_id}</span><p className="text-xs text-gray-400">{c.name}</p></td>
                    <td className="px-4 py-3 text-xs text-gray-500">{c.was_status}</td>
                    <td className="px-4 py-3 text-xs font-medium">{c.now_status}</td>
                    <td className="px-4 py-3"><span className="font-bold text-orange-600">{c.drift_score}</span></td>
                    <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${riskColors[c.risk_level]}`}>{c.risk_level}</span></td>
                  </tr>
                ))}
                {data.changed_controls.length === 0 && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No drift detected.</td></tr>}
              </tbody>
            </table>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
