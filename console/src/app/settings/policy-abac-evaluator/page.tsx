"use client";

import { useState } from "react";
import { usePolicyAbacEvaluator } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Play, CheckCircle, XCircle, Clock, FileSearch, Layers } from "lucide-react";

export default function PolicyAbacEvaluatorPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, evaluate } = usePolicyAbacEvaluator();
  const [userAttrs, setUserAttrs] = useState('{"department": "finance", "role": "analyst"}');
  const [resourceAttrs, setResourceAttrs] = useState('{"type": "document", "classification": "confidential"}');
  const [envAttrs, setEnvAttrs] = useState('{"time": "business_hours", "location": "on_premise"}');
  const [action, setAction] = useState("read");

  if (loading) return <div className="p-8 text-gray-400">Loading ABAC evaluator...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const result = data?.decision_result;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">ABAC Policy Evaluator</h1>
          <p className="text-sm text-gray-400 mt-1">Test attribute-based access control decisions with custom inputs</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Attribute Input Form */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Attribute Input</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div>
            <label className="text-xs text-gray-400 mb-1 block">User Attributes (JSON)</label>
            <textarea
              value={userAttrs}
              onChange={(e) => setUserAttrs(e.target.value)}
              rows={3}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-xs font-mono focus:outline-none focus:border-blue-500"
            />
          </div>
          <div>
            <label className="text-xs text-gray-400 mb-1 block">Resource Attributes (JSON)</label>
            <textarea
              value={resourceAttrs}
              onChange={(e) => setResourceAttrs(e.target.value)}
              rows={3}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-xs font-mono focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          <div>
            <label className="text-xs text-gray-400 mb-1 block">Environment Attributes (JSON)</label>
            <textarea
              value={envAttrs}
              onChange={(e) => setEnvAttrs(e.target.value)}
              rows={2}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-xs font-mono focus:outline-none focus:border-blue-500"
            />
          </div>
          <div>
            <label className="text-xs text-gray-400 mb-1 block">Action</label>
            <select
              value={action}
              onChange={(e) => setAction(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            >
              <option value="read">read</option>
              <option value="write">write</option>
              <option value="delete">delete</option>
              <option value="execute">execute</option>
              <option value="admin">admin</option>
            </select>
          </div>
        </div>
        <button
          onClick={() => evaluate(userAttrs, resourceAttrs, envAttrs, action)}
          className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
        >
          <Play className="w-4 h-4" />
          Evaluate
        </button>
      </div>

      {/* Decision Result */}
      {result && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-lg font-semibold mb-4">Decision Result</h2>
          <div className="flex items-center gap-6">
            <div className={"flex items-center gap-3 " + (
              result.decision === "allow" ? "text-green-400" :
              result.decision === "deny" ? "text-red-400" :
              "text-yellow-400"
            )}>
              {result.decision === "allow" ? <CheckCircle className="w-8 h-8" /> :
               result.decision === "deny" ? <XCircle className="w-8 h-8" /> :
               <Clock className="w-8 h-8" />}
              <span className="text-3xl font-bold uppercase">{result.decision}</span>
            </div>
            <div>
              <p className="text-xs text-gray-400">Evaluation Time</p>
              <p className="text-lg font-bold">{result.evaluation_time_ms}ms</p>
            </div>
            {result.obligations && result.obligations.length > 0 && (
              <div>
                <p className="text-xs text-gray-400">Obligations</p>
                <div className="flex gap-1">
                  {result.obligations.map((ob: any, i: number) => (
                    <span key={i} className="text-xs px-2 py-0.5 rounded bg-yellow-900 text-yellow-300">{ob}</span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Matched Rules */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Layers className="w-5 h-5 text-blue-400" />
            Matched Rules
          </h2>
          <div className="space-y-2">
            {(data?.matched_rules ?? []).map((rule: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{rule.policy_name}</p>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      rule.effect === "allow" ? "bg-green-900 text-green-300" :
                      rule.effect === "deny" ? "bg-red-900 text-red-300" :
                      "bg-gray-700 text-gray-400"
                    )}
                  >
                    {rule.effect}
                  </span>
                </div>
                <code className="text-xs text-blue-400 font-mono break-all">{rule.condition_path}</code>
              </div>
            ))}
            {(data?.matched_rules ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No rules matched.</p>
            )}
          </div>
        </div>

        {/* Attribute Resolution Trace */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <FileSearch className="w-5 h-5 text-purple-400" />
            Attribute Resolution Trace
          </h2>
          <div className="space-y-2">
            {(data?.attribute_resolution_trace ?? []).map((step: any, i: number) => (
              <div key={i} className="flex items-start gap-3">
                <div className={"w-6 h-6 rounded flex items-center justify-center text-xs font-bold flex-shrink-0 " + (
                  step.resolved ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300"
                )}>
                  {i + 1}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium">{step.attribute}</p>
                  <p className="text-xs text-gray-400">Source: {step.source}</p>
                  <p className="text-xs font-mono text-blue-400">Value: {step.value}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
