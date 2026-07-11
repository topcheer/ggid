"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ClipboardCheck, Check, X, Loader2, AlertCircle, Clock,
  User, Shield, CheckCircle2, History,
} from "lucide-react";

interface AccessReviewItem {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  manager_id: string;
  manager_name?: string;
  roles: string[];
  status: "pending" | "approved" | "revoked";
  created_at: string;
  reviewed_at?: string;
  reviewer?: string;
  decision?: string;
  comment?: string;
}

interface ReviewSummary {
  pending: number;
  approved: number;
  revoked: number;
  overdue: number;
}

export default function AccessReviewsPage() {
  const { apiFetch } = useApi();
  const [reviews, setReviews] = useState<AccessReviewItem[]>([]);
  const [history, setHistory] = useState<AccessReviewItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<"pending" | "history">("pending");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [comment, setComment] = useState<Record<string, string>>({});

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [pendingRes, historyRes] = await Promise.all([
        apiFetch<{ reviews?: AccessReviewItem[]; items?: AccessReviewItem[] }>("/api/v1/audit/access-reviews?status=pending").catch(() => ({ reviews: [] as AccessReviewItem[], items: [] as AccessReviewItem[] })),
        apiFetch<{ reviews?: AccessReviewItem[]; items?: AccessReviewItem[] }>("/api/v1/audit/access-reviews?status=completed&limit=20").catch(() => ({ reviews: [] as AccessReviewItem[], items: [] as AccessReviewItem[] })),
      ]);
      setReviews(pendingRes.reviews ?? pendingRes.items ?? []);
      setHistory(historyRes.reviews ?? historyRes.items ?? []);
    } catch {
      setError("Failed to load access reviews");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleDecision = async (id: string, decision: "approve" | "revoke") => {
    setActionLoading(id);
    try {
      await apiFetch(`/api/v1/audit/access-reviews/${id}/decision`, {
        method: "POST",
        body: JSON.stringify({ decision, comment: comment[id] ?? "" }),
      });
      setComment((p) => { const n = { ...p }; delete n[id]; return n; });
      await load();
    } catch {
      setError(`Failed to ${decision} review`);
    } finally {
      setActionLoading(null);
    }
  };

  const handleBulkDecision = async (decision: "approve" | "revoke") => {
    setActionLoading("bulk");
    try {
      await Promise.all(
        Array.from(selected).map((id) =>
          apiFetch(`/api/v1/audit/access-reviews/${id}/decision`, {
            method: "POST",
            body: JSON.stringify({ decision }),
          })
        )
      );
      setSelected(new Set());
      await load();
    } catch {
      setError("Some reviews failed to process");
    } finally {
      setActionLoading(null);
    }
  };

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selected.size === reviews.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(reviews.map((r) => r.id)));
    }
  };

  const summary: ReviewSummary = {
    pending: reviews.length,
    approved: history.filter((r) => r.status === "approved").length,
    revoked: history.filter((r) => r.status === "revoked").length,
    overdue: reviews.filter((r) => {
      const age = Date.now() - new Date(r.created_at).getTime();
      return age > 14 * 24 * 60 * 60 * 1000; // 14 days
    }).length,
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <ClipboardCheck className="h-6 w-6 text-indigo-600" /> Access Reviews
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Periodic access certification — review and approve user permissions.
        </p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">Pending</p>
          <p className="mt-1 text-2xl font-bold text-yellow-600">{summary.pending}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">Overdue</p>
          <p className="mt-1 text-2xl font-bold text-red-600">{summary.overdue}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">Approved (30d)</p>
          <p className="mt-1 text-2xl font-bold text-green-600">{summary.approved}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">Revoked (30d)</p>
          <p className="mt-1 text-2xl font-bold text-gray-500">{summary.revoked}</p>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        <button
          onClick={() => setTab("pending")}
          className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "pending" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400 hover:text-gray-600"}`}
        >
          <Clock className="h-4 w-4" /> Pending ({reviews.length})
        </button>
        <button
          onClick={() => setTab("history")}
          className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "history" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400 hover:text-gray-600"}`}
        >
          <History className="h-4 w-4" /> History ({history.length})
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : tab === "pending" ? (
        <>
          {/* Bulk actions */}
          {selected.size > 0 && (
            <div className="flex items-center gap-3 rounded-lg bg-indigo-50 px-4 py-3 dark:bg-indigo-900/20">
              <span className="text-sm font-medium text-indigo-700 dark:text-indigo-400">{selected.size} selected</span>
              <button onClick={() => handleBulkDecision("approve")} disabled={actionLoading === "bulk"} className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700">
                <Check className="h-3.5 w-3.5" /> Approve All
              </button>
              <button onClick={() => handleBulkDecision("revoke")} disabled={actionLoading === "bulk"} className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">
                <X className="h-3.5 w-3.5" /> Revoke All
              </button>
              {actionLoading === "bulk" && <Loader2 className="h-4 w-4 animate-spin text-indigo-600" />}
              <button onClick={() => setSelected(new Set())} className="ml-auto text-xs text-gray-500">Clear</button>
            </div>
          )}

          {reviews.length === 0 ? (
            <div className={cardCls}>
              <div className="py-12 text-center">
                <CheckCircle2 className="mx-auto h-12 w-12 text-green-300" />
                <p className="mt-4 text-sm text-gray-400">All caught up! No pending access reviews.</p>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {/* Select all */}
              <label className="flex items-center gap-2 text-sm text-gray-500">
                <input type="checkbox" checked={selected.size === reviews.length && reviews.length > 0} onChange={toggleSelectAll} className="rounded border-gray-300 text-indigo-600" />
                Select all
              </label>

              {reviews.map((r) => {
                const isOverdue = Date.now() - new Date(r.created_at).getTime() > 14 * 24 * 60 * 60 * 1000;
                return (
                  <div key={r.id} className={`${cardCls} ${isOverdue ? "border-red-200 dark:border-red-800" : ""}`}>
                    <div className="flex items-start gap-3">
                      <input type="checkbox" checked={selected.has(r.id)} onChange={() => toggleSelect(r.id)} className="mt-1 rounded border-gray-300 text-indigo-600" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <User className="h-4 w-4 text-gray-400" />
                          <span className="font-medium text-gray-800 dark:text-gray-200">{r.user_name}</span>
                          <span className="text-xs text-gray-400">{r.user_email}</span>
                          {isOverdue && <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400">Overdue</span>}
                        </div>
                        <div className="mt-2 flex flex-wrap gap-1">
                          {r.roles.map((role) => (
                            <span key={role} className="flex items-center gap-1 rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">
                              <Shield className="h-3 w-3" />{role}
                            </span>
                          ))}
                        </div>
                        <div className="mt-2 flex items-center gap-3 text-xs text-gray-400">
                          <span>Requested: {new Date(r.created_at).toLocaleDateString()}</span>
                          {r.manager_name && <span>Manager: {r.manager_name}</span>}
                        </div>
                        {/* Comment input */}
                        <input
                          value={comment[r.id] ?? ""}
                          onChange={(e) => setComment((p) => ({ ...p, [r.id]: e.target.value }))}
                          placeholder="Add a comment (optional)..."
                          className="mt-2 w-full max-w-xs rounded-lg border border-gray-200 px-3 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                        />
                      </div>
                      {/* Decision buttons */}
                      <div className="flex flex-col gap-1.5">
                        <button
                          onClick={() => handleDecision(r.id, "approve")}
                          disabled={actionLoading === r.id}
                          className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700 disabled:opacity-50"
                        >
                          {actionLoading === r.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
                          Approve
                        </button>
                        <button
                          onClick={() => handleDecision(r.id, "revoke")}
                          disabled={actionLoading === r.id}
                          className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50"
                        >
                          <X className="h-3.5 w-3.5" /> Revoke
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      ) : (
        /* History tab */
        history.length === 0 ? (
          <div className={cardCls}>
            <div className="py-12 text-center">
              <History className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-4 text-sm text-gray-400">No completed reviews yet.</p>
            </div>
          </div>
        ) : (
          <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr className="text-left text-xs font-semibold uppercase text-gray-500">
                  <th className="px-4 py-3">User</th>
                  <th className="px-4 py-3">Roles</th>
                  <th className="px-4 py-3">Decision</th>
                  <th className="px-4 py-3">Reviewer</th>
                  <th className="px-4 py-3">Reviewed</th>
                  <th className="px-4 py-3">Comment</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {history.map((r) => (
                  <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3">
                      <div className="font-medium text-gray-800 dark:text-gray-200">{r.user_name}</div>
                      <div className="text-xs text-gray-400">{r.user_email}</div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {r.roles.map((role) => <span key={role} className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{role}</span>)}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${r.status === "approved" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"}`}>
                        {r.status === "approved" ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
                        {r.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-500">{r.reviewer ?? "—"}</td>
                    <td className="px-4 py-3 text-gray-500">{r.reviewed_at ? new Date(r.reviewed_at).toLocaleDateString() : "—"}</td>
                    <td className="px-4 py-3 text-xs text-gray-400">{r.comment ?? "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}
    </div>
  );
}
