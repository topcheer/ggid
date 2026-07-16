"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  ShieldAlert, Loader2, AlertCircle, X, Plus, CheckCircle, Clock, Zap,
} from "lucide-react";

interface Incident {
  id: string;
  title: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  status: "open" | "investigating" | "contained" | "resolved" | "closed";
  description: string;
  affected_users: string[];
  source: string;
  created_at: string;
  updated_at: string;
  resolved_at: string;
  resolution_notes: string;
  assigned_to: string;
}

const sevColors: Record<string, string> = {
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

const statusIcons: Record<string, React.ReactNode> = {
  open: <AlertCircle className="h-4 w-4 text-red-500" />,
  investigating: <Clock className="h-4 w-4 text-yellow-500" />,
  contained: <ShieldAlert className="h-4 w-4 text-orange-500" />,
  resolved: <CheckCircle className="h-4 w-4 text-green-500" />,
  closed: <CheckCircle className="h-4 w-4 text-gray-400" />,
};

export default function IncidentsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [resolving, setResolving] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [resolveIncident, setResolveIncident] = useState<Incident | null>(null);
  const [resolveNotes, setResolveNotes] = useState("");

  const [form, setForm] = useState({ title: "", type: "unauthorized_access", severity: "medium" as Incident["severity"], description: "" });

  useState(() => {
    (async () => {
      try { setIncidents(await apiFetch<Incident[]>("/api/v1/audit/incidents").catch(() => [])); }
      catch { setError(t("incidents.failedLoad")); }
      finally { setLoading(false); }
    })();
  });

  const handleCreate = async () => {
    setCreating(true);
    try {
      const created = await apiFetch<Incident>("/api/v1/audit/incidents", { method: "POST", body: JSON.stringify(form) });
      setIncidents((p) => [created, ...p]);
      setShowCreate(false);
      setForm({ title: "", type: "unauthorized_access", severity: "medium", description: "" });
    } catch { setError(t("incidents.createFailed")); }
    finally { setCreating(false); }
  };

  const handleResolve = async () => {
    if (!resolveIncident || !resolveNotes.trim()) return;
    setResolving(resolveIncident.id);
    try {
      await apiFetch(`/api/v1/audit/incidents/${resolveIncident.id}/resolve`, { method: "POST", body: JSON.stringify({ resolution_notes: resolveNotes }) });
      setIncidents((p) => p.map((i) => i.id === resolveIncident.id ? { ...i, status: "resolved", resolution_notes: resolveNotes, resolved_at: new Date().toISOString() } : i));
      setResolveIncident(null); setResolveNotes("");
    } catch { setError(t("incidents.resolveFailed")); }
    finally { setResolving(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const activeIncidents = incidents.filter((i) => i.status !== "resolved" && i.status !== "closed");
  const stats = { critical: activeIncidents.filter((i) => i.severity === "critical").length, high: activeIncidents.filter((i) => i.severity === "high").length, open: activeIncidents.length, resolved: incidents.filter((i) => i.status === "resolved").length };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldAlert className="h-6 w-6 text-red-600" /> {t("incidents.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("incidents.subtitle")}</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Plus className="h-4 w-4" /> {t("incidents.new")}</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Zap className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Critical</span></div><p className="mt-2 text-2xl font-bold text-red-600">{stats.critical}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><ShieldAlert className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">High</span></div><p className="mt-2 text-2xl font-bold text-orange-600">{stats.high}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{stats.open}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><CheckCircle className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Resolved</span></div><p className="mt-2 text-2xl font-bold text-green-600">{stats.resolved}</p></div>
          </div>

          {/* Incidents table */}
          {incidents.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><ShieldAlert className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("incidents.noIncidents")}</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Title</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Type</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Severity</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Affected</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Created</th>
                  <th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {incidents.map((inc) => (
                    <tr key={inc.id} className="bg-white dark:bg-gray-900">
                      <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{inc.title}</div>{inc.description && <div className="text-xs text-gray-400">{inc.description}</div>}</td>
                      <td className="px-4 py-3 text-gray-500">{inc.type}</td>
                      <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${sevColors[inc.severity] || ""}`}>{inc.severity}</span></td>
                      <td className="px-4 py-3"><div className="flex items-center gap-1.5">{statusIcons[inc.status]}<span className="text-gray-600 dark:text-gray-300">{inc.status}</span></div></td>
                      <td className="px-4 py-3 text-gray-500">{inc.affected_users.length} users</td>
                      <td className="px-4 py-3 text-gray-400">{new Date(inc.created_at).toLocaleString()}</td>
                      <td className="px-4 py-3 text-right">
                        {inc.status !== "resolved" && inc.status !== "closed" && (
                          <button onClick={() => { setResolveIncident(inc); setResolveNotes(""); }} className="text-xs text-green-600 hover:underline">Resolve</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{t("incidents.new")}</h3><button onClick={() => setShowCreate(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Title</label><input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Type</label><select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="unauthorized_access">Unauthorized Access</option><option value="data_breach">Data Breach</option><option value="malware">Malware</option><option value="phishing">Phishing</option><option value="insider_threat">Insider Threat</option><option value="privilege_escalation">Privilege Escalation</option><option value="other">Other</option></select></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Severity</label><select value={form.severity} onChange={(e) => setForm({ ...form, severity: e.target.value as Incident["severity"] })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option><option value="critical">Critical</option></select></div>
              </div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Description</label><textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={3} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <button onClick={handleCreate} disabled={!form.title || creating} className="flex w-full items-center justify-center gap-2 rounded-lg bg-red-600 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : null}{t("incidents.createIncident")}</button>
            </div>
          </div>
        </div>
      )}

      {/* Resolve modal */}
      {resolveIncident && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setResolveIncident(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">Resolve: {resolveIncident.title}</h3><button onClick={() => setResolveIncident(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="mb-4 rounded-lg bg-gray-50 p-3 text-sm dark:bg-gray-900"><span className="text-gray-400">Severity:</span> <span className={`font-medium ${resolveIncident.severity === "critical" ? "text-red-600" : resolveIncident.severity === "high" ? "text-orange-600" : "text-gray-600"}`}>{resolveIncident.severity}</span></div>
            <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Resolution Notes</label><textarea value={resolveNotes} onChange={(e) => setResolveNotes(e.target.value)} rows={4} placeholder="Describe the investigation, root cause, and actions taken..." className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
            <button onClick={handleResolve} disabled={!resolveNotes.trim() || resolving !== null} className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-green-600 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{resolving ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}{t("incidents.resolveIncident")}</button>
          </div>
        </div>
      )}
    </div>
  );
}
