"use client";

import { useState } from "react";
import { usePolicyDryRunHistory } from "@ggid/sdk-react";
import { FlaskConical, Play, GitCompare, Download, Filter } from "lucide-react";

export default function PolicyDryRunHistoryPage() {
  const { data, loading, error, refresh, replayRun } = usePolicyDryRunHistory();
  const [filterDecision, setFilterDecision] = useState("all");
  const [compareRuns, setCompareRuns] = useState<string[]>([]);

  if (loading) return <div className="p-8 text-gray-400">Loading dry-run history...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const filtered = (data?.history ?? []).filter((h) => {
    if (filterDecision !== "all" && h.decision !== filterDecision) return false;
    return true;
  });

  const toggleCompare = (id: string) => {
    setCompareRuns((prev) => prev.includes(id) ? prev.filter((r) => r !== id) : prev.length < 2 ? [...prev, id] : prev);
  };

  const run1 = (data?.history ?? []).find((h) => h.run_id === compareRuns[0]);
  const run2 = (data?.history ?? []).find((h) => h.run_id === compareRuns[1]);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Dry-Run History</h1>
          <p className="text-sm text-gray-400 mt-1">Replay, compare, and export policy evaluation runs</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Filter Bar */}
      <div className="flex items-center gap-3 mb-4">
        <Filter className="w-4 h-4 text-gray-400" />
        <select
          value={filterDecision}
          onChange={(e) => setFilterDecision(e.target.value)}
          className="bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm"
        >
          <option value="all">All Decisions</option>
          <option value="allow">Allow</option>
          <option value="deny">Deny</option>
          <option value="not_applicable">Not Applicable</option>
        </select>
        <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium">
          <Download className="w-4 h-4" />
          Export
        </button>
        <span className="text-xs text-gray-500 ml-auto">{compareRuns.length}/2 selected for comparison</span>
      </div>

      {/* History Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Compare</th>
                <th className="text-left py-2 pr-3">Run ID</th>
                <th className="text-left py-2 pr-3">Policy</th>
                <th className="text-left py-2 pr-3">Subject</th>
                <th className="text-left py-2 pr-3">Decision</th>
                <th className="text-left py-2 pr-3">By</th>
                <th className="text-left py-2 pr-3">Duration</th>
                <th className="text-left py-2 pr-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((h) => (
                <tr key={h.run_id} className="border-b border-gray-800">
                  <td className="py-2 pr-3">
                    <input
                      type="checkbox"
                      checked={compareRuns.includes(h.run_id)}
                      onChange={() => toggleCompare(h.run_id)}
                      className="w-3 h-3 accent-blue-600"
                    />
                  </td>
                  <td className="py-2 pr-3 font-mono text-xs text-blue-400">{h.run_id}</td>
                  <td className="py-2 pr-3 text-gray-300 text-xs">{h.policy}</td>
                  <td className="py-2 pr-3 text-gray-300 text-xs">{h.subject}</td>
                  <td className="py-2 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      h.decision === "allow" ? "bg-green-900 text-green-300" :
                      h.decision === "deny" ? "bg-red-900 text-red-300" :
                      "bg-gray-700 text-gray-300"
                    )}>
                      {h.decision}
                    </span>
                  </td>
                  <td className="py-2 pr-3 text-gray-400 text-xs">{h.executed_by}</td>
                  <td className="py-2 pr-3 text-gray-400 text-xs">{h.duration_ms}ms</td>
                  <td className="py-2 pr-3">
                    <button
                      onClick={() => replayRun(h.run_id)}
                      className="text-xs px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded"
                    >
                      <Play className="w-3 h-3 inline" /> Replay
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Compare Runs Diff View */}
      {run1 && run2 && (
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <GitCompare className="w-5 h-5 text-purple-400" />
            Compare Runs
          </h2>
          <div className="grid grid-cols-2 gap-4">
            {[run1, run2].map((run) => (
              <div key={run.run_id} className="bg-gray-800 rounded-lg p-4">
                <p className="text-xs font-mono text-blue-400 mb-2">{run.run_id}</p>
                <div className="space-y-1">
                  <div className="flex justify-between text-xs"><span className="text-gray-400">Policy:</span><span>{run.policy}</span></div>
                  <div className="flex justify-between text-xs"><span className="text-gray-400">Subject:</span><span>{run.subject}</span></div>
                  <div className="flex justify-between text-xs"><span className="text-gray-400">Decision:</span>
                    <span className={run.decision === "allow" ? "text-green-400" : "text-red-400"}>{run.decision}</span>
                  </div>
                  <div className="flex justify-between text-xs"><span className="text-gray-400">Duration:</span><span>{run.duration_ms}ms</span></div>
                  <div className="flex justify-between text-xs"><span className="text-gray-400">By:</span><span>{run.executed_by}</span></div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Saved Run Templates */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <FlaskConical className="w-5 h-5 text-blue-400" />
          Saved Run Templates
        </h2>
        <div className="flex flex-wrap gap-2">
          {(data?.saved_run_templates ?? []).map((t) => (
            <span key={t} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700 hover:border-blue-500 cursor-pointer">
              {t}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
