"use client";

import { useState, useEffect, useCallback } from "react";
import { Grid3x3 } from "lucide-react";

interface HeatmapData {
  framework: string;
  controls: string[];
  months: string[];
  scores: Record<string, Record<string, number>>;
}

const frameworks = ["SOC 2", "ISO 27001", "GDPR", "HIPAA", "PCI DSS", "NIST CSF"];

function scoreColor(score: number): string {
  if (score >= 90) return "bg-green-500 text-white";
  if (score >= 75) return "bg-green-400 text-white";
  if (score >= 60) return "bg-yellow-400 text-white";
  if (score >= 40) return "bg-orange-400 text-white";
  if (score > 0) return "bg-red-400 text-white";
  return "bg-gray-100 dark:bg-gray-800 text-gray-400";
}

export default function ComplianceHeatmapPage() {
  const [framework, setFramework] = useState("SOC 2");
  const [data, setData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/compliance-heatmap?framework=${encodeURIComponent(framework)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [framework]);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Grid3x3 className="w-6 h-6 text-blue-500" /> Compliance Heatmap</h1>
        <p className="text-sm text-gray-500 mt-1">Control coverage scores across months per framework.</p>
      </div>

      <select value={framework} onChange={(e) => setFramework(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        {frameworks.map((f) => <option key={f} value={f}>{f}</option>)}
      </select>

      {data && (
        <>
          {/* Heatmap grid */}
          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50 sticky top-0">
                <tr>
                  <th className="px-4 py-3 text-left font-medium sticky left-0 bg-gray-50 dark:bg-gray-900/50">Control</th>
                  {data.months.map((m) => <th key={m} className="px-3 py-3 text-center font-medium text-xs whitespace-nowrap">{m}</th>)}
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {data.controls.map((ctrl) => (
                  <tr key={ctrl} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-2 font-mono text-xs font-medium sticky left-0 bg-white dark:bg-gray-900 whitespace-nowrap">{ctrl}</td>
                    {data.months.map((m) => {
                      const score = data.scores[ctrl]?.[m] || 0;
                      return (
                        <td key={m} className="px-1 py-1 text-center">
                          <div className={`w-12 h-9 rounded flex items-center justify-center text-xs font-bold ${scoreColor(score)}`} title={`${ctrl} - ${m}: ${score}%`}>
                            {score > 0 ? score : "-"}
                          </div>
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Legend */}
          <div className="flex items-center gap-4 text-xs">
            <span className="text-gray-500">Legend:</span>
            <span className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-green-500" /> 90+</span>
            <span className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-green-400" /> 75-89</span>
            <span className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-yellow-400" /> 60-74</span>
            <span className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-orange-400" /> 40-59</span>
            <span className="flex items-center gap-1"><span className="w-4 h-4 rounded bg-red-400" /> 0-39</span>
          </div>
        </>
      )}

      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
