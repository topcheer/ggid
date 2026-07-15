"use client";

import { useState, useCallback, useEffect } from "react";
import {
  FileCheck, Plus, X, Check, Ban, Loader2, Filter, Clock, CheckCircle2, XCircle,
} from "lucide-react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";

interface AccessRequest {
  id: string;
  tenant_id: string;
  requester_id: string;
  resource_type: string;
  resource_id: string;
  reason: string;
  status: string;
  approver_id?: string;
  denial_reason?: string;
  created_at: string;
  updated_at: string;
}

const RESOURCE_TYPES = ["role", "permission", "organization", "project", "repository", "dataset", "vault_secret"] as const;
const STATUS_TABS = ["pending", "approved", "denied", "expired", "all"] as const;

const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none";
const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
const btnPrimary = "flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50";
const btnGhost = "rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700";

export default function AccessRequestsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [requests, setRequests] = useState<AccessRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("pending");
  const [showCreate, setShowCreate] = useState(false);

  // Create form
  const [createForm, setCreateForm] = useState({
    requester_id: "",
    resource_type: "role",
    resource_id: "",
    reason: "",
  });
  const [creating, setCreating] = useState(false);

  const loadRequests = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params = statusFilter !== "all" ? `?status=${statusFilter}` : "";
      const data = await apiFetch<{ requests?: AccessRequest[]; items?: AccessRequest[]; count?: number }>(`/api/v1/access-requests${params}`);
      setRequests(data.requests || data.items || []);
    } catch {
      setRequests([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch, statusFilter]);

  useEffect(() => {
    loadRequests();
  }, [loadRequests]);

  const showMessage = (m: string) => {
    setMsg(m);
    setTimeout(() => setMsg(""), 4000);
  };

  const handleCreate = async () => {
    if (!createForm.requester_id.trim() || !createForm.resource_id.trim()) {
      setError("Requester ID and Resource ID are required");
      return;
    }
    setCreating(true);
    setError("");
    try {
      await apiFetch("/api/v1/access-requests", {
        method: "POST",
        body: JSON.stringify(createForm),
      });
      showMessage("Access request created");
      setShowCreate(false);
      setCreateForm({ requester_id: "", resource_type: "role", resource_id: "", reason: "" });
      loadRequests();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create request");
    } finally {
      setCreating(false);
    }
  };

  const handleApprove = async (req: AccessRequest) => {
    // Use current user ID as approver (in production this comes from JWT)
    const approverId = localStorage.getItem("ggid_user_id") || "00000000-0000-0000-0000-000000000001";
    try {
      await apiFetch(`/api/v1/access-requests/${req.id}/approve`, {
        method: "POST",
        body: JSON.stringify({ approver_id: approverId }),
      });
      showMessage(`Request approved`);
      loadRequests();
    } catch {
      showMessage("Failed to approve request (API may not be available)");
    }
  };

  const handleDeny = async (req: AccessRequest) => {
    const denialReason = prompt("Reason for denial? (optional)") || "";
    const approverId = localStorage.getItem("ggid_user_id") || "00000000-0000-0000-0000-000000000001";
    try {
      await apiFetch(`/api/v1/access-requests/${req.id}/deny`, {
        method: "POST",
        body: JSON.stringify({ approver_id: approverId, denial_reason: denialReason }),
      });
      showMessage(`Request denied`);
      loadRequests();
    } catch {
      showMessage("Failed to deny request (API may not be available)");
    }
  };

  const statusBadge = (status: string) => {
    switch (status) {
      case "pending":
        return <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900 dark:text-amber-400"><Clock className="h-3 w-3" /> Pending</span>;
      case "approved":
        return <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900 dark:text-green-400"><CheckCircle2 className="h-3 w-3" /> Approved</span>;
      case "denied":
        return <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900 dark:text-red-400"><XCircle className="h-3 w-3" /> Denied</span>;
      case "expired":
        return <span className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-400"><Clock className="h-3 w-3" /> Expired</span>;
      default:
        return <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-400">{status}</span>;
    }
  };

  const pendingCount = requests.filter((r) => r.status === "pending").length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <FileCheck className="h-6 w-6 text-brand-600" /> Access Requests
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage IGA access request workflows — {pendingCount > 0 && <span className="font-medium text-amber-600">{pendingCount} pending</span>}
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className={btnPrimary}>
          <Plus className="h-4 w-4" /> New Request
        </button>
      </div>

      {/* Messages */}
      {msg && <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}
      {error && <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>}

      {/* Status filter tabs */}
      <div className="flex flex-wrap gap-2">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab}
            onClick={() => setStatusFilter(tab)}
            className={`rounded-lg px-3 py-1.5 text-sm font-medium capitalize transition-colors ${
              statusFilter === tab
                ? "bg-brand-600 text-white"
                : "border border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700"
            }`}
          >
            {tab}
            {tab === "pending" && pendingCount > 0 && (
              <span className={`ml-1.5 rounded-full px-1.5 py-0.5 text-xs ${statusFilter === "pending" ? "bg-white/20" : "bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-400"}`}>
                {pendingCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Loading */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
        </div>
      ) : requests.length === 0 ? (
        <div className={cardCls}>
          <div className="py-8 text-center">
            <FileCheck className="mx-auto mb-3 h-10 w-10 text-gray-400" />
            <p className="text-gray-500 dark:text-gray-400">No {statusFilter !== "all" ? statusFilter : ""} access requests</p>
          </div>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full min-w-[800px]">
            <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Requester</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Resource</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Reason</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {requests.map((req) => (
                <tr key={req.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                  <td className="px-4 py-3">
                    <span className="font-mono text-xs text-gray-700 dark:text-gray-300">{req.requester_id.slice(0, 8)}...</span>
                  </td>
                  <td className="px-4 py-3">
                    <div>
                      <span className="rounded bg-brand-100 px-1.5 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-900 dark:text-brand-400">{req.resource_type}</span>
                      <span className="ml-2 font-mono text-xs text-gray-600 dark:text-gray-400">{req.resource_id}</span>
                    </div>
                  </td>
                  <td className="max-w-xs truncate px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{req.reason || "—"}</td>
                  <td className="px-4 py-3">{statusBadge(req.status)}</td>
                  <td className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400">{new Date(req.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex justify-end gap-1">
                      {req.status === "pending" && (
                        <>
                          <button
                            onClick={() => handleApprove(req)}
                            title="Approve"
                            className="flex items-center gap-1 rounded-lg border border-green-300 px-2 py-1 text-xs font-medium text-green-600 hover:bg-green-50 dark:border-green-800 dark:hover:bg-green-950"
                          >
                            <Check className="h-3.5 w-3.5" /> Approve
                          </button>
                          <button
                            onClick={() => handleDeny(req)}
                            title="Deny"
                            className="flex items-center gap-1 rounded-lg border border-red-300 px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
                          >
                            <Ban className="h-3.5 w-3.5" /> Deny
                          </button>
                        </>
                      )}
                      {req.status === "denied" && req.denial_reason && (
                        <span className="text-xs text-gray-400" title={req.denial_reason}>Reason: {req.denial_reason.slice(0, 30)}</span>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setShowCreate(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
                <Plus className="h-5 w-5 text-brand-600" /> Create Access Request
              </h2>
              <button onClick={() => setShowCreate(false)} className="text-gray-400 hover:text-gray-600" aria-label="Close">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Requester ID *</label>
                <input value={createForm.requester_id} onChange={(e) => setCreateForm({ ...createForm, requester_id: e.target.value })} className={inputCls} placeholder="user-uuid" autoFocus />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Resource Type</label>
                <select value={createForm.resource_type} onChange={(e) => setCreateForm({ ...createForm, resource_type: e.target.value })} className={inputCls}>
                  {RESOURCE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Resource ID *</label>
                <input value={createForm.resource_id} onChange={(e) => setCreateForm({ ...createForm, resource_id: e.target.value })} className={inputCls} placeholder="resource-uuid or name" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Reason</label>
                <textarea value={createForm.reason} onChange={(e) => setCreateForm({ ...createForm, reason: e.target.value })} className={inputCls} rows={3} placeholder="Why do you need access?" />
              </div>
            </div>
            <div className="mt-6 flex gap-2">
              <button onClick={handleCreate} disabled={creating || !createForm.requester_id || !createForm.resource_id} className={btnPrimary}>
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} Submit Request
              </button>
              <button onClick={() => setShowCreate(false)} className={btnGhost}>Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
