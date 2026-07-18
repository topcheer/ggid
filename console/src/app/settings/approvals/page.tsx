"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  ClipboardCheck, Loader2, AlertCircle, X, CheckCircle, XCircle, ChevronRight, Clock,
} from "lucide-react";

interface ApprovalStep {
  step: number;
  name: string;
  approver: string;
  status: "pending" | "approved" | "rejected" | "skipped";
  acted_at: string;
  comment: string;
}

interface ApprovalRequest {
  id: string;
  request_type: string;
  requester: string;
  requester_name: string;
  description: string;
  current_step: number;
  total_steps: number;
  approver_chain: ApprovalStep[];
  status: "pending" | "approved" | "rejected" | "expired" | "cancelled";
  created_at: string;
  expires_at: string;
}

const statusIcons: Record<string, React.ReactNode> = {
  approved: <CheckCircle className="h-4 w-4 text-green-500" />,
  rejected: <XCircle className="h-4 w-4 text-red-500" />,
  pending: <Clock className="h-4 w-4 text-yellow-500" />,
  skipped: <ChevronRight className="h-4 w-4 text-gray-400" />,
};

export default function ApprovalsPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [pending, setPending] = useState<ApprovalRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actioning, setActioning] = useState<string | null>(null);
  const [selectedReq, setSelectedReq] = useState<ApprovalRequest | null>(null);
  const [comment, setComment] = useState("");

  useEffect(() => {
    (async () => {
      try { setPending(await apiFetch<ApprovalRequest[]>("/api/v1/policy/approvals?status=pending").catch(() => [])); }
      catch { setError("Failed to load approvals"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleAction = async (req: ApprovalRequest, action: "approve" | "reject") => {
    setActioning(req.id);
    try {
      await apiFetch(`/api/v1/policy/approvals/${req.id}/${action}`, { method: "POST", body: JSON.stringify({ comment }) });
      setPending((p) => p.filter((r: any) => r.id !== req.id));
      setSelectedReq(null); setComment("");
    } catch { setError(`${action === "approve" ? "Approve" : "Reject"} failed`); }
    finally { setActioning(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ClipboardCheck className="h-6 w-6 text-blue-600" /> {t("approvals.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("approvals.subtitle")}</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-600" /></div>
      : pending.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><ClipboardCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("approvals.noPending")}</p></div></div>
      ) : (
        <div className="space-y-3">
          {pending.map((req: any) => (
            <div key={req.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{req.request_type}</span>
                    <span className="font-medium text-gray-900 dark:text-white">{req.requester_name || req.requester.slice(0, 12)}</span>
                    <span className="text-xs text-gray-400">{t("approvals.step")} {req.current_step} {t("approvals.of")} {req.total_steps}</span>
                  </div>
                  {req.description && <p className="mt-1 text-sm text-gray-500">{req.description}</p>}
                  {/* Approver chain */}
                  <div className="mt-2 flex items-center gap-1">
                    {req.approver_chain.map((s: any, i: number) => (
                      <React.Fragment key={i}>
                        <div className="flex items-center gap-1 rounded px-2 py-0.5 text-xs">
                          {statusIcons[s.status]}
                          <span className={`font-medium ${s.status === "pending" ? "text-yellow-600" : s.status === "approved" ? "text-green-600" : s.status === "rejected" ? "text-red-600" : "text-gray-400"}`}>{s.name}</span>
                        </div>
                        {i < req.approver_chain.length - 1 && <ChevronRight className="h-3 w-3 text-gray-300" />}
                      </React.Fragment>
                    ))}
                  </div>
                </div>
                <button onClick={() => { setSelectedReq(req); setComment(""); }} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">{t("approvals.review")}</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Action modal */}
      {selectedReq && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setSelectedReq(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">Review: {selectedReq.request_type}</h3><button onClick={() => setSelectedReq(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="mb-4 rounded-lg bg-gray-50 p-3 text-sm dark:bg-gray-900"><div className="text-gray-400">{t("approvals.requester")}</div><div className="font-medium text-gray-900 dark:text-white">{selectedReq.requester_name || selectedReq.requester.slice(0, 12)}</div>{selectedReq.description && <p className="mt-1 text-gray-500">{selectedReq.description}</p>}</div>
            <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("approvals.comment")}</label><textarea aria-label="Text input" value={comment} onChange={(e) => setComment(e.target.value)} rows={3} placeholder={t("approvals.commentPlaceholder")} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
            <div className="mt-4 flex gap-3">
              <button onClick={() => handleAction(selectedReq, "approve")} disabled={actioning === selectedReq.id} className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-green-600 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{actioning === selectedReq.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}{t("approvals.approve")}</button>
              <button onClick={() => handleAction(selectedReq, "reject")} disabled={actioning === selectedReq.id} className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-red-600 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{actioning === selectedReq.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />}{t("approvals.reject")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
