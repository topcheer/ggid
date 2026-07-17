"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, CheckCircle, XCircle,
  Globe, Database, Lock, FileText, AlertTriangle, TrendingUp,
  Plus, Clock, ArrowRight, Eye, Download, Filter, Scale,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

// ==================== Types ====================
interface ComplianceLaw {
  id: string;
  name: string;
  jurisdiction: string;
  compliance_pct: number;
  requirements: { id: string; title: string; status: "met" | "partial" | "unmet" }[];
  last_audit: string;
}

interface DataAsset {
  id: string;
  name: string;
  category: "personal" | "financial" | "health" | "credentials" | "operational";
  classification: "general" | "important" | "core";
  fields: string[];
  retention_days: number;
  encrypted: boolean;
  access_count: number;
  cross_border: boolean;
}

interface DSRRequest {
  id: string;
  request_type: "access" | "delete" | "portability" | "rectify" | "restrict";
  law: string;
  user_id: string;
  username: string;
  status: "submitted" | "processing" | "completed" | "rejected" | "overdue";
  submitted_at: string;
  deadline: string;
  assigned_to: string | null;
  notes: string;
}

interface PIIField {
  id: string;
  field_name: string;
  table: string;
  classification: "general" | "important" | "core";
  pii_type: string;
  access_roles: string[];
  retention_days: number;
  masked: boolean;
}

interface CrossBorderTransfer {
  id: string;
  data_type: string;
  source_region: string;
  destination_region: string;
  legal_basis: string;
  status: "approved" | "pending" | "rejected" | "expired";
  approved_by: string | null;
  approved_at: string | null;
  expires_at: string;
}

const categoryColors: Record<string, string> = {
  personal: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  financial: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  health: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  credentials: "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
  operational: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
};

const classColors: Record<string, string> = {
  general: "text-green-600 bg-green-50 dark:bg-green-950/20",
  important: "text-yellow-600 bg-yellow-50 dark:bg-yellow-950/20",
  core: "text-red-600 bg-red-50 dark:bg-red-950/20",
};

const dsrStatusColors: Record<string, string> = {
  submitted: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  processing: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  completed: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  rejected: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  overdue: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 animate-pulse",
};

const transferStatusColors: Record<string, string> = {
  approved: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  pending: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  rejected: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  expired: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
};

type Tab = "scorecard" | "assets" | "pii" | "dsr" | "crossborder";

export default function DataGovernancePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("scorecard");
  const [laws, setLaws] = useState<ComplianceLaw[]>([]);
  const [assets, setAssets] = useState<DataAsset[]>([]);
  const [dsrRequests, setDsrRequests] = useState<DSRRequest[]>([]);
  const [piiFields, setPiiFields] = useState<PIIField[]>([]);
  const [transfers, setTransfers] = useState<CrossBorderTransfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  // DSR create
  const [showDsrForm, setShowDsrForm] = useState(false);
  const [dsrType, setDsrType] = useState<DSRRequest["request_type"]>("access");
  const [dsrUserId, setDsrUserId] = useState("");
  const [dsrLaw, setDsrLaw] = useState("GDPR");
  const [submitting, setSubmitting] = useState(false);
  // Cross-border approve
  const [actingId, setActingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [lawsRes, assetsRes, dsrRes, piiRes, transferRes] = await Promise.all([
        fetch("/api/v1/audit/compliance/laws", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/data-assets", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/dsr", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/pii-inventory", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/cross-border", { headers: h }).catch(() => null),
      ]);
      if (lawsRes?.ok) { const d = await lawsRes.json(); setLaws(d.laws || d.items || []); }
      if (assetsRes?.ok) { const d = await assetsRes.json(); setAssets(d.assets || d.items || []); }
      if (dsrRes?.ok) { const d = await dsrRes.json(); setDsrRequests(d.requests || d.items || []); }
      if (piiRes?.ok) { const d = await piiRes.json(); setPiiFields(d.fields || d.items || []); }
      if (transferRes?.ok) { const d = await transferRes.json(); setTransfers(d.transfers || d.items || []); }
    } catch { setError("Failed to load data governance data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const submitDsr = async () => {
    if (!dsrUserId) return;
    setSubmitting(true);
    try {
      const res = await fetch("/api/v1/audit/compliance/dsr", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ request_type: dsrType, user_id: dsrUserId, law: dsrLaw }),
      });
      if (res.ok) { setShowDsrForm(false); setDsrUserId(""); loadData(); }
      else { setError("Failed to submit DSR request"); }
    } catch { setError("Network error"); }
    finally { setSubmitting(false); }
  };

  const approveTransfer = async (id: string, approved: boolean) => {
    setActingId(id);
    try {
      await fetch(`/api/v1/audit/compliance/cross-border/${id}/${approved ? "approve" : "reject"}`, {
        method: "POST",
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      setTransfers(prev => prev.map(tr => tr.id === id ? { ...tr, status: approved ? "approved" : "rejected" } : tr));
    } catch { setError("Failed to update transfer"); }
    finally { setActingId(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const tabs: { id: Tab; label: string; icon: typeof Shield }[] = [
    { id: "scorecard", label: "Compliance Scorecard", icon: Scale },
    { id: "assets", label: "Data Assets", icon: Database },
    { id: "pii", label: "PII Inventory", icon: Lock },
    { id: "dsr", label: "DSR Requests", icon: FileText },
    { id: "crossborder", label: "Cross-Border", icon: Globe },
  ];

  const filteredPii = search ? piiFields.filter(f => f.field_name.includes(search) || f.table.includes(search) || f.pii_type.includes(search)) : piiFields;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="h-6 w-6 text-emerald-500" />
            {t("dataGovernance.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("dataGovernance.subtitle")}
          </p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {tabs.map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
            className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " +
              (tab === tb.id ? "border-emerald-600 text-emerald-600 dark:text-emerald-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}>
            <Icon className="h-4 w-4" /> {tb.label}
          </button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-500" /></div> : (<>

      {/* COMPLIANCE SCORECARD */}
      {tab === "scorecard" && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {laws.length === 0 ? (
              <div className={cardCls + " sm:col-span-2 lg:col-span-4"}><div className="py-8 text-center"><Scale className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("dataGovernance.noComplianceData")}</p></div></div>
            ) : laws.map(law => (
              <div key={law.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <div><span className="font-bold text-gray-900 dark:text-white">{law.name}</span><p className="text-xs text-gray-400">{law.jurisdiction}</p></div>
                  <span className={"text-2xl font-bold " + (law.compliance_pct >= 90 ? "text-green-600" : law.compliance_pct >= 70 ? "text-yellow-600" : "text-red-600")}>{law.compliance_pct}%</span>
                </div>
                <div className="mt-2 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div className={"h-full rounded-full " + (law.compliance_pct >= 90 ? "bg-green-500" : law.compliance_pct >= 70 ? "bg-yellow-500" : "bg-red-500")} style={{ width: `${law.compliance_pct}%` }} />
                </div>
                <div className="mt-3 space-y-1">
                  {law.requirements?.map(req => (
                    <div key={req.id} className="flex items-center gap-2 text-xs">
                      {req.status === "met" ? <CheckCircle className="h-3 w-3 text-green-500" /> : req.status === "partial" ? <AlertTriangle className="h-3 w-3 text-yellow-500" /> : <XCircle className="h-3 w-3 text-red-500" />}
                      <span className="text-gray-600 dark:text-gray-400">{req.title}</span>
                    </div>
                  ))}
                </div>
                {law.last_audit && <p className="mt-2 text-xs text-gray-400">Last audit: {new Date(law.last_audit).toLocaleDateString()}</p>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* DATA ASSETS */}
      {tab === "assets" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Database className="h-4 w-4" /> Data Classification Matrix</h2>
          {assets.length === 0 ? <div className="py-8 text-center"><Database className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("dataGovernance.noAssets")}</p></div> : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Asset</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Category</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Classification</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Retention</th>
                  <th scope="col" className="px-4 py-3 text-center font-medium">Encrypted</th>
                  <th scope="col" className="px-4 py-3 text-center font-medium">Cross-Border</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Access</th>
                </tr></thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {assets.map(a => (
                    <tr key={a.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-4 py-3"><span className="font-medium">{a.name}</span><div className="text-xs text-gray-400">{a.fields?.join(", ")}</div></td>
                      <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + (categoryColors[a.category] || "")}>{a.category}</span></td>
                      <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (classColors[a.classification] || "")}>{a.classification}</span></td>
                      <td className="px-4 py-3 text-right text-xs">{a.retention_days}d</td>
                      <td className="px-4 py-3 text-center">{a.encrypted ? <Lock className="h-4 w-4 mx-auto text-green-500" /> : <XCircle className="h-4 w-4 mx-auto text-red-500" />}</td>
                      <td className="px-4 py-3 text-center">{a.cross_border ? <Globe className="h-4 w-4 mx-auto text-orange-500" /> : <span className="text-xs text-gray-300">—</span>}</td>
                      <td className="px-4 py-3 text-right text-xs font-mono">{a.access_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* PII INVENTORY */}
      {tab === "pii" && (
        <div className={cardCls}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> PII Data Inventory</h2>
            <div className="relative">
              <Filter className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" />
              <input aria-label="Search PII fields" type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder="Search fields..." className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" />
            </div>
          </div>
          {filteredPii.length === 0 ? <div className="py-8 text-center"><Lock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("dataGovernance.noPiiFields")}</p></div> : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Field</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Table</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">PII Type</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Classification</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Access Roles</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Retention</th>
                  <th scope="col" className="px-4 py-3 text-center font-medium">Masked</th>
                </tr></thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {filteredPii.map(f => (
                    <tr key={f.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-4 py-3 font-mono text-xs">{f.field_name}</td>
                      <td className="px-4 py-3 font-mono text-xs text-gray-500">{f.table}</td>
                      <td className="px-4 py-3 text-xs">{f.pii_type}</td>
                      <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (classColors[f.classification] || "")}>{f.classification}</span></td>
                      <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{f.access_roles?.map(r => <span key={r} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-xs font-mono">{r}</span>)}</div></td>
                      <td className="px-4 py-3 text-right text-xs">{f.retention_days}d</td>
                      <td className="px-4 py-3 text-center">{f.masked ? <Lock className="h-4 w-4 mx-auto text-green-500" /> : <Eye className="h-4 w-4 mx-auto text-gray-400" />}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* DSR REQUESTS */}
      {tab === "dsr" && (
        <>
          <div className="flex justify-end">
            <button onClick={() => setShowDsrForm(true)} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-700"><Plus className="h-4 w-4" /> New DSR Request</button>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileText className="h-4 w-4" /> Data Subject Rights Requests</h2>
            {dsrRequests.length === 0 ? <div className="py-8 text-center"><FileText className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No DSR requests.</p></div> : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-900/50"><tr>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Type</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Law</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Submitted</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Deadline</th>
                    <th scope="col" className="px-4 py-3 text-left font-medium">Assigned</th>
                  </tr></thead>
                  <tbody className="divide-y dark:divide-gray-800">
                    {dsrRequests.map(r => (
                      <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                        <td className="px-4 py-3"><span className="font-mono text-xs">{r.request_type}</span></td>
                        <td className="px-4 py-3 text-xs">{r.username || r.user_id}</td>
                        <td className="px-4 py-3 text-xs">{r.law}</td>
                        <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (dsrStatusColors[r.status] || "")}>{r.status}</span></td>
                        <td className="px-4 py-3 text-xs text-gray-500">{new Date(r.submitted_at).toLocaleDateString()}</td>
                        <td className="px-4 py-3 text-xs"><span className={new Date(r.deadline) < new Date() ? "text-red-600 font-medium" : "text-gray-500"}>{new Date(r.deadline).toLocaleDateString()}</span></td>
                        <td className="px-4 py-3 text-xs text-gray-500">{r.assigned_to || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}

      {/* CROSS-BORDER */}
      {tab === "crossborder" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Globe className="h-4 w-4" /> Cross-Border Data Transfer Approvals</h2>
          {transfers.length === 0 ? <div className="py-8 text-center"><Globe className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No cross-border transfers registered.</p></div> : (
            <div className="space-y-3">
              {transfers.map(tr => (
                <div key={tr.id} className="flex items-start justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-sm font-medium">{tr.data_type}</span>
                      <span className={"px-2 py-0.5 rounded text-xs " + (transferStatusColors[tr.status] || "")}>{tr.status}</span>
                    </div>
                    <div className="mt-2 flex items-center gap-2 text-xs text-gray-500">
                      <span>{tr.source_region}</span>
                      <ArrowRight className="h-3 w-3" />
                      <span>{tr.destination_region}</span>
                      <span className="ml-3">Legal basis: <span className="font-medium">{tr.legal_basis || "—"}</span></span>
                    </div>
                    {tr.approved_by && <p className="mt-1 text-xs text-gray-400">Approved by: {tr.approved_by} · {tr.approved_at ? new Date(tr.approved_at).toLocaleDateString() : "—"}</p>}
                    <p className="mt-0.5 text-xs text-gray-400">Expires: {new Date(tr.expires_at).toLocaleDateString()}</p>
                  </div>
                  {tr.status === "pending" && (
                    <div className="flex items-center gap-2">
                      <button onClick={() => approveTransfer(tr.id, true)} disabled={actingId === tr.id} className="rounded-lg bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700 hover:bg-green-100 dark:bg-green-950/20 disabled:opacity-50">Approve</button>
                      <button onClick={() => approveTransfer(tr.id, false)} disabled={actingId === tr.id} className="rounded-lg bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100 dark:bg-red-950/20 disabled:opacity-50">Reject</button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      </>)}

      {/* DSR Create Dialog */}
      {showDsrForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowDsrForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><FileText className="h-5 w-5 text-emerald-500" /> New DSR Request</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Request Type</label>
                <select aria-label="DSR type" value={dsrType} onChange={e => setDsrType(e.target.value as DSRRequest["request_type"])} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="access">Access (right to know)</option>
                  <option value="delete">Delete (right to erasure)</option>
                  <option value="portability">Portability</option>
                  <option value="rectify">Rectification</option>
                  <option value="restrict">Restriction</option>
                </select>
              </div>
              <div><label className="text-sm font-medium">User ID *</label><input aria-label="User ID" type="text" value={dsrUserId} onChange={e => setDsrUserId(e.target.value)} placeholder="user-uuid" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Regulation</label>
                <select aria-label="DSR law" value={dsrLaw} onChange={e => setDsrLaw(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="GDPR">GDPR (EU)</option>
                  <option value="PIPL">PIPL (China)</option>
                  <option value="CCPA">CCPA (California)</option>
                  <option value="ISO27001">ISO 27001</option>
                </select>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowDsrForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={submitDsr} disabled={!dsrUserId || submitting} className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50">{submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : "Submit"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
