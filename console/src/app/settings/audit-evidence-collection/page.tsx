"use client";

import { useAuditEvidenceCollection } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { FileCheck, Upload, Clock, CheckCircle, AlertTriangle } from "lucide-react";

export default function AuditEvidenceCollectionPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditEvidenceCollection();

  if (loading) return <div className="p-8 text-gray-400">Loading evidence collection...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Evidence Collection</h1>
          <p className="text-sm text-gray-400 mt-1">Manage evidence collection for compliance audits</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
            <Upload className="w-4 h-4" /> Upload Evidence
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <FileCheck className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Total Requests</p>
          <p className="text-xl font-bold">{data?.evidence_requests?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Collected</p>
          <p className="text-xl font-bold text-green-400">{data?.evidence_requests?.filter((r: any) => r.status === "collected").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Pending</p>
          <p className="text-xl font-bold text-yellow-400">{data?.evidence_requests?.filter((r: any) => r.status === "pending").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Overdue</p>
          <p className="text-xl font-bold text-red-400">{data?.evidence_requests?.filter((r: any) => r.status === "overdue").length ?? 0}</p>
        </div>
      </div>

      {/* Collection Progress per Framework */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Collection Progress by Framework</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {(data?.collection_progress ?? []).map((p: any) => (
            <div key={p.framework} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-semibold">{p.framework}</span>
                <span className={"text-xs font-bold " + (p.progress_pct === 100 ? "text-green-400" : p.progress_pct >= 50 ? "text-yellow-400" : "text-red-400")}>
                  {p.progress_pct}%
                </span>
              </div>
              <div className="h-1.5 bg-gray-700 rounded-full">
                <div className={"h-full rounded-full " + (p.progress_pct === 100 ? "bg-green-500" : p.progress_pct >= 50 ? "bg-yellow-500" : "bg-red-500")} style={{ width: p.progress_pct + "%" }} />
              </div>
              <p className="text-xs text-gray-500 mt-1">{p.collected}/{p.total} controls</p>
            </div>
          ))}
        </div>
      </div>

      {/* Evidence Requests */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Evidence Requests</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Framework</th>
                <th scope="col" className="text-left py-2 pr-3">Control ID</th>
                <th scope="col" className="text-left py-2 pr-3">Requested By</th>
                <th scope="col" className="text-left py-2 pr-3">Deadline</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.evidence_requests ?? []).map((r: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs">{r.framework}</td>
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{r.control_id}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{r.requested_by}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{r.deadline}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      r.status === "collected" ? "bg-green-900 text-green-300" :
                      r.status === "pending" ? "bg-yellow-900 text-yellow-300" :
                      "bg-red-900 text-red-300"
                    )}>
                      {r.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Evidence Repository */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <FileCheck className="w-4 h-4 text-blue-400" />
          Evidence Repository
        </h2>
        <div className="space-y-2">
          {(data?.evidence_repository ?? []).map((f: any, i: number) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              {f.verified ? <CheckCircle className="w-4 h-4 text-green-400" /> : <Clock className="w-4 h-4 text-yellow-400" />}
              <div className="flex-1">
                <p className="text-sm font-medium">{f.file_name}</p>
                <p className="text-xs text-gray-400 font-mono">SHA256: {f.hash.substring(0, 32)}...</p>
              </div>
              <div className="text-right">
                <p className="text-xs text-gray-400">{f.uploaded_by}</p>
                <p className="text-xs text-gray-500">{f.uploaded_at}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
