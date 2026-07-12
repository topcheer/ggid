"use client";

import { useAuditDataSovereignty } from "@ggid/sdk-react";
import { Globe, Shield, AlertTriangle, CheckCircle, ArrowRight } from "lucide-react";

export default function AuditDataSovereigntyPage() {
  const { data, loading, error, refresh } = useAuditDataSovereignty();

  if (loading) return <div className="p-8 text-gray-400">Loading data sovereignty...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Data Sovereignty</h1>
          <p className="text-sm text-gray-400 mt-1">Data residency, cross-border transfers, and regulatory compliance</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Globe className="w-4 h-4" />
            <span className="text-xs text-gray-400">Residency Regions</span>
          </div>
          <p className="text-2xl font-bold">{data?.data_residency_regions?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Localization Status</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.data_localization_status ?? "unknown"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <ArrowRight className="w-4 h-4" />
            <span className="text-xs text-gray-400">Pending Transfers</span>
          </div>
          <p className="text-2xl font-bold">{data?.pending_transfers?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Violations</span>
          </div>
          <p className="text-2xl font-bold">{data?.sovereignty_violations?.length ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Data Residency Regions */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Data Residency Regions</h2>
          <div className="space-y-2">
            {(data?.data_residency_regions ?? []).map((r) => (
              <div key={r.region} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{r.region}</p>
                  <div className="flex items-center gap-2">
                    {r.allowed ? (
                      <span className="flex items-center gap-1 text-xs text-green-400"><CheckCircle className="w-3 h-3" /> Allowed</span>
                    ) : (
                      <span className="flex items-center gap-1 text-xs text-red-400"><AlertTriangle className="w-3 h-3" /> Blocked</span>
                    )}
                  </div>
                </div>
                {r.encryption_required && (
                  <span className="text-xs px-2 py-0.5 bg-gray-700 rounded text-gray-300">Encryption required</span>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Cross-Border Transfer Rules */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Cross-Border Transfer Rules</h2>
          <div className="space-y-2">
            {(data?.cross_border_transfer_rules ?? []).map((rule, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-blue-400">{rule.source_region}</span>
                  <ArrowRight className="w-3 h-3 text-gray-500" />
                  <span className="text-sm font-medium text-purple-400">{rule.destination_region}</span>
                </div>
                <p className="text-xs text-gray-400">Mechanism: {rule.transfer_mechanism}</p>
                <p className="text-xs text-gray-500">Data types: {rule.data_types.join(", ")}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* GDPR Compliance */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Shield className="w-5 h-5 text-green-400" />
          GDPR Compliance
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
            <span className="text-sm text-gray-300">Article 45 (Adequacy Decision)</span>
            <span className={"text-xs px-2 py-0.5 rounded " + (data?.gdpr_article_45 ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
              {data?.gdpr_article_45 ? "Compliant" : "Not Met"}
            </span>
          </div>
          <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
            <span className="text-sm text-gray-300">Article 49 (Derogations)</span>
            <span className={"text-xs px-2 py-0.5 rounded " + (data?.gdpr_article_49 ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
              {data?.gdpr_article_49 ? "Compliant" : "Not Met"}
            </span>
          </div>
        </div>
      </div>

      {/* Sovereignty Violations */}
      {(data?.sovereignty_violations ?? []).length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6 mt-6 border border-red-800">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-5 h-5 text-red-400" />
            Sovereignty Violations
          </h2>
          <div className="space-y-2">
            {(data?.sovereignty_violations ?? []).map((v, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{v.violation_type}</p>
                  <span className="text-xs px-2 py-0.5 rounded bg-red-900 text-red-300">{v.severity}</span>
                </div>
                <p className="text-xs text-gray-400">{v.description}</p>
                <p className="text-xs text-gray-500 mt-1">Region: {v.region} - Detected: {v.detected_at}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
