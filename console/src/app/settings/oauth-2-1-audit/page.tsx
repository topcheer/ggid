"use client";

import { useOAuth21Audit } from "@ggid/sdk-react";
import { CheckCircle, XCircle, AlertTriangle, FileCheck, Zap } from "lucide-react";

export default function OAuth21AuditPage() {
  const { data, loading, error, refresh } = useOAuth21Audit();
  const pct = data?.overall_compliance_pct ?? 0;

  if (loading) return <div className="p-8 text-gray-400">Loading OAuth 2.1 audit...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">OAuth 2.1 Compliance Audit</h1>
          <p className="text-sm text-gray-400 mt-1">Verify OAuth 2.1 specification compliance</p>
        </div>
        <button onClick={refresh} aria-label="Refresh audit" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Overall Compliance */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex items-center gap-6">
          <div className="relative w-24 h-24">
            <svg className="w-24 h-24 -rotate-90" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="40" fill="none" stroke="#374151" strokeWidth="10" />
              <circle
                cx="50" cy="50" r="40"
                fill="none"
                stroke={pct === 100 ? "#22c55e" : pct >= 80 ? "#eab308" : "#ef4444"}
                strokeWidth="10"
                strokeDasharray={((pct) / 100 * 251.2) + " " + 251.2}
                strokeLinecap="round"
              />
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className={"text-xl font-bold " + (pct === 100 ? "text-green-400" : pct >= 80 ? "text-yellow-400" : "text-red-400")}>
                {pct}%
              </span>
            </div>
          </div>
          <div>
            <h2 className="text-lg font-semibold">Overall Compliance Score</h2>
            <p className="text-sm text-gray-400">
              {data?.compliance_checklist?.filter((c) => c.status === "pass").length ?? 0} of {data?.compliance_checklist?.length ?? 0} checks passing
            </p>
          </div>
        </div>
      </div>

      {/* Compliance Checklist */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <FileCheck className="w-5 h-5 text-blue-400" />
          Compliance Checklist
        </h2>
        <div className="space-y-2">
          {(data?.compliance_checklist ?? []).map((c) => (
            <div key={c.item} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              {c.status === "pass" ? <CheckCircle className="w-5 h-5 text-green-400 flex-shrink-0" /> :
               c.status === "fail" ? <XCircle className="w-5 h-5 text-red-400 flex-shrink-0" /> :
               <AlertTriangle className="w-5 h-5 text-yellow-400 flex-shrink-0" />}
              <div className="flex-1">
                <p className="text-sm font-medium">{c.item}</p>
                <p className="text-xs text-gray-400">{c.description}</p>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (
                c.status === "pass" ? "bg-green-900 text-green-300" :
                c.status === "fail" ? "bg-red-900 text-red-300" :
                "bg-yellow-900 text-yellow-300"
              )}>
                {c.status.toUpperCase()}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Non-Compliant Clients */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-4 h-4 text-red-400" />
          Non-Compliant Clients
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Client</th>
                <th className="text-left py-2 pr-3">Issue</th>
                <th className="text-left py-2 pr-3">Severity</th>
                <th className="text-left py-2 pr-3">Remediation</th>
              </tr>
            </thead>
            <tbody>
              {(data?.non_compliant_clients ?? []).map((c, i) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.client_id}</td>
                  <td className="py-3 pr-3 text-xs">{c.issue}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      c.severity === "critical" ? "bg-red-900 text-red-300" :
                      c.severity === "high" ? "bg-orange-900 text-orange-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}>
                      {c.severity}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.remediation}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {(data?.non_compliant_clients?.length ?? 0) === 0 && (
          <p className="text-sm text-green-400 mt-2">All clients are compliant</p>
        )}
      </div>

      {/* Remediation Actions */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Zap className="w-4 h-4 text-yellow-400" />
          Recommended Remediation Actions
        </h2>
        <div className="space-y-2">
          {(data?.remediation_actions ?? []).map((a, i) => (
            <div key={i} className="flex items-start gap-3 bg-gray-800 rounded-lg p-3">
              <span className="text-xs font-bold text-blue-400 mt-0.5">{i + 1}.</span>
              <div className="flex-1">
                <p className="text-sm font-medium">{a.action}</p>
                <p className="text-xs text-gray-400">{a.description}</p>
              </div>
              <span className="text-xs text-gray-500">{a.affected_count} clients</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
