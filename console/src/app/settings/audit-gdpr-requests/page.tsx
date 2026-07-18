"use client";

import { useAuditGdprRequests } from "@ggid/sdk-react";
import { FileText, UserCheck, CheckCircle, Clock, AlertTriangle, Play, ShieldCheck } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuditGdprRequestsPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, processRequest } = useAuditGdprRequests();

  if (loading) return <div className="p-8 text-gray-400">Loading GDPR requests...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">GDPR Request Management</h1>
          <p className="text-sm text-gray-400 mt-1">Process data subject access, erasure, portability, and rectification requests</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Pending</span>
          </div>
          <p className="text-2xl font-bold">{data?.request_queue?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Completed (30d)</span>
          </div>
          <p className="text-2xl font-bold">{data?.completed_stats?.total_30d ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <ShieldCheck className="w-4 h-4" />
            <span className="text-xs text-gray-400">SLA Compliance</span>
          </div>
          <p className="text-2xl font-bold">{data?.sla_compliance_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Overdue</span>
          </div>
          <p className="text-2xl font-bold">{data?.completed_stats?.overdue ?? 0}</p>
        </div>
      </div>

      {/* Request Queue */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Request Queue</h2>
        <div className="space-y-3">
          {(data?.request_queue ?? []).map((req) => {
            const daysLeft = req.deadline_days;
            const overdue = daysLeft < 0;
            const urgent = daysLeft >= 0 && daysLeft <= 3;
            return (
              <div key={req.id} className="bg-gray-800 rounded-lg p-4">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div className={"w-10 h-10 rounded-lg flex items-center justify-center " + (
                      req.request_type === "erasure" ? "bg-red-900 text-red-300" :
                      req.request_type === "access" ? "bg-blue-900 text-blue-300" :
                      req.request_type === "portability" ? "bg-purple-900 text-purple-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}>
                      <FileText className="w-5 h-5" />
                    </div>
                    <div>
                      <p className="font-semibold capitalize">{req.request_type}</p>
                      <p className="text-xs text-gray-400">User: {req.user_id}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-xs text-gray-500">Deadline</p>
                    <p
                      className={"text-sm font-bold " + (
                        overdue ? "text-red-400" : urgent ? "text-yellow-400" : "text-green-400"
                      )}
                    >
                      {overdue ? `${Math.abs(daysLeft)}d overdue` : `${daysLeft}d left`}
                    </p>
                  </div>
                </div>

                {/* Data Subject Verification */}
                <div className="flex items-center gap-2 mb-3">
                  <UserCheck className={"w-4 h-4 " + (req.identity_verified ? "text-green-400" : "text-yellow-400")} />
                  <span className="text-xs text-gray-400">
                    {req.identity_verified ? "Identity verified" : "Identity pending verification"}
                  </span>
                  <span
                    className={"text-xs px-2 py-0.5 rounded ml-auto " + (
                      req.status === "pending" ? "bg-yellow-900 text-yellow-300" :
                      req.status === "processing" ? "bg-blue-900 text-blue-300" :
                      "bg-green-900 text-green-300"
                    )}
                  >
                    {req.status}
                  </span>
                </div>

                {/* Anonymization Preview for erasure */}
                {req.request_type === "erasure" && req.anonymization_preview && (
                  <div className="bg-gray-900 rounded-lg p-2 mb-3">
                    <p className="text-xs text-gray-500 mb-1">Anonymization Preview:</p>
                    <div className="flex flex-wrap gap-1">
                      {req.anonymization_preview.map((field: any, i: number) => (
                        <span key={i} className="text-xs px-2 py-0.5 rounded bg-red-900 text-red-300 font-mono">{field}</span>
                      ))}
                    </div>
                  </div>
                )}

                <button
                  onClick={() => processRequest(req.id)}
                  className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-md text-xs font-medium transition"
                >
                  <Play className="w-3 h-3" />
                  Process
                </button>
              </div>
            );
          })}
          {(data?.request_queue ?? []).length === 0 && (
            <div className="bg-gray-800 rounded-xl p-12 text-center text-gray-500">No pending GDPR requests.</div>
          )}
        </div>
      </div>

      {/* Completed Stats */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Completed Requests Breakdown</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {(Object.entries(data?.completed_stats?.by_type ?? {}) as [string, number][]).map(([type, count]) => (
            <div key={type} className="bg-gray-800 rounded-lg p-3 text-center">
              <p className="text-xs text-gray-400 capitalize mb-1">{type}</p>
              <p className="text-xl font-bold text-green-400">{count}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
