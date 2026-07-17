"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Check, CheckCircle,
  XCircle, Clock, FileText, GitBranch, Users, Download, Search,
  ChevronRight, ArrowRight, Bell, Eye, AlertTriangle, Code,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ConsentRecord {
  id: string;
  user_id: string;
  username: string;
  client_id: string;
  client_name: string;
  scopes: string[];
  version: string;
  granted_at: string;
  revoked_at: string | null;
  status: "active" | "revoked" | "expired";
  authorization_details?: { type: string; actions: string[]; resources: string[] }[];
}

interface ConsentVersion {
  id: string;
  version: string;
  text: string;
  effective_date: string;
  affected_users: number;
  reconsent_required: boolean;
  previous_version: string;
}

interface ComplianceReport {
  total_users: number;
  consented: number;
  pending: number;
  revoked: number;
  compliance_rate_pct: number;
  gdpr_rate_pct: number;
  ccpa_rate_pct: number;
}

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  revoked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  expired: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
};

type Tab = "history" | "versions" | "revoke" | "rar" | "report";

export default function ConsentManagementPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("history");
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [versions, setVersions] = useState<ConsentVersion[]>([]);
  const [report, setReport] = useState<ComplianceReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Filters
  const [searchUser, setSearchUser] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  // Actions
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [selectedVersion, setSelectedVersion] = useState("");
  // RAR
  const [rarInput, setRarInput] = useState('[{"type":"payment_initiation","actions":["read","execute"],"resources":["account:12345"]}]');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [cRes, vRes, rRes] = await Promise.all([
        fetch("/api/v1/oauth/consents?page_size=100", { headers: h }).catch(() => null),
        fetch("/api/v1/oauth/consent-versions", { headers: h }).catch(() => null),
        fetch("/api/v1/oauth/consent/report", { headers: h }).catch(() => null),
      ]);
      if (cRes?.ok) { const d = await cRes.json(); setConsents(d.consents || d.items || []); }
      if (vRes?.ok) { const d = await vRes.json(); setVersions(d.versions || d.items || []); }
      if (rRes?.ok) setReport(await rRes.json());
    } catch { setError("Failed to load consent data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const revokeConsent = async (id: string) => {
    setRevokingId(id);
    try {
      await fetch(`/api/v1/oauth/consents/${id}/revoke`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setConsents(prev => prev.map(c => c.id === id ? { ...c, status: "revoked", revoked_at: new Date().toISOString() } : c));
    } catch { setError("Failed to revoke consent"); }
    finally { setRevokingId(null); }
  };

  const notifyReconsent = async (versionId: string) => {
    try {
      await fetch(`/api/v1/oauth/consent-versions/${versionId}/notify`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setError("Re-consent notifications sent to affected users.");
    } catch { setError("Failed to send notifications"); }
  };

  const parseRAR = (): { type: string; actions: string[]; resources: string[] }[] => {
    try { return JSON.parse(rarInput); } catch { return []; }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filteredConsents = consents.filter(c => {
    if (statusFilter !== "all" && c.status !== statusFilter) return false;
    if (searchUser && !c.username?.includes(searchUser) && !c.user_id.includes(searchUser)) return false;
    return true;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-purple-500" /> Consent Management</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">GDPR/CCPA consent lifecycle — history, versioning, revocation, and compliance reporting.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "history" as Tab, label: "Consent History", icon: Clock },
          { id: "versions" as Tab, label: "Version Management", icon: GitBranch },
          { id: "revoke" as Tab, label: "Revoke", icon: XCircle },
          { id: "rar" as Tab, label: "RAR Details", icon: Code },
          { id: "report" as Tab, label: "Compliance Report", icon: FileText },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-purple-600 text-purple-600 dark:text-purple-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-500" /></div> : (<>

      {/* CONSENT HISTORY */}
      {tab === "history" && (
        <div className={cardCls}>
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <div className="relative flex-1 min-w-[200px]"><Search className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" /><input aria-label="Search user" type="text" value={searchUser} onChange={e => setSearchUser(e.target.value)} placeholder="Search user..." className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" /></div>
            <select aria-label="Filter status" value={statusFilter} onChange={e => setStatusFilter(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm"><option value="all">All Status</option><option value="active">Active</option><option value="revoked">Revoked</option><option value="expired">Expired</option></select>
            <span className="text-xs text-gray-400">{filteredConsents.length} records</span>
          </div>
          {filteredConsents.length === 0 ? <div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No consent records.</p></div> : (
            <div className="overflow-x-auto max-h-[500px] overflow-y-auto"><table className="w-full text-sm">
              <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80"><tr><th scope="col" className="px-3 py-2 text-left font-medium text-xs">User</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Client</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Scopes</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Version</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Status</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Granted</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Revoked</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{filteredConsents.map(c => (
                <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-3 py-2 text-xs"><span className="font-medium">{c.username || c.user_id}</span></td>
                  <td className="px-3 py-2 text-xs">{c.client_name || c.client_id}</td>
                  <td className="px-3 py-2"><div className="flex flex-wrap gap-0.5">{c.scopes?.map(s => <span key={s} className="px-1 py-0.5 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 text-xs font-mono">{s}</span>)}</div></td>
                  <td className="px-3 py-2 text-center text-xs font-mono">{c.version}</td>
                  <td className="px-3 py-2 text-center"><span className={"px-1.5 py-0.5 rounded text-xs font-medium " + (statusColors[c.status] || "")}>{c.status}</span></td>
                  <td className="px-3 py-2 text-xs text-gray-500">{c.granted_at ? new Date(c.granted_at).toLocaleDateString() : "—"}</td>
                  <td className="px-3 py-2 text-xs text-gray-500">{c.revoked_at ? new Date(c.revoked_at).toLocaleDateString() : "—"}</td>
                </tr>
              ))}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* VERSION MANAGEMENT */}
      {tab === "versions" && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">{versions.length === 0 ? (
            <div className={cardCls + " sm:col-span-2"}><div className="py-8 text-center"><GitBranch className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No consent versions defined.</p></div></div>
          ) : versions.map(v => (
            <div key={v.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div><div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">v{v.version}</span>{v.reconsent_required && <span className="px-1.5 py-0.5 rounded text-xs bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400">Re-consent required</span>}</div><p className="text-xs text-gray-400 mt-0.5">Effective: {new Date(v.effective_date).toLocaleDateString()} · Prev: v{v.previous_version}</p></div>
                <span className="text-xs text-gray-400">{v.affected_users} users</span>
              </div>
              <div className="mt-3 rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><p className="text-xs text-gray-600 dark:text-gray-400 whitespace-pre-wrap line-clamp-3">{v.text}</p></div>
              {v.reconsent_required && <button onClick={() => notifyReconsent(v.id)} className="mt-3 flex items-center gap-1 rounded-lg bg-orange-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-orange-700"><Bell className="h-3 w-3" /> Notify {v.affected_users} Users</button>}
            </div>
          ))}</div>
        </div>
      )}

      {/* REVOKE */}
      {tab === "revoke" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><XCircle className="h-4 w-4" /> Admin Consent Revocation</h2>
          <p className="text-sm text-gray-500 mb-4">Revoke active consents — tokens are immediately invalidated via token revocation.</p>
          {consents.filter(c => c.status === "active").length === 0 ? <div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No active consents to revoke.</p></div> : (
            <div className="space-y-2">{consents.filter(c => c.status === "active").slice(0, 20).map(c => (
              <div key={c.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3"><div className="flex h-8 w-8 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><CheckCircle className="h-4 w-4 text-green-500" /></div><div><p className="text-sm font-medium">{c.username || c.user_id}</p><p className="text-xs text-gray-400">{c.client_name} · {c.scopes?.join(", ")}</p></div></div>
                <button onClick={() => revokeConsent(c.id)} disabled={revokingId === c.id} className="flex items-center gap-1 rounded-lg bg-red-50 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-100 dark:bg-red-950/20 disabled:opacity-50">{revokingId === c.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <XCircle className="h-3 w-3" />} Revoke</button>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* RAR DETAILS */}
      {tab === "rar" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> Authorization Details (JSON)</h2>
            <textarea aria-label="RAR JSON" value={rarInput} onChange={e => setRarInput(e.target.value)} rows={8} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            <p className="mt-2 text-xs text-gray-400">Rich Authorization Requests — parsed into human-readable consent text.</p>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Human-Readable Preview</h2>
            <div className="space-y-3">
              {parseRAR().map((detail, i) => (
                <div key={i} className="rounded-lg border p-3 dark:border-gray-700">
                  <p className="font-semibold text-sm text-gray-900 dark:text-white">{detail.type.replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase())}</p>
                  <div className="mt-2 space-y-1">
                    <p className="text-xs text-gray-500">This application requests to:</p>
                    {detail.actions?.map(a => <div key={a} className="flex items-center gap-1 text-xs"><ChevronRight className="h-3 w-3 text-purple-400" /><span className="font-mono">{a}</span> resources matching <span className="font-mono">{detail.resources?.join(", ") || "all"}</span></div>)}
                  </div>
                </div>
              ))}
              {parseRAR().length === 0 && <p className="text-sm text-gray-400">Invalid JSON — fix input to preview.</p>}
            </div>
          </div>
        </div>
      )}

      {/* COMPLIANCE REPORT */}
      {tab === "report" && (
        <div className="space-y-4">
          {report ? (
            <>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                <div className={cardCls + " text-center"}><Users className="h-6 w-6 mx-auto text-gray-400" /><p className="mt-2 text-2xl font-bold">{report.total_users}</p><p className="text-xs text-gray-400">Total Users</p></div>
                <div className={cardCls + " text-center"}><CheckCircle className="h-6 w-6 mx-auto text-green-500" /><p className="mt-2 text-2xl font-bold text-green-600">{report.consented}</p><p className="text-xs text-gray-400">Consented</p></div>
                <div className={cardCls + " text-center"}><Clock className="h-6 w-6 mx-auto text-yellow-500" /><p className="mt-2 text-2xl font-bold text-yellow-600">{report.pending}</p><p className="text-xs text-gray-400">Pending</p></div>
                <div className={cardCls + " text-center"}><XCircle className="h-6 w-6 mx-auto text-red-500" /><p className="mt-2 text-2xl font-bold text-red-600">{report.revoked}</p><p className="text-xs text-gray-400">Revoked</p></div>
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                {[{ label: "Overall Compliance", pct: report.compliance_rate_pct }, { label: "GDPR", pct: report.gdpr_rate_pct }, { label: "CCPA", pct: report.ccpa_rate_pct }].map(m => (
                  <div key={m.label} className={cardCls}><div className="flex items-center justify-between"><span className="text-sm font-medium">{m.label}</span><span className={"text-lg font-bold " + (m.pct >= 80 ? "text-green-600" : m.pct >= 50 ? "text-yellow-600" : "text-red-600")}>{m.pct}%</span></div><div className="mt-2 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={"h-full rounded-full " + (m.pct >= 80 ? "bg-green-500" : m.pct >= 50 ? "bg-yellow-500" : "bg-red-500")} style={{ width: `${m.pct}%` }} /></div></div>
                ))}
              </div>
              {report.pending > 0 && <div className={cardCls}><div className="flex items-center justify-between"><div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-yellow-500" /><span className="text-sm font-medium">{report.pending} users have not given consent</span></div><button className="flex items-center gap-1 rounded-lg bg-orange-600 px-3 py-2 text-sm font-medium text-white hover:bg-orange-700"><Bell className="h-4 w-4" /> Send Batch Notification</button></div></div>}
            </>
          ) : <div className={cardCls}><div className="py-8 text-center"><FileText className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No compliance report available.</p></div></div>}
        </div>
      )}

      </>)}
    </div>
  );
}
