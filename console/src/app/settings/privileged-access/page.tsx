"use client";

import { useState, useEffect, useCallback } from "react";
import { Crown, Trash2, AlertTriangle, Clock, Shield, CheckCircle2, X } from "lucide-react";

interface PrivilegedAccount {
  id: string;
  user_id: string;
  username: string;
  email: string;
  roles: string[];
  granted_at: string;
  justification: string;
  expires_at: string;
  days_until_expiry: number;
}

export default function PrivilegedAccessPage() {
  const [accounts, setAccounts] = useState<PrivilegedAccount[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [revoking, setRevoking] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const fetchAccounts = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/privileged-access", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setAccounts(data.accounts || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchAccounts(); }, [fetchAccounts]);

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const batchRevoke = async () => {
    setRevoking(true);
    try {
      await fetch("/api/v1/policy/privileged-access/batch-revoke", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ ids: [...selectedIds] }),
      });
      setAccounts((prev) => prev.filter((a) => !selectedIds.has(a.id)));
      setSelectedIds(new Set());
      setShowConfirm(false);
    } catch { /* noop */ }
    finally { setRevoking(false); }
  };

  const expiringSoon = accounts.filter((a) => a.days_until_expiry <= 7);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Crown className="w-6 h-6 text-yellow-500" /> Privileged Access Management</h1>
          <p className="text-sm text-gray-500 mt-1">Monitor and revoke privileged account access.</p>
        </div>
        {selectedIds.size > 0 && (
          <button onClick={() => setShowConfirm(true)} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 flex items-center gap-2">
            <Trash2 className="w-4 h-4" /> Revoke Selected ({selectedIds.size})
          </button>
        )}
      </div>

      {/* Expiring soon alert */}
      {expiringSoon.length > 0 && (
        <div className="rounded-lg border border-orange-200 dark:border-orange-900 bg-orange-50 dark:bg-orange-900/20 p-4">
          <div className="flex items-center gap-2 mb-2">
            <AlertTriangle className="w-5 h-5 text-orange-500" />
            <span className="font-semibold text-orange-700 dark:text-orange-400">{expiringSoon.length} privileged access{expiringSoon.length > 1 ? "es" : ""} expiring within 7 days</span>
          </div>
          <div className="space-y-1">
            {expiringSoon.map((a) => (
              <div key={a.id} className="flex items-center gap-2 text-sm text-orange-600 dark:text-orange-400">
                <Clock className="w-3 h-3" />
                <span className="font-medium">{a.username}</span>
                <span>expires in {a.days_until_expiry} day{a.days_until_expiry !== 1 ? "s" : ""} ({a.expires_at})</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Total Privileged</span>
          <p className="text-2xl font-bold mt-1">{accounts.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Expiring Soon</span>
          <p className="text-2xl font-bold mt-1 text-orange-600">{expiringSoon.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Expired</span>
          <p className="text-2xl font-bold mt-1 text-red-600">{accounts.filter((a) => a.days_until_expiry <= 0).length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Unique Roles</span>
          <p className="text-2xl font-bold mt-1">{new Set(accounts.flatMap((a) => a.roles)).size}</p>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th className="px-4 py-3 text-left font-medium w-8">
                <input type="checkbox" checked={selectedIds.size === accounts.length && accounts.length > 0} onChange={(e) => setSelectedIds(e.target.checked ? new Set(accounts.map((a) => a.id)) : new Set())} className="rounded" />
              </th>
              <th className="px-4 py-3 text-left font-medium">User</th>
              <th className="px-4 py-3 text-left font-medium">Roles</th>
              <th className="px-4 py-3 text-left font-medium">Granted</th>
              <th className="px-4 py-3 text-left font-medium">Justification</th>
              <th className="px-4 py-3 text-left font-medium">Expires</th>
              <th className="px-4 py-3 text-left font-medium">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {accounts.map((a) => {
              const isExpired = a.days_until_expiry <= 0;
              const isExpiring = a.days_until_expiry > 0 && a.days_until_expiry <= 7;
              return (
                <tr key={a.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-4 py-3"><input type="checkbox" checked={selectedIds.has(a.id)} onChange={() => toggleSelect(a.id)} className="rounded" /></td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Shield className="w-3 h-3 text-gray-400" />
                      <div>
                        <span className="font-medium">{a.username}</span>
                        <p className="text-xs text-gray-400">{a.email}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {a.roles.map((r, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400 font-mono">{r}</span>)}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-gray-500">{a.granted_at}</td>
                  <td className="px-4 py-3 max-w-xs truncate text-gray-500" title={a.justification}>{a.justification || "-"}</td>
                  <td className="px-4 py-3 text-gray-500">{a.expires_at}</td>
                  <td className="px-4 py-3">
                    {isExpired ? <span className="px-2 py-0.5 rounded text-xs bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">Expired</span> :
                     isExpiring ? <span className="px-2 py-0.5 rounded text-xs bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400">{a.days_until_expiry}d left</span> :
                     <span className="px-2 py-0.5 rounded text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"><CheckCircle2 className="w-3 h-3 inline mr-0.5" />Active</span>}
                  </td>
                </tr>
              );
            })}
            {accounts.length === 0 && !loading && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No privileged accounts found.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Batch revoke confirmation */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowConfirm(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /> Confirm Batch Revoke</h3>
              <button onClick={() => setShowConfirm(false)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 text-sm space-y-2">
              <p>Revoking privileged access for <span className="font-bold text-red-600">{selectedIds.size} account{selectedIds.size > 1 ? "s" : ""}</span>:</p>
              <ul className="text-gray-500 ml-4 list-disc max-h-40 overflow-y-auto">
                {accounts.filter((a) => selectedIds.has(a.id)).map((a) => <li key={a.id}>{a.username} ({a.roles.join(", ")})</li>)}
              </ul>
              <p className="text-red-600">This will immediately remove all privileged roles. Users will lose access to admin functions.</p>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowConfirm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={batchRevoke} disabled={revoking} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 disabled:opacity-50 flex items-center gap-1"><Trash2 className="w-4 h-4" /> {revoking ? "Revoking..." : "Revoke Access"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
