"use client";

import { useAdaptiveAuthConfig } from "@ggid/sdk-react";
import { Shield, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AdaptiveAuthConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAdaptiveAuthConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading adaptive auth config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const riskColors: Record<string, string> = { low: "bg-green-900 text-green-300", medium: "bg-yellow-900 text-yellow-300", high: "bg-orange-900 text-orange-300", critical: "bg-red-900 text-red-300" };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Adaptive Authentication</h1><p className="text-sm text-gray-400 mt-1">Risk-based step-up authentication</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Shield className="w-4 h-4 text-blue-400" /> Risk Threshold Matrix</h2>
        <div className="grid grid-cols-4 gap-3">
          {(data?.risk_thresholds ?? []).map((t: any) => (
            <div key={t.level} className="bg-gray-800 rounded-lg p-4 text-center">
              <span className={"text-xs px-2 py-0.5 rounded inline-block mb-2 " + (riskColors[t.level] ?? "bg-gray-700")}>{t.level}</span>
              <p className="text-sm font-medium mt-1">{t.required_factor}</p>
              <p className="text-xs text-gray-500 mt-0.5">Score: {t.score_range}</p>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Signal Weights</h2>
          <div className="space-y-3">
            {(data?.signal_weights ?? []).map((s: any) => (
              <div key={s.signal}>
                <div className="flex justify-between text-xs mb-1"><span>{s.signal}</span><span className="text-gray-400">{s.weight}%</span></div>
                <div className="w-full bg-gray-800 rounded-full h-2"><div className="bg-blue-600 h-2 rounded-full" style={{ width: s.weight + "%" }} /></div>
              </div>
            ))}
          </div>
        </div>
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Zap className="w-4 h-4 text-yellow-400" /> Step-Up Triggers</h2>
            <div className="space-y-1">
              {(data?.step_up_triggers ?? []).map((t: any) => (
                <div key={t} className="flex items-center gap-2 bg-gray-800 rounded p-2"><span className="text-xs">{t}</span></div>
              ))}
            </div>
          </div>
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3">Per-Role Override</h2>
            <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">Role</th><th className="text-left py-2">Min Factor</th></tr></thead>
              <tbody>{(data?.role_overrides ?? []).map((r: any) => (
                <tr key={r.role} className="border-b border-gray-800"><td className="py-2">{r.role}</td><td className="py-2 text-xs text-blue-400">{r.min_factor}</td></tr>
              ))}</tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
