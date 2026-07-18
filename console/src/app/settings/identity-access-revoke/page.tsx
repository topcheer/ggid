"use client";

import { useState } from "react";
import { useIdentityAccessRevoke } from "@ggid/sdk-react";
import { Zap, Search, AlertTriangle, CheckCircle, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityAccessRevokePage() {
  const { data, loading, error, refresh, executeRevoke } = useIdentityAccessRevoke();
  const [searchQuery, setSearchQuery] = useState("");
  const [revokeAll, setRevokeAll] = useState(false);
  const [selected, setSelected] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [notifyManager, setNotifyManager] = useState(true);
  const [showConfirm, setShowConfirm] = useState(false);
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idAccessRevoke.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const allOptions = ["sessions", "tokens", "api_keys", "app_access", "ssh_keys"];
  const toggleSelect = (key: string) => {
    setSelected((prev) => prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]);
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idAccessRevoke.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idAccessRevoke.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Search & Select */}
        <div className="space-y-6">
          {/* User Search */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Search className="w-5 h-5 text-blue-400" />
              Search User
            </h2>
            <input
              type="text"
              placeholder="Search by username or email..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            />
            <div className="mt-3 space-y-1 max-h-32 overflow-y-auto">
              {(data?.searchable_users ?? []).filter((u) => !searchQuery || u.username.includes(searchQuery)).slice(0, 5).map((u) => (
                <div key={u.user_id} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                  <div>
                    <p className="text-sm font-medium">{u.username}</p>
                    <p className="text-xs text-gray-500">{u.email}</p>
                  </div>
                  <span className={"text-xs px-2 py-0.5 rounded " + (u.active ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                    {u.active ? "Active" : "Inactive"}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Revoke Scope */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">{t("idAccessRevoke.revokeScope")}</h2>
            <label className="flex items-center gap-2 mb-3 cursor-pointer">
              <input
                type="checkbox"
                checked={revokeAll}
                onChange={(e) => { setRevokeAll(e.target.checked); setSelected(e.target.checked ? allOptions : []); }}
                className="w-4 h-4 accent-red-600"
              />
              <span className="text-sm font-medium">{t("idAccessRevoke.revokeEverything")}</span>
            </label>
            <div className="space-y-2">
              {allOptions.map((opt) => (
                <label key={opt} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={revokeAll || selected.includes(opt)}
                    onChange={() => toggleSelect(opt)}
                    disabled={revokeAll}
                    className="w-4 h-4 accent-blue-600"
                  />
                  <span className="text-sm capitalize text-gray-300">{opt.replace(/_/g, " ")}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Reason & Options */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">{t("idAccessRevoke.reason")}</h2>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={2}
              placeholder="Reason for revocation..."
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 mb-3"
            />
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={notifyManager}
                onChange={(e) => setNotifyManager(e.target.checked)}
                className="w-4 h-4 accent-blue-600"
              />
              <span className="text-sm text-gray-300">{t("idAccessRevoke.notifyManager")}</span>
            </label>
            <button
              onClick={() => setShowConfirm(true)}
              disabled={selected.length === 0 && !revokeAll}
              className="w-full mt-4 flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium transition"
            >
              <Zap className="w-4 h-4" />
              Execute Revocation
            </button>
          </div>
        </div>

        {/* Right: Impact & Log */}
        <div className="space-y-6">
          {/* Estimated Impact */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-yellow-400" />
              Estimated Impact
            </h2>
            <div className="grid grid-cols-2 gap-3">
              {(data?.estimated_impact ?? []).map((imp) => (
                <div key={imp.category} className="bg-gray-800 rounded-lg p-3">
                  <p className="text-xs text-gray-400 mb-1 capitalize">{imp.category.replace(/_/g, " ")}</p>
                  <p className="text-xl font-bold text-red-400">{imp.count}</p>
                </div>
              ))}
            </div>
          </div>

          {/* Execution Log */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Activity className="w-5 h-5 text-blue-400" />
              Execution Log
            </h2>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {(data?.execution_log ?? []).map((entry: any, i: number) => (
                <div key={i} className="flex items-center gap-2 bg-gray-800 rounded-lg p-2">
                  {entry.success ? <CheckCircle className="w-3 h-3 text-green-400" /> : <AlertTriangle className="w-3 h-3 text-red-400" />}
                  <div className="flex-1">
                    <p className="text-xs font-medium">{entry.action}</p>
                    <p className="text-xs text-gray-500">{entry.timestamp}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Confirm Modal */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4 border border-red-700">
            <h2 className="text-lg font-bold text-red-400 mb-2">{t("idAccessRevoke.confirmRevocation")}</h2>
            <p className="text-sm text-gray-300 mb-4">
              You are about to revoke: <strong>{revokeAll ? "ALL ACCESS" : selected.join(", ")}</strong>
              {reason && <br />}
              Reason: {reason}
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => { executeRevoke(revokeAll ? allOptions : selected, reason, notifyManager); setShowConfirm(false); }}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
              >
                Confirm
              </button>
              <button
                onClick={() => setShowConfirm(false)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
