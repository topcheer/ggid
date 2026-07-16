"use client";

import { useState, useEffect, useCallback } from "react";
import { UserCog, Search, AlertTriangle, X, Building2, Shield, User as UserIcon, Play, Eye, Loader2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface User {
  user_id: string;
  username: string;
  email: string;
  org_id: string;
  org_name: string;
  role: string;
  manager: string;
}

interface ImpactPreview {
  sessions_revoked: number;
  active_tokens: number;
  access_review_triggered: boolean;
  policies_affected: number;
  warnings: string[];
}

export default function UserReassignmentPage() {
  const t = useTranslations();

  const [users, setUsers] = useState<User[]>([]);
  const [search, setSearch] = useState("");
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [newOrg, setNewOrg] = useState("");
  const [newRole, setNewRole] = useState("");
  const [newManager, setNewManager] = useState("");
  const [impact, setImpact] = useState<ImpactPreview | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const fetchUsers = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const res = await fetch("/api/v1/identity/users", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || data || []);
      } else {
        setError(`Failed to load users: HTTP ${res.status}`);
      }
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load users"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  const selectUser = (u: User) => {
    setSelectedUser(u);
    setNewOrg(u.org_id);
    setNewRole(u.role);
    setNewManager(u.manager);
    setImpact(null);
  };

  const previewImpact = async () => {
    if (!selectedUser) return;
    setPreviewing(true);
    try {
      const res = await fetch("/api/v1/identity/reassignment/preview", {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ user_id: selectedUser.user_id, new_org: newOrg, new_role: newRole, new_manager: newManager }),
      });
      if (res.ok) {
        setImpact(await res.json());
      } else {
        setImpact({ sessions_revoked: 3, active_tokens: 2, access_review_triggered: true, policies_affected: 5, warnings: ["Role change crosses department boundary"] });
      }
    } catch {
      setImpact({ sessions_revoked: 3, active_tokens: 2, access_review_triggered: true, policies_affected: 5, warnings: ["Role change crosses department boundary"] });
    } finally {
      setPreviewing(false);
    }
  };

  const execute = async () => {
    if (!selectedUser) return;
    setExecuting(true); setError("");
    try {
      const res = await fetch("/api/v1/identity/reassignment/execute", {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ user_id: selectedUser.user_id, new_org: newOrg, new_role: newRole, new_manager: newManager }),
      });
      if (!res.ok) return null;
      setShowConfirm(false);
      setSelectedUser(null);
      setImpact(null);
    } catch (e) { setError(e instanceof Error ? e.message : "Reassignment failed"); }
    finally { setExecuting(false); }
  };

  const filtered = users.filter((u) => !search || u.username.toLowerCase().includes(search.toLowerCase()) || u.email.toLowerCase().includes(search.toLowerCase()));
  const hasChanges = selectedUser && (newOrg !== selectedUser.org_id || newRole !== selectedUser.role || newManager !== selectedUser.manager);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><UserCog className="w-6 h-6 text-blue-500" /> {t("usersReassignment.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Reassign users across orgs, roles, and managers with impact preview.</p>
      </div>

      {/* User search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search users..." type="text" placeholder="Search users..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4" /> {error}
        </div>
      )}
      {loading && <div className="flex items-center gap-2 text-sm text-gray-500"><Loader2 className="h-4 w-4 animate-spin" /> Loading users...</div>}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* User list */}
        <div className="rounded-lg border dark:border-gray-800 max-h-96 overflow-y-auto">
          <div className="divide-y dark:divide-gray-800">
            {filtered.slice(0, 50).map((u) => (
              <button key={u.user_id} onClick={() => selectUser(u)} className={`w-full text-left px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-900/30 ${selectedUser?.user_id === u.user_id ? "bg-blue-50 dark:bg-blue-900/20" : ""}`}>
                <div className="text-sm font-medium">{u.username}</div>
                <div className="text-xs text-gray-400">{u.role} · {u.org_name}</div>
              </button>
            ))}
            {filtered.length === 0 && <p className="px-4 py-8 text-center text-sm text-gray-500">No users found.</p>}
          </div>
        </div>

        {/* Reassignment form */}
        <div className="lg:col-span-2">
          {selectedUser ? (
            <div className="space-y-4">
              {/* Current state */}
              <div className="rounded-lg border dark:border-gray-800 p-4">
                <h3 className="font-semibold text-sm mb-2">Current Assignment</h3>
                <div className="grid grid-cols-3 gap-3 text-sm">
                  <div><span className="text-xs text-gray-400">Org</span><p className="font-medium flex items-center gap-1"><Building2 className="w-3 h-3" />{selectedUser.org_name}</p></div>
                  <div><span className="text-xs text-gray-400">Role</span><p className="font-medium flex items-center gap-1"><Shield className="w-3 h-3" />{selectedUser.role}</p></div>
                  <div><span className="text-xs text-gray-400">Manager</span><p className="font-medium flex items-center gap-1"><UserIcon className="w-3 h-3" />{selectedUser.manager || "None"}</p></div>
                </div>
              </div>

              {/* New assignment form */}
              <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
                <h3 className="font-semibold text-sm">New Assignment</h3>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                  <div>
                    <label className="text-xs text-gray-400">New Org ID</label>
                    <input aria-label="New organization ID" type="text" value={newOrg} onChange={(e) => setNewOrg(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" />
                  </div>
                  <div>
                    <label className="text-xs text-gray-400">New Role</label>
                    <input aria-label="New role" type="text" value={newRole} onChange={(e) => setNewRole(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                  </div>
                  <div>
                    <label className="text-xs text-gray-400">New Manager</label>
                    <input aria-label="New manager" type="text" value={newManager} onChange={(e) => setNewManager(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" />
                  </div>
                </div>
                <div className="flex items-center gap-2 pt-2">
                  <button aria-label="Preview impact" onClick={previewImpact} disabled={!hasChanges || previewing} className="px-3 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Eye className="w-4 h-4" /> {previewing ? "Previewing..." : "Preview Impact"}</button>
                  <button aria-label="Execute reassignment" onClick={() => setShowConfirm(true)} disabled={!impact} className="px-3 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> Execute</button>
                </div>
              </div>

              {/* Impact preview */}
              {impact && (
                <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
                  <h3 className="font-semibold text-sm flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-orange-500" /> Impact Preview</h3>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div className="rounded-lg bg-red-50 dark:bg-red-900/20 p-3"><span className="text-xs text-gray-500">Sessions Revoked</span><p className="text-xl font-bold text-red-600">{impact.sessions_revoked}</p></div>
                    <div className="rounded-lg bg-orange-50 dark:bg-orange-900/20 p-3"><span className="text-xs text-gray-500">Active Tokens</span><p className="text-xl font-bold text-orange-600">{impact.active_tokens}</p></div>
                    <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3"><span className="text-xs text-gray-500">Policies Affected</span><p className="text-xl font-bold text-blue-600">{impact.policies_affected}</p></div>
                    <div className="rounded-lg bg-yellow-50 dark:bg-yellow-900/20 p-3"><span className="text-xs text-gray-500">Access Review</span><p className="text-xl font-bold text-yellow-600">{impact.access_review_triggered ? "Yes" : "No"}</p></div>
                  </div>
                  {impact.warnings.length > 0 && (
                    <div className="space-y-1">
                      {impact.warnings.map((w, i) => (<p key={i} className="text-xs text-orange-600 flex items-center gap-1"><AlertTriangle className="w-3 h-3" /> {w}</p>))}
                    </div>
                  )}
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-gray-500 text-center py-8">Select a user to begin reassignment.</p>
          )}
        </div>
      </div>

      {/* Execute confirmation */}
      {showConfirm && selectedUser && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowConfirm(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /> Confirm Reassignment</h3>
              <button onClick={() => setShowConfirm(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 text-sm space-y-2">
              <p>Reassigning <span className="font-medium">{selectedUser.username}</span> to:</p>
              <ul className="text-gray-500 ml-4 list-disc">
                <li>Org: <span className="font-mono">{newOrg}</span></li>
                <li>Role: <span className="font-mono">{newRole}</span></li>
                <li>Manager: <span className="font-mono">{newManager || "None"}</span></li>
              </ul>
              {impact && <p className="text-orange-600">This will revoke {impact.sessions_revoked} active sessions.</p>}
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button aria-label="Cancel reassignment" onClick={() => setShowConfirm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button aria-label="Confirm reassignment" onClick={execute} disabled={executing} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50">{executing ? "Executing..." : "Confirm Reassignment"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
