"use client";

import { useState } from "react";
import { useComplianceGapHeatmap } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Grid3x3, Download } from "lucide-react";

export default function ComplianceGapHeatmapPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useComplianceGapHeatmap();
  const [selectedCell, setSelectedCell] = useState<string | null>(null);

  if (loading) return <div className="p-8 text-gray-400">Loading heatmap...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const statusColors: Record<string, string> = {
    compliant: "bg-green-600",
    partial: "bg-yellow-600",
    gap: "bg-red-600",
    "not_applicable": "bg-gray-700",
  };

  const selectedDetail = selectedCell ? data?.drill_down?.[selectedCell] : null;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Compliance Gap Heatmap</h1>
          <p className="text-sm text-gray-400 mt-1">Visualize compliance gaps across frameworks and categories</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" /> Export
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 mb-4">
        {Object.entries(statusColors).map(([status, color]) => (
          <div key={status} className="flex items-center gap-1">
            <span className={"w-3 h-3 rounded " + color} />
            <span className="text-xs text-gray-400 capitalize">{status.replace("_", " ")}</span>
          </div>
        ))}
      </div>

      {/* Heatmap Grid */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr>
                <th scope="col" className="text-left text-xs text-gray-400 p-2 sticky left-0 bg-gray-900">Framework</th>
                {(data?.control_categories ?? []).map((cat: any) => (
                  <th scope="col" key={cat} className="text-center text-xs text-gray-400 p-2 min-w-24">{cat}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {(data?.frameworks ?? []).map((fw: any) => (
                <tr key={fw}>
                  <td className="text-sm font-medium p-2 sticky left-0 bg-gray-900">{fw}</td>
                  {(data?.control_categories ?? []).map((cat: any) => {
                    const cellKey = fw + ":" + cat;
                    const cellData = data?.heatmap?.[cellKey];
                    const status = cellData?.status ?? "not_applicable";
                    return (
                      <td key={cat} className="p-1 text-center">
                        <button
                          onClick={() => setSelectedCell(selectedCell === cellKey ? null : cellKey)}
                          className={"w-20 h-12 rounded " + (statusColors[status] ?? "bg-gray-700") + " " + (selectedCell === cellKey ? "ring-2 ring-white" : "")}
                        >
                          <span className="text-xs font-bold text-white">{cellData?.controls_count ?? 0}</span>
                        </button>
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Drill-Down Detail */}
      {selectedDetail && (
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Detail: {selectedCell}</h2>
          <div className="space-y-2">
            {selectedDetail.map((d: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-mono text-blue-400">{d.control}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (statusColors[d.status] ?? "bg-gray-700") + " text-white"}>
                    {d.status}
                  </span>
                </div>
                <p className="text-xs text-gray-400">Requirement: {d.requirement}</p>
                <p className="text-xs text-gray-400">Current: {d.current_state}</p>
                {d.remediation && <p className="text-xs text-yellow-400 mt-1">Remediation: {d.remediation}</p>}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
