"use client";

import { useState } from "react";
import { useAuditComplianceScheduler } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Calendar, Plus, CheckCircle, AlertTriangle, Clock, FileText } from "lucide-react";

export default function AuditComplianceSchedulerPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditComplianceScheduler();
  const [showModal, setShowModal] = useState(false);

  if (loading) return <div className="p-8 text-gray-400">Loading compliance scheduler...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Compliance Audit Scheduler</h1>
          <p className="text-sm text-gray-400 mt-1">Schedule recurring compliance audits and track evidence collection</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-1 px-3 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <Plus className="w-4 h-4" />
            Add Schedule
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Calendar className="w-4 h-4" />
            <span className="text-xs text-gray-400">Scheduled Audits</span>
          </div>
          <p className="text-2xl font-bold">{data?.scheduled_audits?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Evidence Ready</span>
          </div>
          <p className="text-2xl font-bold text-green-400">{data?.evidence_collection_status?.ready ?? 0}/{data?.evidence_collection_status?.total ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Upcoming (30d)</span>
          </div>
          <p className="text-2xl font-bold">{data?.upcoming_deadlines_30d?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Overdue</span>
          </div>
          <p className="text-2xl font-bold text-red-400">{data?.overdue_alerts?.length ?? 0}</p>
        </div>
      </div>

      {/* Scheduled Audits Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Scheduled Audits</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Framework</th>
                <th scope="col" className="text-left py-2 pr-3">Frequency</th>
                <th scope="col" className="text-left py-2 pr-3">Next Run</th>
                <th scope="col" className="text-left py-2 pr-3">Scope</th>
                <th scope="col" className="text-left py-2 pr-3">Owner</th>
              </tr>
            </thead>
            <tbody>
              {(data?.scheduled_audits ?? []).map((a) => (
                <tr key={a.id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-medium">{a.framework}</td>
                  <td className="py-3 pr-3 text-gray-300 font-mono text-xs">{a.frequency_cron}</td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{a.next_run}</td>
                  <td className="py-3 pr-3 text-gray-300 text-xs">{a.scope}</td>
                  <td className="py-3 pr-3 text-gray-300">{a.owner}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Preparation Checklist */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <FileText className="w-5 h-5 text-blue-400" />
            Audit Preparation Checklist
          </h2>
          <div className="space-y-2">
            {(data?.audit_preparation_checklist ?? []).map((item: any, i: number) => (
              <div key={i} className="flex items-center gap-2 bg-gray-800 rounded-lg p-2">
                <span
                  className={"w-2 h-2 rounded-full " + (
                    item.status === "ready" ? "bg-green-500" :
                    item.status === "in_progress" ? "bg-yellow-500" :
                    "bg-red-500"
                  )}
                />
                <span className="text-sm flex-1">{item.task}</span>
                <span className="text-xs text-gray-400 capitalize">{item.status.replace(/_/g, " ")}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Upcoming Deadlines */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">Upcoming Deadlines (30d)</h2>
            <div className="space-y-2">
              {(data?.upcoming_deadlines_30d ?? []).map((d: any, i: number) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                  <div>
                    <p className="text-sm font-medium">{d.framework}</p>
                    <p className="text-xs text-gray-500">{d.description}</p>
                  </div>
                  <span className="text-xs text-yellow-400">{d.days_left}d left</span>
                </div>
              ))}
            </div>
          </div>

          {/* Overdue Alerts */}
          {(data?.overdue_alerts ?? []).length > 0 && (
            <div className="bg-gray-900 rounded-xl p-6 border border-red-800">
              <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
                <AlertTriangle className="w-5 h-5 text-red-400" />
                Overdue Alerts
              </h2>
              <div className="space-y-2">
                {(data?.overdue_alerts ?? []).map((a: any, i: number) => (
                  <div key={i} className="bg-gray-800 rounded-lg p-2">
                    <p className="text-sm font-medium">{a.framework}</p>
                    <p className="text-xs text-red-400">{a.days_overdue} days overdue</p>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Add Schedule Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4 border border-gray-700">
            <h2 className="text-lg font-bold mb-4">Add Audit Schedule</h2>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Framework</label>
                <select aria-label="Select option" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm">
                  <option>SOC2</option>
                  <option>ISO27001</option>
                  <option>GDPR</option>
                  <option>HIPAA</option>
                  <option>PCI-DSS</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Cron Expression</label>
                <input aria-label="0 0 1 * *" type="text" placeholder="0 0 1 * *" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Scope</label>
                <input aria-label="All systems" type="text" placeholder="All systems" className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm" />
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium">Create</button>
              <button onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
