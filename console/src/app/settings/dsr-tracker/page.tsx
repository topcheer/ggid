"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { FileText, Plus, Play, X, Clock, AlertTriangle, CheckCircle2 } from "lucide-react";

interface DSRRequest {
  id: string;
  type: "access" | "erasure" | "portability" | "rectification";
  user_id: string;
  username: string;
  email: string;
  status: "pending" | "in_progress" | "completed" | "overdue";
  created_at: string;
  due_date: string;
  days_remaining: number;
  notes: string;
}

const typeColors: Record<string, string> = {
  access: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  erasure: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  portability: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
  rectification: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
};

const statusColors: Record<string, string> = {
  pending: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  in_progress: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  completed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  overdue: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function DSRTrackerPage() {
  const t = useTranslations();
  const [requests, setRequests] = useState<DSRRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [newType, setNewType] = useState<DSRRequest["type"]>("access");
  const [newUserId, setNewUserId] = useState("");
  const [newNotes, setNewNotes] = useState("");
  const [processingId, setProcessingId] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/dsr", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setRequests(data.requests || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchRequests(); }, [fetchRequests]);

  const createRequest = async () => {
    if (!newUserId) return;
    try {
      await fetch("/api/v1/audit/dsr", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ type: newType, user_id: newUserId, notes: newNotes }),
      });
      setShowCreate(false);
      setNewUserId("");
      setNewNotes("");
      fetchRequests();
    } catch { /* noop */ }
  };

  const processRequest = async (id: string, status: DSRRequest["status"]) => {
    setProcessingId(id);
    try {
      await fetch(`/api/v1/audit/dsr/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ status }),
      });
      setRequests((prev) => prev.map((r) => r.id === id ? { ...r, status } : r));
    } catch { /* noop */ }
    finally { setProcessingId(null); }
  };

  const overdue = requests.filter((r) => r.status === "overdue" || (r.status !== "completed" && r.days_remaining <= 0));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><FileText className="w-6 h-6 text-blue-500" />{t("dsrTracker.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">GDPR Data Subject Request tracking with SLA countdown.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> New Request</button>
      </div>

      {/* Overdue alert */}
      {overdue.length > 0 && (
        <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2">
          <AlertTriangle className="w-5 h-5 text-red-500" />
          <span className="font-semibold text-red-700 dark:text-red-400">{overdue.length} DSR request{overdue.length > 1 ? "s" : ""} overdue</span>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total</span><p className="text-2xl font-bold mt-1">{requests.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Pending</span><p className="text-2xl font-bold mt-1 text-yellow-600">{requests.filter((r) => r.status === "pending").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Completed</span><p className="text-2xl font-bold mt-1 text-green-600">{requests.filter((r) => r.status === "completed").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Overdue</span><p className="text-2xl font-bold mt-1 text-red-600">{overdue.length}</p></div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">Type</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Created</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Due Date</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">SLA</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {requests.map((r) => (
              <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${typeColors[r.type]}`}>{r.type}</span></td>
                <td className="px-4 py-3"><span className="font-medium">{r.username}</span><p className="text-xs text-gray-400">{r.email}</p></td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[r.status]}`}>{r.status.replace("_", " ")}</span></td>
                <td className="px-4 py-3 text-gray-500">{r.created_at}</td>
                <td className="px-4 py-3 text-gray-500">{r.due_date}</td>
                <td className="px-4 py-3">
                  {r.status === "completed" ? <span className="text-green-600 text-xs flex items-center gap-1"><CheckCircle2 className="w-3 h-3" /> Done</span> :
                   r.days_remaining <= 0 ? <span className="text-red-600 text-xs flex items-center gap-1"><AlertTriangle className="w-3 h-3" /> Overdue</span> :
                   <span className={`text-xs flex items-center gap-1 ${r.days_remaining <= 3 ? "text-red-600" : r.days_remaining <= 7 ? "text-orange-600" : "text-gray-500"}`}><Clock className="w-3 h-3" /> {r.days_remaining}d left</span>}
                </td>
                <td className="px-4 py-3">
                  {r.status === "pending" && <button onClick={() => processRequest(r.id, "in_progress")} disabled={processingId === r.id} className="text-xs font-medium text-blue-600 hover:underline disabled:opacity-50">Start</button>}
                  {r.status === "in_progress" && <button onClick={() => processRequest(r.id, "completed")} disabled={processingId === r.id} className="text-xs font-medium text-green-600 hover:underline disabled:opacity-50 flex items-center gap-1"><Play className="w-3 h-3" /> Complete</button>}
                </td>
              </tr>
            ))}
            {requests.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No DSR requests.</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold">New DSR Request</h3>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div>
                <label className="text-sm font-medium">Type</label>
                <select aria-label="new Type" value={newType} onChange={(e) => setNewType(e.target.value as DSRRequest["type"])} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">
                  <option value="access">Access</option>
                  <option value="erasure">Erasure</option>
                  <option value="portability">Portability</option>
                  <option value="rectification">Rectification</option>
                </select>
              </div>
              <div><label className="text-sm font-medium">User ID</label><input aria-label="user-uuid" type="text" value={newUserId} onChange={(e) => setNewUserId(e.target.value)} placeholder="user-uuid" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Notes</label><textarea aria-label="Text input" value={newNotes} onChange={(e) => setNewNotes(e.target.value)} rows={3} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={createRequest} disabled={!newUserId} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
