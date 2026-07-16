"use client";
import { useState, useEffect, useCallback } from "react";
import { FileText, Clock, Check, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface Request { id: string; target_role: string; justification: string; duration_days: number; approver: string; status: "pending" | "approved" | "rejected" | "expired"; submitted_at: string; expires_at: string; days_remaining: number; comments: { author: string; text: string; timestamp: string }[]; }
export default function AccessRequestPage() {
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ target_role: "", justification: "", duration_days: 7, approver: "" });
  const [myRequests, setMyRequests] = useState<Request[]>([]);
  const [approvalQueue, setApprovalQueue] = useState<Request[]>([]);
  const [tab, setTab] = useState("my_requests");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const t = useTranslations();

  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const [mine, queue] = await Promise.all([
        fetch("/api/v1/policy/access-request?scope=mine", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`))),
        fetch("/api/v1/policy/access-request?scope=queue", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`))),
      ]);
      setMyRequests(mine.requests || mine || []);
      setApprovalQueue(queue.requests || queue || []);
    } catch (err) { setError(err instanceof Error ? err.message : "An error occurred"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const submitRequest = async () => {
    if (!form.target_role) return;
    try { await fetch("/api/v1/policy/access-request", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowForm(false); setForm({ target_role: "", justification: "", duration_days: 7, approver: "" }); loadData(); }
    catch { /* noop */ }
  };
  const decide = async (id: string, decision: string) => { try { await fetch("/api/v1/policy/access-request/" + id, { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ decision }) }); loadData(); } catch { /* noop */ } };

  if (loading) return (<div className="p-8 flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">{t("accessRequest.error")}: {error}</p><button aria-label="action" onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">{t("accessRequest.retry")}</button></div></div>);

  const list = tab === "my_requests" ? myRequests : approvalQueue;
  const statusColors: Record<string, string> = { pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400", approved: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400", rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400", expired: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400" };
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><FileText className="w-6 h-6 text-blue-500" /> {t("accessRequest.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("accessRequest.subtitle")}</p></div><button onClick={() => setShowForm(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">{t("accessRequest.newRequest")}</button></div>
      <div className="flex gap-2"><button onClick={() => setTab("my_requests")} className={"px-4 py-2 rounded-lg text-sm font-medium " + (tab === "my_requests" ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>{t("accessRequest.myRequests")}</button><button onClick={() => setTab("approvals")} className={"px-4 py-2 rounded-lg text-sm font-medium " + (tab === "approvals" ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>{t("accessRequest.approvalQueue")} {approvalQueue.length > 0 && "(" + approvalQueue.length + ")"}</button></div>
      {showForm && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowForm(false)}><div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("accessRequest.newAccessRequest")}</h3></div><div className="px-6 py-4 space-y-3"><div><label className="text-sm font-medium">{t("accessRequest.targetRole")}</label><input aria-label="role:admin" type="text" value={form.target_role} onChange={(e) => setForm({ ...form, target_role: e.target.value })} placeholder="role:admin" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div><div><label className="text-sm font-medium">{t("accessRequest.justification")}</label><textarea value={form.justification} onChange={(e) => setForm({ ...form, justification: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">{t("accessRequest.durationDays")}</label><input type="number" min={1} value={form.duration_days} onChange={(e) => setForm({ ...form, duration_days: parseInt(e.target.value) })} className="w-20 mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div><div><label className="text-sm font-medium">{t("accessRequest.approver")}</label><input type="text" value={form.approver} onChange={(e) => setForm({ ...form, approver: e.target.value })} placeholder="manager@example.com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div></div><div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowForm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("accessRequest.cancel")}</button><button onClick={submitRequest} disabled={!form.target_role} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50" aria-label="Action">{t("accessRequest.submit")}</button></div></div></div>)}
      <div className="space-y-2">{list.map((r) => (<div key={r.id} className="rounded-lg border dark:border-gray-800 p-3"><div className="flex items-center justify-between"><div><span className="font-medium text-sm">{r.target_role}</span><p className="text-xs text-gray-400">{r.submitted_at} - approver: {r.approver}</p></div><div className="flex items-center gap-2"><span className={"px-2 py-0.5 rounded text-xs " + statusColors[r.status]}>{r.status}</span>{r.status === "pending" && r.days_remaining <= 3 && <span className="flex items-center gap-1 text-xs text-orange-600"><Clock className="w-3 h-3" />{r.days_remaining}d</span>}</div></div><p className="text-sm text-gray-500 mt-1">{r.justification}</p>{r.comments.length > 0 && (<div className="mt-2 space-y-0.5">{r.comments.map((c, i) => (<div key={i} className="text-xs text-gray-400"><span className="font-medium">{c.author}:</span> {c.text}</div>))}</div>)}{tab === "approvals" && r.status === "pending" && <div className="flex gap-2 mt-2"><button onClick={() => decide(r.id, "approved")} className="px-3 py-1 rounded text-xs bg-green-600 text-white flex items-center gap-1"><Check className="w-3 h-3" /> {t("accessRequest.approve")}</button><button onClick={() => decide(r.id, "rejected")} className="px-3 py-1 rounded text-xs bg-red-600 text-white flex items-center gap-1"><X className="w-3 h-3" /> {t("accessRequest.reject")}</button></div>}</div>))}{list.length === 0 && <p className="text-sm text-gray-500 text-center py-8">{t("accessRequest.noRequests")}</p>}</div>
    </div>
  );
}
