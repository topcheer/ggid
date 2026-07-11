"use client";

import { useState, useEffect, useCallback } from "react";
import { Camera, History, RotateCcw, X, AlertTriangle } from "lucide-react";

interface Snapshot {
  id: string;
  policy_id: string;
  version: number;
  description: string;
  created_at: string;
  created_by: string;
}

export default function PolicySnapshotsPage() {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [rollbackTarget, setRollbackTarget] = useState<Snapshot | null>(null);
  const [policyId, setPolicyId] = useState("");
  const [description, setDescription] = useState("");
  const [filterPolicy, setFilterPolicy] = useState("");

  const fetchSnapshots = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/snapshots", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setSnapshots(data.snapshots || data || []);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshots();
  }, [fetchSnapshots]);

  const createSnapshot = async () => {
    if (!policyId) return;
    try {
      await fetch("/api/v1/policy/snapshots", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ policy_id: policyId, description }),
      });
      setShowCreate(false);
      setPolicyId("");
      setDescription("");
      fetchSnapshots();
    } catch {
      /* noop */
    }
  };

  const doRollback = async () => {
    if (!rollbackTarget) return;
    try {
      await fetch(`/api/v1/policy/snapshots/${rollbackTarget.id}/rollback`, {
        method: "POST",
        headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      setRollbackTarget(null);
      fetchSnapshots();
    } catch {
      /* noop */
    }
  };

  const filtered = filterPolicy ? snapshots.filter((s) => s.policy_id.includes(filterPolicy)) : snapshots;
  const uniquePolicies = [...new Set(snapshots.map((s) => s.policy_id))];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Camera className="w-6 h-6 text-blue-500" /> Policy Snapshots</h1>
          <p className="text-sm text-gray-500 mt-1">Create versioned snapshots and roll back policies.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2">
          <Camera className="w-4 h-4" /> Create Snapshot
        </button>
      </div>

      {/* Filter */}
      <div className="flex items-center gap-3">
        <select value={filterPolicy} onChange={(e) => setFilterPolicy(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">All Policies</option>
          {uniquePolicies.map((p) => (<option key={p} value={p}>{p}</option>))}
        </select>
        <span className="text-sm text-gray-500">{filtered.length} snapshots</span>
      </div>

      {/* Snapshot table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th className="px-4 py-3 text-left font-medium">Snapshot ID</th>
              <th className="px-4 py-3 text-left font-medium">Policy ID</th>
              <th className="px-4 py-3 text-left font-medium">Version</th>
              <th className="px-4 py-3 text-left font-medium">Description</th>
              <th className="px-4 py-3 text-left font-medium">Created By</th>
              <th className="px-4 py-3 text-left font-medium">Created At</th>
              <th className="px-4 py-3 text-left font-medium">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {filtered.map((snap) => (
              <tr key={snap.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs">{snap.id}</td>
                <td className="px-4 py-3 font-mono text-xs">{snap.policy_id}</td>
                <td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400">v{snap.version}</span></td>
                <td className="px-4 py-3 max-w-xs truncate" title={snap.description}>{snap.description || "-"}</td>
                <td className="px-4 py-3">{snap.created_by}</td>
                <td className="px-4 py-3 text-gray-500">{snap.created_at}</td>
                <td className="px-4 py-3">
                  <button onClick={() => setRollbackTarget(snap)} className="text-orange-600 hover:underline text-xs font-medium flex items-center gap-1">
                    <RotateCcw className="w-3 h-3" /> Rollback
                  </button>
                </td>
              </tr>
            ))}
            {filtered.length === 0 && !loading && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No snapshots found.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Create snapshot modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Camera className="w-5 h-5 text-blue-500" /> Create Snapshot</h3>
              <button onClick={() => setShowCreate(false)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-4">
              <div>
                <label className="text-sm font-medium">Policy ID</label>
                <input type="text" value={policyId} onChange={(e) => setPolicyId(e.target.value)} placeholder="policy-uuid" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
              </div>
              <div>
                <label className="text-sm font-medium">Description (optional)</label>
                <textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Pre-deployment checkpoint" rows={3} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
              </div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={createSnapshot} disabled={!policyId} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Rollback confirmation modal */}
      {rollbackTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setRollbackTarget(null)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /> Confirm Rollback</h3>
              <button onClick={() => setRollbackTarget(null)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-2">
              <p className="text-sm">Rolling back policy <span className="font-mono font-medium">{rollbackTarget.policy_id}</span> to snapshot <span className="font-mono font-medium">{rollbackTarget.id}</span> (v{rollbackTarget.version}).</p>
              <p className="text-sm text-orange-600">This will overwrite the current policy configuration. This action cannot be undone.</p>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setRollbackTarget(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={doRollback} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 flex items-center gap-1"><RotateCcw className="w-4 h-4" /> Rollback</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
