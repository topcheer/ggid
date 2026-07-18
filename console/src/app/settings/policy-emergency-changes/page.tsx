"use client";

import { useState } from "react";
import { usePolicyEmergencyChanges } from "@ggid/sdk-react";
import { AlertTriangle, Clock, Plus, CheckCircle, XCircle, FileText } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyEmergencyChangesPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, requestEmergency } = usePolicyEmergencyChanges();
  const [showModal, setShowModal] = useState(false);

  if (loading) return <div className="p-8 text-gray-400">Loading emergency changes...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Emergency Policy Changes</h1>
          <p className="text-sm text-gray-400 mt-1">Time-limited emergency policy overrides with auto-revert</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
          >
            <Plus className="w-4 h-4" />
            Request Emergency
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Active Emergencies</p>
          <p className="text-xl font-bold text-red-400">{data?.active_emergencies?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Revert</p>
          <p className="text-sm font-bold">{data?.active_emergencies?.[0]?.expires_at ?? "N/A"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <FileText className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Review Required</p>
          <p className="text-sm font-bold">{data?.post_incident_review_required ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">History (30d)</p>
          <p className="text-xl font-bold">{data?.emergency_history?.length ?? 0}</p>
        </div>
      </div>

      {/* Active Emergencies */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-5 h-5 text-red-400" />
          Active Emergencies
        </h2>
        <div className="space-y-3">
          {(data?.active_emergencies ?? []).map((e: any) => (
            <div key={e.policy} className="bg-red-900/20 border border-red-800 rounded-lg p-4">
              <div className="flex items-start justify-between mb-2">
                <div>
                  <p className="text-sm font-semibold">{e.policy}</p>
                  <p className="text-xs text-gray-400">Change: {e.change_type} - Approved by: {e.approved_by}</p>
                </div>
                <span className="text-xs px-2 py-0.5 rounded bg-red-900 text-red-300 animate-pulse">ACTIVE</span>
              </div>
              <div className="flex items-center gap-4 text-xs text-gray-400">
                <span>Effective: {e.effective_at}</span>
                <span>Expires: <span className="text-yellow-400 font-medium">{e.expires_at}</span></span>
              </div>
              <div className="mt-2">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-gray-500">Time remaining</span>
                  <span className="text-xs font-mono text-yellow-400">{e.time_remaining}</span>
                </div>
                <div className="h-1 bg-gray-700 rounded-full">
                  <div className="h-full bg-yellow-500 rounded-full" style={{ width: e.time_remaining_pct + "%" }} />
                </div>
              </div>
            </div>
          ))}
          {(data?.active_emergencies?.length ?? 0) === 0 && (
            <p className="text-sm text-gray-500">No active emergencies</p>
          )}
        </div>
      </div>

      {/* Emergency History */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">Emergency History (30d)</h2>
        <div className="space-y-2">
          {(data?.emergency_history ?? []).map((h: any) => (
            <div key={h.id} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <span className={"w-2 h-2 rounded-full " + (h.outcome === "reverted" ? "bg-green-500" : h.outcome === "expired" ? "bg-yellow-500" : "bg-red-500")} />
              <div className="flex-1">
                <p className="text-xs font-medium">{h.policy} - {h.change_type}</p>
                <p className="text-xs text-gray-500">{h.approved_by} - {h.timestamp}</p>
              </div>
              <span className="text-xs capitalize text-gray-400">{h.outcome}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Request Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowModal(false)}>
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4" onClick={(ev) => ev.stopPropagation()}>
            <h3 className="text-lg font-semibold mb-4">Request Emergency Change</h3>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-400">Policy</label>
                <input aria-label="Policy name" className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" placeholder="Policy name" />
              </div>
              <div>
                <label className="text-xs text-gray-400">Change Description</label>
                <textarea aria-label="Describe the change" className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" rows={2} placeholder="Describe the change" />
              </div>
              <div>
                <label className="text-xs text-gray-400">Justification</label>
                <textarea aria-label="Why is this an emergency?" className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" rows={3} placeholder="Why is this an emergency?" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-400">Duration (hours)</label>
                  <input aria-label="Input field" type="number" defaultValue={4} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" />
                </div>
                <div>
                  <label className="text-xs text-gray-400">Approver Chain</label>
                  <select aria-label="Select option" className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm">
                    <option>CISO + CTO</option>
                    <option>Security Lead</option>
                    <option>On-call Admin</option>
                  </select>
                </div>
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button
                onClick={() => { requestEmergency(); setShowModal(false); }}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
              >
                Submit Request
              </button>
              <button onClick={() => setShowModal(false)} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
