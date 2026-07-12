"use client";

import { useState } from "react";
import { usePolicyEvalTimeline } from "@ggid/sdk-react";
import { Clock, Zap, CheckCircle } from "lucide-react";

export default function PolicyEvalTimelinePage() {
  const { data, loading, error, refresh } = usePolicyEvalTimeline();
  const [selectedPolicy, setSelectedPolicy] = useState("");

  if (loading) return <div className="p-8 text-gray-400">Loading policy eval timeline...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const evalData = data?.evaluations?.find((e) => e.policy === selectedPolicy) ?? data?.evaluations?.[0];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Evaluation Timeline</h1>
          <p className="text-sm text-gray-400 mt-1">Trace policy evaluation step-by-step</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Policy Selector */}
      <div className="mb-6">
        <select value={selectedPolicy} onChange={(e) => setSelectedPolicy(e.target.value)} className="px-3 py-2 bg-gray-800 rounded-lg text-sm">
          {(data?.evaluations ?? []).map((e) => <option key={e.policy} value={e.policy}>{e.policy}</option>)}
        </select>
      </div>

      {evalData && (
        <>
          {/* Total Eval Time + Cache */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
            <div className="bg-gray-900 rounded-xl p-4 text-center">
              <Clock className="w-5 h-5 text-blue-400 mx-auto mb-1" />
              <p className="text-xs text-gray-400">Total Eval Time</p>
              <p className={"text-2xl font-bold " + (evalData.total_eval_time_ms > 100 ? "text-yellow-400" : "text-green-400")}>{evalData.total_eval_time_ms}ms</p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4 text-center">
              <Zap className="w-5 h-5 text-purple-400 mx-auto mb-1" />
              <p className="text-xs text-gray-400">Cache</p>
              <p className={"text-sm font-bold " + (evalData.cache_hit ? "text-green-400" : "text-gray-400")}>{evalData.cache_hit ? "HIT" : "MISS"}</p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4 text-center">
              <CheckCircle className="w-5 h-5 mx-auto mb-1 text-gray-400" />
              <p className="text-xs text-gray-400">Decision</p>
              <p className={"text-sm font-bold " + (evalData.decision === "allow" ? "text-green-400" : "text-red-400")}>{evalData.decision}</p>
            </div>
          </div>

          {/* Step Timeline */}
          <div className="bg-gray-900 rounded-xl p-6 mb-6">
            <h2 className="text-sm font-semibold mb-4">Evaluation Steps</h2>
            <div className="relative pl-6">
              <div className="absolute left-2 top-0 bottom-0 w-0.5 bg-gray-700" />
              {evalData.steps.map((step, i) => (
                <div key={i} className="relative pb-4 last:pb-0">
                  <div className={"absolute -left-4 w-3 h-3 rounded-full " + (step.latency_ms > 50 ? "bg-yellow-500" : "bg-green-500")} />
                  <div className="ml-4">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{step.name}</span>
                      <span className="text-xs text-gray-500">{step.latency_ms}ms</span>
                    </div>
                    <p className="text-xs text-gray-400">{step.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Matched Rules */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3">Matched Rules</h2>
            <div className="space-y-2">
              {evalData.matched_rules.map((r, i) => (
                <div key={i} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs font-mono text-blue-400">{r.rule_id}</span>
                    <span className={"text-xs px-1.5 py-0.5 rounded " + (r.matched ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>{r.matched ? "matched" : "not matched"}</span>
                  </div>
                  <p className="text-xs text-gray-400 font-mono">{r.condition}</p>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
