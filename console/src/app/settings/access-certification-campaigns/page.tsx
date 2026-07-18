"use client";

import { useAccessCertificationCampaigns } from "@ggid/sdk-react";
import { ClipboardCheck, Users, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AccessCertificationCampaignsPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAccessCertificationCampaigns();

  if (loading) return <div className="p-8 text-gray-400">Loading access certification campaigns...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Access Certification Campaigns</h1>
          <p className="text-sm text-gray-400 mt-1">Periodic access review and certification</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Campaigns */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        {(data?.campaigns ?? []).map((c: any) => (
          <div key={c.id} className="bg-gray-900 rounded-xl p-5">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <ClipboardCheck className="w-5 h-5 text-blue-400" />
                <h3 className="text-sm font-semibold">{c.name}</h3>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (
                c.status === "active" ? "bg-green-900 text-green-300" :
                c.status === "completed" ? "bg-gray-700 text-gray-300" :
                "bg-yellow-900 text-yellow-300"
              )}>{c.status}</span>
            </div>
            <div className="space-y-1 mb-3">
              <p className="text-xs text-gray-400">Scope: {c.scope}</p>
              <p className="text-xs text-gray-400">Period: {c.period}</p>
            </div>
            <div className="mb-2">
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs text-gray-500">Completion</span>
                <span className="text-xs font-bold">{c.completion_pct}%</span>
              </div>
              <div className="w-full bg-gray-800 rounded-full h-2">
                <div className="bg-blue-600 h-2 rounded-full" style={{ width: c.completion_pct + "%" }} />
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Reviewer Workload */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Users className="w-4 h-4" /> Reviewer Workload</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Reviewer</th>
                <th scope="col" className="text-left py-2 pr-3">Assigned</th>
                <th scope="col" className="text-left py-2 pr-3">Completed</th>
                <th scope="col" className="text-left py-2 pr-3">Pending</th>
              </tr>
            </thead>
            <tbody>
              {(data?.reviewer_workload ?? []).map((r: any) => (
                <tr key={r.reviewer} className="border-b border-gray-800">
                  <td className="py-2 pr-3 text-sm">{r.reviewer}</td>
                  <td className="py-2 pr-3 text-xs text-gray-400">{r.assigned}</td>
                  <td className="py-2 pr-3 text-xs text-green-400">{r.completed}</td>
                  <td className="py-2 pr-3 text-xs text-yellow-400">{r.pending}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pending Reviews */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">Pending Reviews Queue</h2>
        <div className="space-y-2">
          {(data?.pending_reviews ?? []).map((rev: any) => (
            <div key={rev.id} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <div className="flex-1">
                <p className="text-sm font-medium">{rev.user}</p>
                <p className="text-xs text-gray-400">Role: {rev.role} - Last accessed: {rev.last_accessed}</p>
              </div>
              <span className="text-xs text-gray-500">Reviewer: {rev.reviewer}</span>
              <div className="flex gap-1">
                <button className="text-xs px-2 py-1 bg-green-900 text-green-300 rounded hover:bg-green-800">Certify</button>
                <button className="text-xs px-2 py-1 bg-red-900 text-red-300 rounded hover:bg-red-800">Revoke</button>
                <button className="text-xs px-2 py-1 bg-yellow-900 text-yellow-300 rounded hover:bg-yellow-800">Modify</button>
              </div>
            </div>
          ))}
          {(data?.pending_reviews ?? []).length === 0 && <p className="text-sm text-gray-500 text-center py-4">No pending reviews</p>}
        </div>
      </div>

      {/* Auto-Escalation */}
      {data?.auto_escalation && (
        <div className="bg-gray-900 rounded-xl p-6 mt-6">
          <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><AlertCircle className="w-4 h-4 text-yellow-400" /> Auto-Escalation</h2>
          <p className="text-sm text-gray-300">{data.auto_escalation}</p>
        </div>
      )}
    </div>
  );
}
