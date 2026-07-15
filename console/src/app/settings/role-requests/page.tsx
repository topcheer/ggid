"use client";

import { useState, useEffect, useCallback } from "react";
import { UserPlus, Check, X, Clock, ArrowRight } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RoleRequest {
  id: string;
  requester: string;
  requester_email: string;
  requested_role: string;
  justification: string;
  status: "pending" | "approved" | "rejected" | "completed";
  approval_step: { step: number; total: number; current_approver: string };
  created_at: string;
  decided_at: string | null;
  is_mine: boolean;
}

const statusColors: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  approved: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  completed: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function RoleRequestsPage() {
  const t = useTranslations();

  const [requests, setRequests] = useState<RoleRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState<"pending" | "mine">("pending");
  const [actionId, setActionId] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState({ requested_role: "", justification: "" });

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/role-requests", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setRequests(data.requests || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchRequests(); }, [fetchRequests]);

  const approve = async (id: string) => {
    setActionId(id);
    try {
      await fetch(`/api/v1/policy/role-requests/${id}/approve`, { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      setRequests((prev) => prev.map((r) => r.id === id ? { ...r, status: r.approval_step.step >= r.approval_step.total ? "completed" : "approved" } : r));
    } catch { /* noop */ }
    finally { setActionId(null); }
  };

  const reject = async (id: string) => {
    setActionId(id);
    try {
      await fetch(`/api/v1/policy/role-requests/${id}/reject`, { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      setRequests((prev) => prev.map((r) => r.id === id ? { ...r, status: "rejected" } : r));
    } catch { /* noop */ }
    finally { setActionId(null); }
  };

  const createRequest = async () => {
    if (!createForm.requested_role || !createForm.justification) return;
    try {
      await fetch("/api/v1/policy/role-requests", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(createForm),
      });
      setShowCreate(false);
      setCreateForm({ requested_role: "", justification: "" });
      fetchRequests();
    } catch { /* noop */ }
  };

  const filtered = tab === "pending" ? requests.filter((r) => r.status === "pending") : requests.filter((r) => r.is_mine);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><UserPlus className="w-6 h-6 text-blue-500" /> {t("roleRequests.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Request and approve role assignments with multi-step approval workflow.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><UserPlus className="w-4 h-4" /> Request Role</button>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b dark:border-gray-800">
        <button onClick={() => setTab("pending")} className={`px-4 py-2 text-sm font-medium border-b-2 ${tab === "pending" ? "border-blue-500 text-blue-600" : "border-transparent text-gray-500"}`}>Pending Approvals ({requests.filter((r) => r.status === "pending").length})</button>
        <button onClick={() => setTab("mine")} className={`px-4 py-2 text-sm font-medium border-b-2 ${tab === "mine" ? "border-blue-500 text-blue-600" : "border-transparent text-gray-500"}`}>My Requests ({requests.filter((r) => r.is_mine).length})</button>
      </div>

      {/* Request list */}
      <div className="space-y-3">
        {filtered.map((r) => (
          <div key={r.id} className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{r.requester}</span>
                  <ArrowRight className="w-3 h-3 text-gray-400" />
                  <span className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{r.requested_role}</span>
                  <span className={`px-2 py-0.5 rounded text-xs ${statusColors[r.status]}`}>{r.status}</span>
                </div>
                <p className="text-sm text-gray-500 mt-1">{r.justification}</p>
                <div className="flex items-center gap-3 mt-2 text-xs text-gray-400">
                  <span className="flex items-center gap-1"><Clock className="w-3 h-3" /> {r.created_at}</span>
                  <span>Approver: {r.approval_step.current_approver}</span>
                  <span>Step {r.approval_step.step}/{r.approval_step.total}</span>
                </div>
              </div>
              {r.status === "pending" && tab === "pending" && (
                <div className="flex items-center gap-2 ml-4">
                  <button onClick={() => approve(r.id)} disabled={actionId === r.id} className="px-3 py-1.5 rounded-lg text-xs font-medium text-green-700 bg-green-50 dark:bg-green-900/20 hover:bg-green-100 flex items-center gap-1"><Check className="w-3 h-3" /> Approve</button>
                  <button onClick={() => reject(r.id)} disabled={actionId === r.id} className="px-3 py-1.5 rounded-lg text-xs font-medium text-red-700 bg-red-50 dark:bg-red-900/20 hover:bg-red-100 flex items-center gap-1"><X className="w-3 h-3" /> Reject</button>
                </div>
              )}
            </div>
          </div>
        ))}
        {filtered.length === 0 && !loading && (
          <p className="text-sm text-gray-500 text-center py-8">{tab === "pending" ? "No pending approvals." : "No requests from you."}</p>
        )}
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold">Request Role</h3>
              <button onClick={() => setShowCreate(false)}><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Role</label><input type="text" value={createForm.requested_role} onChange={(e) => setCreateForm({ ...createForm, requested_role: e.target.value })} placeholder="admin" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Justification</label><textarea value={createForm.justification} onChange={(e) => setCreateForm({ ...createForm, justification: e.target.value })} rows={3} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={createRequest} disabled={!createForm.requested_role || !createForm.justification} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Submit Request</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
