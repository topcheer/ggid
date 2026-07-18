"use client";

import { useState } from "react";
import { usePolicyBreakGlass } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ShieldAlert, Zap, Clock, CheckCircle, XCircle, AlertTriangle } from "lucide-react";

export default function PolicyBreakGlassPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, activateBreakGlass } = usePolicyBreakGlass();
  const [showModal, setShowModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState("");
  const [justification, setJustification] = useState("");
  const [duration, setDuration] = useState(30);

  if (loading) return <div className="p-8 text-gray-400">Loading break glass config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Break Glass Access</h1>
          <p className="text-sm text-gray-400 mt-1">Emergency privileged access with automatic expiration and audit trail</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
          >
            <Zap className="w-4 h-4" />
            Activate Break Glass
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <ShieldAlert className="w-4 h-4" />
            <span className="text-xs text-gray-400">Break Glass Roles</span>
          </div>
          <p className="text-2xl font-bold">{data?.break_glass_roles?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Sessions</span>
          </div>
          <p className="text-2xl font-bold">{data?.active_sessions?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Cooldown Period</span>
          </div>
          <p className="text-2xl font-bold">{data?.cooldown_period_minutes ?? 0}m</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Max Concurrent</span>
          </div>
          <p className="text-2xl font-bold">{data?.max_concurrent ?? 0}</p>
        </div>
      </div>

      {/* Activate Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4 border border-gray-700">
            <h2 className="text-lg font-bold mb-4">Activate Break Glass Access</h2>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Select Role</label>
                <select
                  value={selectedRole}
                  onChange={(e) => setSelectedRole(e.target.value)}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                >
                  <option value="">-- Select --</option>
                  {(data?.break_glass_roles ?? []).map((r) => (
                    <option key={r.role} value={r.role}>{r.role}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Justification</label>
                <textarea
                  value={justification}
                  onChange={(e) => setJustification(e.target.value)}
                  rows={2}
                  placeholder="Reason for emergency access..."
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
              <div>
                <label className="text-xs text-gray-400 mb-1 block">Duration (minutes)</label>
                <input
                  type="number"
                  value={duration}
                  onChange={(e) => setDuration(Number(e.target.value))}
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button
                onClick={() => {
                  activateBreakGlass(selectedRole, justification, duration);
                  setShowModal(false);
                }}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
              >
                Activate
              </button>
              <button
                onClick={() => setShowModal(false)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Break Glass Roles */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Break Glass Roles</h2>
          <div className="space-y-2">
            {(data?.break_glass_roles ?? []).map((r) => (
              <div key={r.role} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{r.role}</p>
                  <span className="text-xs text-gray-400">Expires in {r.auto_expire_minutes}m</span>
                </div>
                <div className="flex items-center gap-3 text-xs text-gray-400">
                  {r.justification_required && (
                    <span className="flex items-center gap-1"><AlertTriangle className="w-3 h-3 text-yellow-400" /> Justification required</span>
                  )}
                  {r.notify_on_use && (
                    <span className="flex items-center gap-1"><CheckCircle className="w-3 h-3 text-green-400" /> Notifies on use</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Active Sessions */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Zap className="w-5 h-5 text-red-400" />
            Active Break Glass Sessions
          </h2>
          <div className="space-y-2">
            {(data?.active_sessions ?? []).map((s) => (
              <div key={s.id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{s.user}</p>
                  <span className="text-xs text-red-400">{s.role}</span>
                </div>
                <p className="text-xs text-gray-400">Expires: {s.expires_at}</p>
                <p className="text-xs text-gray-500">Justification: {s.justification}</p>
              </div>
            ))}
            {(data?.active_sessions ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No active break glass sessions.</p>
            )}
          </div>
        </div>
      </div>

      {/* Usage History */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">Usage History</h2>
        <div className="space-y-2 max-h-48 overflow-y-auto">
          {(data?.usage_history ?? []).map((h: any, i: number) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              {h.outcome === "expired" ? <Clock className="w-4 h-4 text-gray-400" /> :
               h.outcome === "revoked" ? <XCircle className="w-4 h-4 text-red-400" /> :
               <CheckCircle className="w-4 h-4 text-green-400" />}
              <div className="flex-1">
                <p className="text-sm font-medium">{h.user} - {h.role}</p>
                <p className="text-xs text-gray-400">{h.timestamp} - {h.duration_minutes}m</p>
              </div>
              <span className="text-xs text-gray-400">{h.outcome}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
