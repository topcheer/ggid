"use client";

import { useState } from "react";
import { useIdentitySoftDeleteConfig } from "@ggid/sdk-react";
import { Trash2, RotateCcw, Clock, AlertTriangle, Archive } from "lucide-react";

export default function IdentitySoftDeleteConfigPage() {
  const { data, loading, error, refresh, restoreItem, purgeAll } = useIdentitySoftDeleteConfig();
  const [showPurgeConfirm, setShowPurgeConfirm] = useState(false);

  if (loading) return <div className="p-8 text-gray-400">Loading soft delete config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Soft Delete Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Recoverable deletion with configurable retention and auto-purge</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowPurgeConfirm(true)}
            className="flex items-center gap-1 px-3 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
          >
            <Trash2 className="w-4 h-4" />
            Purge All
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Config */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Retention</span>
          </div>
          <p className="text-2xl font-bold">{data?.retention_days ?? 0}d</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <Trash2 className="w-4 h-4" />
            <span className="text-xs text-gray-400">Auto-Purge After</span>
          </div>
          <p className="text-2xl font-bold">{data?.auto_purge_after_days ?? 0}d</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <RotateCcw className="w-4 h-4" />
            <span className="text-xs text-gray-400">Recoverable Window</span>
          </div>
          <p className="text-2xl font-bold">{data?.recoverable_window_days ?? 0}d</p>
        </div>
      </div>

      {/* Per-Entity Config */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <h2 className="text-sm font-semibold mb-3">Per-Entity Configuration</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {(data?.per_entity_config ?? []).map((e) => (
            <div key={e.entity} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center gap-2 mb-1">
                <Archive className="w-3 h-3 text-gray-400" />
                <p className="text-xs font-medium capitalize">{e.entity}</p>
              </div>
              <p className="text-xs text-gray-400">Retention: {e.retention_days}d</p>
              <span className={"text-xs " + (e.enabled ? "text-green-400" : "text-red-400")}>
                {e.enabled ? "Enabled" : "Disabled"}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Soft-Deleted Items Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Soft-Deleted Items</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Entity</th>
                <th className="text-left py-2 pr-3">Name</th>
                <th className="text-left py-2 pr-3">Deleted At</th>
                <th className="text-left py-2 pr-3">Purge At</th>
                <th className="text-left py-2 pr-3">Restorable</th>
                <th className="text-left py-2 pr-3">Action</th>
              </tr>
            </thead>
            <tbody>
              {(data?.soft_deleted_items ?? []).map((item) => (
                <tr key={item.id} className="border-b border-gray-800">
                  <td className="py-3 pr-3">
                    <span className="text-xs px-2 py-0.5 rounded bg-gray-700 capitalize">{item.entity}</span>
                  </td>
                  <td className="py-3 pr-3 font-medium">{item.name}</td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{item.deleted_at}</td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{item.purge_at}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs " + (item.restorable ? "text-green-400" : "text-red-400")}>
                      {item.restorable ? "Yes" : "No"}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <button
                      onClick={() => restoreItem(item.id)}
                      disabled={!item.restorable}
                      className="text-xs px-2 py-1 bg-green-600 hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed rounded"
                    >
                      <RotateCcw className="w-3 h-3 inline" /> Restore
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Purge Confirm Modal */}
      {showPurgeConfirm && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4 border border-red-700">
            <h2 className="text-lg font-bold text-red-400 mb-2">Confirm Purge All</h2>
            <p className="text-sm text-gray-300 mb-4">
              This will permanently delete all {data?.soft_deleted_items?.length ?? 0} soft-deleted items. This action cannot be undone.
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => { purgeAll(); setShowPurgeConfirm(false); }}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium"
              >
                Purge All
              </button>
              <button
                onClick={() => setShowPurgeConfirm(false)}
                className="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium"
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
