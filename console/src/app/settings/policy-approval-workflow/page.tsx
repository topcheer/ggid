"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { CheckCircle, XCircle, Clock, ChevronRight } from "lucide-react";

interface Approval {
  id: string;
  policy_name: string;
  requested_by: string;
  risk_level: "low" | "medium" | "high" | "critical";
  submitted_at: string;
  expires_at: string;
  days_remaining: number;
  change_summary: string;
  approval_chain: { approver: string; status: "pending" | "approved" | "rejected"; acted_at: string | null }[];
  comments: { author: string; text: string; timestamp: string }[];
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function PolicyApprovalWorkflowPage() {
  const t = useTranslations();
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [comment, setComment] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/policy/approval-workflow", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setApprovals(d.approvals || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const decide = async (id: string, decision: "approved" | "rejected") => {
    try { await fetch("/api/v1/policy/approval-workflow/" + id, { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ decision, comment }) }); setComment(""); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><CheckCircle className="w-6 h-6 text-green-500" />{t("policyApprovalWorkflow.title")}</h1><p className="text-sm text-gray-500 mt-1">Review and approve pending policy changes with approval chains.</p></div>

      <div className="space-y-3">
        {approvals.map((a) => (
          <div key={a.id} className="rounded-lg border dark:border-gray-800">
            <div className="flex items-center justify-between p-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30" onClick={() => setExpanded(expanded === a.id ? null : a.id)}>
              <div className="flex items-center gap-2"><ChevronRight className={"w-4 h-4 text-gray-400 transition-transform " + (expanded === a.id ? "rotate-90" : "")} /><div><span className="font-medium text-sm">{a.policy_name}</span><p className="text-xs text-gray-400">by {a.requested_by} - {a.submitted_at}</p></div></div>
              <div className="flex items-center gap-2"><span className={"px-2 py-0.5 rounded text-xs " + riskColors[a.risk_level]}>{a.risk_level}</span>{a.days_remaining <= 3 && <span className="flex items-center gap-1 text-xs text-orange-600"><Clock className="w-3 h-3" /> {a.days_remaining}d</span>}</div>
            </div>
            <div className="px-3 pb-2 text-sm text-gray-500">{a.change_summary}</div>
            {expanded === a.id && (
              <div className="border-t dark:border-gray-800 p-3 bg-gray-50 dark:bg-gray-900/30 space-y-3">
                <div><h4 className="text-xs font-semibold text-gray-500 mb-2">Approval Chain</h4><div className="flex items-center gap-2">{a.approval_chain.map((step, i) => (<div key={i} className="flex items-center gap-1"><span className={"text-xs px-2 py-0.5 rounded " + (step.status === "approved" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : step.status === "rejected" ? "bg-red-100 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 dark:bg-gray-800")}>{step.approver}</span>{i < a.approval_chain.length - 1 && <span className="text-gray-300">{"->"}</span>}</div>))}</div></div>
                {a.comments.length > 0 && (<div><h4 className="text-xs font-semibold text-gray-500 mb-1">Comments</h4><div className="space-y-1">{a.comments.map((c, i) => (<div key={i} className="text-xs text-gray-500"><span className="font-medium">{c.author}:</span> {c.text} <span className="text-gray-400">({c.timestamp})</span></div>))}</div></div>)}
                <div className="flex items-center gap-2"><input type="text" value={comment} onChange={(e) => setComment(e.target.value)} placeholder="Add comment..." className="flex-1 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /><button onClick={() => decide(a.id, "approved")} className="px-3 py-1 rounded text-xs bg-green-600 text-white flex items-center gap-1"><CheckCircle className="w-3 h-3" /> Approve</button><button onClick={() => decide(a.id, "rejected")} className="px-3 py-1 rounded text-xs bg-red-600 text-white flex items-center gap-1"><XCircle className="w-3 h-3" /> Reject</button></div>
              </div>
            )}
          </div>
        ))}
        {approvals.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No pending approvals.</p>}
      </div>
    </div>
  );
}
