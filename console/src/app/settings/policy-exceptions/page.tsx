"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { ShieldOff, Plus, X, ChevronRight, Clock } from "lucide-react";

interface Exception {
  id: string;
  policy_id: string;
  policy_name: string;
  reason: string;
  granted_to: string;
  approver: string;
  risk_override_level: "low" | "medium" | "high" | "critical";
  created_at: string;
  expires_at: string;
  days_remaining: number;
  audit_trail: { timestamp: string; action: string; actor: string; detail: string }[];
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function PolicyExceptionsPage() {
  const t = useTranslations();
  const [exceptions, setExceptions] = useState<Exception[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [form, setForm] = useState({ policy_id: "", reason: "", granted_to: "", risk_override_level: "low", expires_at: "" });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/exceptions", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setExceptions(d.exceptions || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const create = async () => {
    if (!form.policy_id || !form.granted_to) return;
    try { await fetch("/api/v1/policy/exceptions", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowCreate(false); setForm({ policy_id: "", reason: "", granted_to: "", risk_override_level: "low", expires_at: "" }); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ShieldOff className="w-6 h-6 text-orange-500" />{t("policyExceptions.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage time-limited policy exceptions with risk overrides and audit trails.</p></div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 flex items-center gap-2"><Plus className="w-4 h-4" /> New Exception</button>
      </div>

      <div className="space-y-3">{exceptions.map((e) => (
        <div key={e.id} className="rounded-lg border dark:border-gray-800">
          <div className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30" onClick={() => setExpanded(expanded === e.id ? null : e.id)}>
            <div className="flex items-center gap-3"><ChevronRight className={`w-4 h-4 text-gray-400 transition-transform ${expanded === e.id ? "rotate-90" : ""}`} /><div><span className="font-medium">{e.policy_name}</span><p className="text-xs text-gray-400 font-mono">{e.policy_id} - granted to: {e.granted_to}</p></div></div>
            <div className="flex items-center gap-2"><span className={`px-2 py-0.5 rounded text-xs ${riskColors[e.risk_override_level]}`}>{e.risk_override_level}</span>{e.days_remaining >= 0 && <span className="flex items-center gap-1 text-xs text-orange-600"><Clock className="w-3 h-3" /> {e.days_remaining}d left</span>}</div>
          </div>
          <div className="px-4 pb-2 text-sm text-gray-500"><span>Reason: </span>{e.reason}<span className="ml-4 text-xs">Approved by: {e.approver}</span></div>
          {expanded === e.id && e.audit_trail.length > 0 && (
            <div className="border-t dark:border-gray-800 px-4 py-3 bg-gray-50 dark:bg-gray-900/30"><h4 className="text-xs font-semibold mb-2 text-gray-500">Audit Trail</h4><div className="space-y-1">{e.audit_trail.map((t, i) => (<div key={i} className="flex items-center gap-2 text-xs"><span className="text-gray-400">{t.timestamp}</span><span className="font-mono text-gray-500">{t.action}</span><span>{t.detail}</span><span className="text-gray-400">by {t.actor}</span></div>))}</div></div>
          )}
        </div>
      ))}{exceptions.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No policy exceptions.</p>}</div>

      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">New Policy Exception</h3><button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Policy ID</label><input aria-label="pol-xxxx" type="text" value={form.policy_id} onChange={(e) => setForm({ ...form, policy_id: e.target.value })} placeholder="pol-xxxx" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Granted To</label><input aria-label="user:alice" type="text" value={form.granted_to} onChange={(e) => setForm({ ...form, granted_to: e.target.value })} placeholder="user:alice" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Reason</label><input aria-label="Temporary access for project X" type="text" value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} placeholder="Temporary access for project X" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Risk Override Level</label><select aria-label="form" value={form.risk_override_level} onChange={(e) => setForm({ ...form, risk_override_level: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option><option value="critical">Critical</option></select></div>
              <div><label className="text-sm font-medium">Expires At</label><input aria-label="form" type="datetime-local" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={create} disabled={!form.policy_id || !form.granted_to} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 disabled:opacity-50">Create</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
