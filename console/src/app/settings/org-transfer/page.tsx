"use client";

import { useState, useEffect, useCallback } from "react";
import { ArrowRightLeft, Search, AlertTriangle, X, Play, Building2, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface User {
  user_id: string;
  username: string;
  email: string;
  org_id: string;
  org_name: string;
  role: string;
}

interface TransferImpact {
  roles_revoked: string[];
  default_role_assigned: string;
  managers_notified: string[];
  policies_affected: number;
  sessions_revoked: number;
}

export default function OrgTransferPage() {
  const t = useTranslations();

  const [users, setUsers] = useState<User[]>([]);
  const [search, setSearch] = useState("");
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [newOrgId, setNewOrgId] = useState("");
  const [impact, setImpact] = useState<TransferImpact | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [transferError, setTransferError] = useState("");

  const retryLoadUsers = () => { setError(""); setLoading(true); fetch("/api/v1/identity/users", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).then(async (res) => {
    if (!res.ok) throw new Error(`Failed to load users: HTTP ${res.status}`);
    const data = await res.json(); setUsers(data.users || data || []);
  }).catch((e) => {
    setError(e instanceof Error ? e.message : "Failed to load users");
  }).finally(() => setLoading(false)); };

  useEffect(() => {
    setLoading(true); setError("");
    fetch("/api/v1/identity/users", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).then(async (res) => {
      if (!res.ok) throw new Error(`Failed to load users: HTTP ${res.status}`);
      const data = await res.json(); setUsers(data.users || data || []);
    }).catch((e) => {
      setError(e instanceof Error ? e.message : "Failed to load users");
    }).finally(() => setLoading(false));
  }, []);

  const previewImpact = useCallback(async () => {
    if (!selectedUser || !newOrgId) return;
    setPreviewing(true); setTransferError("");
    try {
      const res = await fetch("/api/v1/identity/org-transfer/preview", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ user_id: selectedUser.user_id, new_org_id: newOrgId }) });
      if (!res.ok) throw new Error(`Preview failed: HTTP ${res.status}`);
      setImpact(await res.json());
    } catch (e) {
      setTransferError(e instanceof Error ? e.message : "Failed to preview transfer");
    } finally { setPreviewing(false); }
  }, [selectedUser, newOrgId]);

  const execute = async () => {
    if (!selectedUser) return;
    setExecuting(true); setTransferError("");
    try {
      const res = await fetch("/api/v1/identity/org-transfer/execute", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ user_id: selectedUser.user_id, new_org_id: newOrgId }) });
      if (!res.ok) throw new Error(`Transfer failed: HTTP ${res.status}`);
      setShowConfirm(false); setSelectedUser(null); setImpact(null); setNewOrgId("");
    } catch (e) {
      setTransferError(e instanceof Error ? e.message : "Failed to execute transfer");
    } finally { setExecuting(false); }
  };

  const filtered = users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase()));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ArrowRightLeft className="w-6 h-6 text-blue-500" /> {t("orgTransfer.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Transfer users between organizations with impact preview.</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* User list */}
        <div className="space-y-3">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input type="text" placeholder="Search users..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          </div>
          {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span>{error}</span><button onClick={retryLoadUsers} className="text-xs underline hover:text-red-700">Retry</button></div>}
          {loading ? (
            <div className="rounded-lg border dark:border-gray-800 p-8 text-center">
              <div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" />
              <div className="text-sm text-gray-500">Loading users...</div>
            </div>
          ) : (
          <div className="rounded-lg border dark:border-gray-800 max-h-80 overflow-y-auto">
            <div className="divide-y dark:divide-gray-800">
              {filtered.length === 0 ? (
                <div className="px-3 py-4 text-center text-sm text-gray-500">No users found.</div>
              ) : filtered.slice(0, 30).map((u) => (
                <button key={u.user_id} onClick={() => { setSelectedUser(u); setNewOrgId(""); setImpact(null); }} className={`w-full text-left px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-900/30 ${selectedUser?.user_id === u.user_id ? "bg-blue-50 dark:bg-blue-900/20" : ""}`}>
                  <div className="text-sm font-medium">{u.username}</div>
                  <div className="text-xs text-gray-400">{u.role} · {u.org_name}</div>
                </button>
              ))}
            </div>
          </div>
          )}
        </div>

        {/* Transfer form + impact */}
        <div className="lg:col-span-2">
          {transferError && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600">{transferError}</div>}
          {selectedUser ? (
            <div className="space-y-4">
              <div className="rounded-lg border dark:border-gray-800 p-4">
                <div className="flex items-center gap-2 mb-3"><Building2 className="w-4 h-4 text-gray-400" /><span className="text-sm text-gray-500">Current: <span className="font-medium text-gray-700 dark:text-gray-300">{selectedUser.org_name}</span> ({selectedUser.role})</span></div>
                <div className="flex items-center gap-2 mb-3"><ArrowRightLeft className="w-4 h-4 text-blue-400" /><span className="text-sm text-gray-500">Transfer to:</span><input type="text" value={newOrgId} onChange={(e) => setNewOrgId(e.target.value)} placeholder="new-org-uuid" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
                <button onClick={previewImpact} disabled={!newOrgId || previewing} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{previewing ? "Previewing..." : "Preview Impact"}</button>
                {previewing && (
                  <div className="mt-2 rounded-lg border dark:border-gray-800 p-3 text-center">
                    <div className="inline-block w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-1" />
                    <div className="text-xs text-gray-500">Previewing impact...</div>
                  </div>
                )}
              </div>

              {impact && (
                <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
                  <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-orange-500" /> Transfer Impact</h3>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="rounded-lg bg-red-50 dark:bg-red-900/20 p-3"><span className="text-xs text-gray-500">Sessions Revoked</span><p className="text-xl font-bold text-red-600">{impact.sessions_revoked}</p></div>
                    <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3"><span className="text-xs text-gray-500">Policies Affected</span><p className="text-xl font-bold text-blue-600">{impact.policies_affected}</p></div>
                  </div>
                  <div><span className="text-xs text-gray-400">Roles Revoked</span><div className="flex flex-wrap gap-1 mt-1">{impact.roles_revoked.map((r, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 font-mono">{r}</span>)}</div></div>
                  <div><span className="text-xs text-gray-400">Default Role Assigned</span><p className="text-sm font-mono mt-0.5">{impact.default_role_assigned}</p></div>
                  <div><span className="text-xs text-gray-400">Managers Notified</span><div className="flex flex-wrap gap-1 mt-1">{impact.managers_notified.map((m, i) => <span key={i} className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400">{m}</span>)}</div></div>
                  <button onClick={() => setShowConfirm(true)} className="w-full px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 flex items-center justify-center gap-2"><Play className="w-4 h-4" /> Execute Transfer</button>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-gray-500 text-center py-8">Select a user to begin transfer.</p>
          )}
        </div>
      </div>

      {/* Confirm modal */}
      {showConfirm && selectedUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowConfirm(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /> Confirm Transfer</h3>
              <button onClick={() => setShowConfirm(false)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 text-sm space-y-2">
              <p>Transfer <span className="font-medium">{selectedUser.username}</span> from <span className="font-mono">{selectedUser.org_name}</span> to <span className="font-mono">{newOrgId}</span>.</p>
              {impact && <p className="text-orange-600">{impact.sessions_revoked} sessions will be revoked, {impact.roles_revoked.length} roles removed.</p>}
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowConfirm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={execute} disabled={executing} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50">{executing ? "Transferring..." : "Confirm Transfer"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
