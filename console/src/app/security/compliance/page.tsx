"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, CheckCircle, XCircle,
  Download, FileText, Calendar, Search, ChevronRight, ChevronLeft,
  Link2, Hash, Lock, ArrowRight, Filter, Clock, AlertTriangle,
  Upload, Eye, Sparkles,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Framework {
  id: string;
  name: string;
  version: string;
  jurisdiction: string;
  total_controls: number;
  met: number;
  partial: number;
  unmet: number;
  auto_collected: number;
  manual: number;
}

interface ControlItem {
  id: string;
  framework_id: string;
  control_ref: string;
  title: string;
  description: string;
  status: "met" | "partial" | "unmet";
  collection: "auto" | "manual";
  evidence_count: number;
  last_collected: string;
  mapped_ggid_controls: string[];
}

interface ExportJob {
  id: string;
  framework: string;
  date_range: string;
  format: string;
  status: "queued" | "generating" | "completed" | "failed";
  download_url: string | null;
  created_at: string;
}

interface HashChainProof {
  block_height: number;
  verified: boolean;
  last_block_hash: string;
  tamper_alerts: number;
  verified_at: string;
}

const FRAMEWORKS = [
  { id: "soc2", name: "SOC 2 Type II", version: "2017 TSC", jurisdiction: "USA", icon: Shield, color: "text-blue-500" },
  { id: "iso27001", name: "ISO 27001", version: "2022", jurisdiction: "Global", icon: Lock, color: "text-green-500" },
  { id: "dengbao", name: "等保 2.0", version: "Level 3", jurisdiction: "China", icon: Shield, color: "text-red-500" },
  { id: "gdpr", name: "GDPR", version: "2018", jurisdiction: "EU", icon: FileText, color: "text-purple-500" },
  { id: "pipl", name: "PIPL", version: "2021", jurisdiction: "China", icon: FileText, color: "text-orange-500" },
];

const statusConfig = {
  met: { color: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400", icon: CheckCircle, label: "Met" },
  partial: { color: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400", icon: AlertTriangle, label: "Partial" },
  unmet: { color: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400", icon: XCircle, label: "Unmet" },
};

const EXPORT_STEPS = ["Framework", "Date Range", "Controls", "Format", "Review"];
const FORMATS = [
  { id: "pdf", name: "PDF Report", icon: FileText, desc: "Formatted report with cover page" },
  { id: "csv", name: "CSV Bundle", icon: Download, desc: "Machine-readable spreadsheet" },
  { id: "zip", name: "ZIP Archive", icon: Upload, desc: "All evidence files bundled" },
];

type Tab = "dashboard" | "controls" | "export" | "mapping" | "audit";

export default function ComplianceExportPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("dashboard");
  const [frameworks, setFrameworks] = useState<Framework[]>([]);
  const [controls, setControls] = useState<ControlItem[]>([]);
  const [exports, setExports] = useState<ExportJob[]>([]);
  const [hashProof, setHashProof] = useState<HashChainProof | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Filters
  const [selectedFramework, setSelectedFramework] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [searchCtrl, setSearchCtrl] = useState("");
  // Export wizard
  const [showWizard, setShowWizard] = useState(false);
  const [wizStep, setWizStep] = useState(0);
  const [wizFramework, setWizFramework] = useState("");
  const [wizFromDate, setWizFromDate] = useState("");
  const [wizToDate, setWizToDate] = useState("");
  const [wizSelectedControls, setWizSelectedControls] = useState<string[]>([]);
  const [wizFormat, setWizFormat] = useState("pdf");
  const [wizGenerating, setWizGenerating] = useState(false);
  const [wizResult, setWizResult] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [fwRes, ctrlRes, expRes, hashRes] = await Promise.all([
        fetch("/api/v1/audit/compliance/frameworks", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/controls?page_size=100", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/exports", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/hash-chain/proof", { headers: h }).catch(() => null),
      ]);
      if (fwRes?.ok) { const d = await fwRes.json(); setFrameworks(d.frameworks || d.items || []); }
      if (ctrlRes?.ok) { const d = await ctrlRes.json(); setControls(d.controls || d.items || []); }
      if (expRes?.ok) { const d = await expRes.json(); setExports(d.exports || d.items || []); }
      if (hashRes?.ok) setHashProof(await hashRes.json());
    } catch { setError("Failed to load compliance data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const startExport = async () => {
    setWizGenerating(true);
    try {
      const res = await fetch("/api/v1/audit/compliance/exports", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ framework: wizFramework, from_date: wizFromDate, to_date: wizToDate, controls: wizSelectedControls, format: wizFormat }),
      });
      if (res.ok) {
        const d = await res.json();
        setWizResult(d.download_url || "Export queued. You'll be notified when ready.");
        loadData();
      }
    } catch { setError("Export failed"); }
    finally { setWizGenerating(false); }
  };

  const toggleControl = (id: string) => {
    setWizSelectedControls(prev => prev.includes(id) ? prev.filter(c => c !== id) : [...prev, id]);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filteredControls = controls.filter(c => {
    if (selectedFramework && c.framework_id !== selectedFramework) return false;
    if (statusFilter !== "all" && c.status !== statusFilter) return false;
    if (searchCtrl && !c.control_ref.includes(searchCtrl) && !c.title.includes(searchCtrl)) return false;
    return true;
  });
  const wizControls = controls.filter(c => !wizFramework || c.framework_id === wizFramework);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-emerald-500" /> {t("complianceExport.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("complianceExport.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => { setWizStep(0); setWizFramework(""); setWizSelectedControls([]); setWizResult(null); setShowWizard(true); }} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-700"><Download className="h-4 w-4" /> Export Evidence</button>
          <button onClick={loadData} disabled={loading} aria-label="Refresh" className="rounded-lg border border-gray-300 p-2 text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /></button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "dashboard" as Tab, label: "Dashboard", icon: Shield },
          { id: "controls" as Tab, label: "Controls", icon: CheckCircle },
          { id: "export" as Tab, label: "Export History", icon: Download },
          { id: "mapping" as Tab, label: "Control Mapping", icon: Link2 },
          { id: "audit" as Tab, label: "Audit Chain", icon: Hash },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-emerald-600 text-emerald-600 dark:text-emerald-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-500" /></div> : (<>

      {/* DASHBOARD */}
      {tab === "dashboard" && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
            {frameworks.length === 0 ? FRAMEWORKS.map(fw => (
              <div key={fw.id} className={cardCls + " text-center opacity-50"}><fw.icon className={"h-8 w-8 mx-auto " + fw.color} /><p className="mt-2 text-xs font-medium">{fw.name}</p><p className="text-xs text-gray-400">{fw.jurisdiction}</p></div>
            )) : frameworks.map(fw => {
              const fdef = FRAMEWORKS.find(f => f.id === fw.id) || FRAMEWORKS[0];
              const pct = fw.total_controls ? Math.round((fw.met / fw.total_controls) * 100) : 0;
              return (
                <div key={fw.id} className={cardCls + " hover:shadow-md transition"}>
                  <div className="flex items-center gap-2"><fdef.icon className={"h-5 w-5 " + fdef.color} /><div><p className="font-semibold text-sm">{fw.name}</p><p className="text-xs text-gray-400">{fw.version}</p></div></div>
                  <div className="mt-3"><div className="flex items-center justify-between"><span className="text-xs text-gray-400">Compliance</span><span className={"text-lg font-bold " + (pct >= 80 ? "text-green-600" : pct >= 50 ? "text-yellow-600" : "text-red-600")}>{pct}%</span></div><div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={"h-full rounded-full " + (pct >= 80 ? "bg-green-500" : pct >= 50 ? "bg-yellow-500" : "bg-red-500")} style={{ width: `${pct}%` }} /></div></div>
                  <div className="mt-3 grid grid-cols-3 gap-1 text-center"><div><p className="text-xs text-green-500">{fw.met}</p><p className="text-xs text-gray-400">Met</p></div><div><p className="text-xs text-yellow-500">{fw.partial}</p><p className="text-xs text-gray-400">Partial</p></div><div><p className="text-xs text-red-500">{fw.unmet}</p><p className="text-xs text-gray-400">Unmet</p></div></div>
                  <div className="mt-2 flex items-center justify-between text-xs text-gray-400"><span><Sparkles className="inline h-3 w-3" /> {fw.auto_collected} auto</span><span><Upload className="inline h-3 w-3" /> {fw.manual} manual</span></div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* CONTROLS */}
      {tab === "controls" && (
        <div className={cardCls}>
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <select aria-label="Filter framework" value={selectedFramework} onChange={e => setSelectedFramework(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm"><option value="">All Frameworks</option>{FRAMEWORKS.map(f => <option key={f.id} value={f.id}>{f.name}</option>)}</select>
            <select aria-label="Filter status" value={statusFilter} onChange={e => setStatusFilter(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm"><option value="all">All Status</option><option value="met">Met</option><option value="partial">Partial</option><option value="unmet">Unmet</option></select>
            <div className="relative flex-1 min-w-[200px]"><Search className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" /><input aria-label="Search controls" type="text" value={searchCtrl} onChange={e => setSearchCtrl(e.target.value)} placeholder="Search control ref or title..." className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" /></div>
            <span className="text-xs text-gray-400">{filteredControls.length} controls</span>
          </div>
          {filteredControls.length === 0 ? <div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No controls found.</p></div> : (
            <div className="overflow-x-auto max-h-[500px] overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80"><tr><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Ref</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Title</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Status</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Collection</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Evidence</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Last Collected</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{filteredControls.map(c => { const sc = statusConfig[c.status]; const SIcon = sc.icon; return (
                  <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-3 py-2"><span className="font-mono text-xs font-medium">{c.control_ref}</span></td>
                    <td className="px-3 py-2"><span className="text-xs">{c.title}</span><p className="text-xs text-gray-400">{c.description}</p></td>
                    <td className="px-3 py-2 text-center"><span className={"inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium " + sc.color}><SIcon className="h-3 w-3" /> {sc.label}</span></td>
                    <td className="px-3 py-2 text-center"><span className={"px-1.5 py-0.5 rounded text-xs " + (c.collection === "auto" ? "bg-blue-100 text-blue-600 dark:bg-blue-900/30" : "bg-orange-100 text-orange-600 dark:bg-orange-900/30")}>{c.collection}</span></td>
                    <td className="px-3 py-2 text-center text-xs">{c.evidence_count} files</td>
                    <td className="px-3 py-2 text-xs text-gray-500">{c.last_collected ? new Date(c.last_collected).toLocaleDateString() : "—"}</td>
                  </tr>
                ); })}</tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* EXPORT HISTORY */}
      {tab === "export" && (
        <div className={cardCls}>
          <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Export History</h2>
          {exports.length === 0 ? <div className="py-8 text-center"><Download className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No exports yet. Click "Export Evidence" to start.</p></div> : (
            <div className="space-y-2">{exports.map(ex => (
              <div key={ex.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3"><FileText className="h-5 w-5 text-gray-400" /><div><div className="flex items-center gap-2"><span className="font-medium text-sm">{ex.framework}</span><span className="text-xs text-gray-400">{ex.format.toUpperCase()}</span><span className={"px-1.5 py-0.5 rounded text-xs " + (ex.status === "completed" ? "bg-green-100 text-green-600 dark:bg-green-900/30" : ex.status === "generating" ? "bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30" : "bg-gray-100 dark:bg-gray-800")}>{ex.status}</span></div><p className="text-xs text-gray-400">{ex.date_range} · {new Date(ex.created_at).toLocaleString()}</p></div></div>
                {ex.download_url && ex.status === "completed" && <a href={ex.download_url} className="flex items-center gap-1 rounded-lg bg-emerald-50 px-3 py-1.5 text-xs font-medium text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-950/20 dark:text-emerald-400"><Download className="h-3 w-3" /> Download</a>}
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* CONTROL MAPPING */}
      {tab === "mapping" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Link2 className="h-4 w-4" /> CCM Control Cross-Mapping Matrix</h2>
          {filteredControls.length === 0 ? <div className="py-8 text-center"><Link2 className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No mapping data.</p></div> : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Control</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Title</th>{FRAMEWORKS.map(f => <th key={f.id} scope="col" className="px-2 py-2 text-center font-medium text-xs">{f.id.toUpperCase()}</th>)}</tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{filteredControls.map(c => (
                  <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-3 py-2"><span className="font-mono text-xs">{c.control_ref}</span></td>
                    <td className="px-3 py-2 text-xs">{c.title}</td>
                    {FRAMEWORKS.map(f => { const mapped = c.mapped_ggid_controls?.some(g => g.includes(f.id)) || c.framework_id === f.id; return <td key={f.id} className="px-2 py-2 text-center">{mapped ? <CheckCircle className="h-4 w-4 mx-auto text-green-500" /> : <span className="text-xs text-gray-300">—</span>}</td>; })}
                  </tr>
                ))}</tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* AUDIT CHAIN */}
      {tab === "audit" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Hash className="h-4 w-4" /> Hash Chain Integrity</h2>
            {hashProof ? (
              <div className="space-y-4">
                <div className={"flex items-center gap-3 rounded-xl border-2 p-4 " + (hashProof.verified ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30")}>
                  {hashProof.verified ? <CheckCircle className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                  <div><p className={"text-lg font-bold " + (hashProof.verified ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400")}>{hashProof.verified ? "CHAIN INTACT" : "CHAIN BROKEN"}</p><p className="text-xs text-gray-500">{hashProof.tamper_alerts} tamper alerts · {hashProof.block_height} blocks</p></div>
                </div>
                <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Last Block Hash</span><p className="font-mono text-xs break-all mt-1">{hashProof.last_block_hash}</p></div>
                <div className="rounded-lg border p-3 dark:border-gray-700"><span className="text-xs text-gray-400">Last Verified</span><p className="text-xs mt-1">{new Date(hashProof.verified_at).toLocaleString()}</p></div>
              </div>
            ) : <div className="py-8 text-center"><Hash className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No hash chain proof available.</p></div>}
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> Evidence Tamper Protection</h2>
            <div className="space-y-3">
              <div className="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30"><p className="text-xs text-blue-700 dark:text-blue-400">All compliance evidence is cryptographically linked via a hash chain. Each evidence block's hash includes the previous block's hash, creating an immutable audit trail.</p></div>
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm"><CheckCircle className="h-4 w-4 text-green-500" /><span>Append-only evidence storage</span></div>
                <div className="flex items-center gap-2 text-sm"><CheckCircle className="h-4 w-4 text-green-500" /><span>SHA-256 block hashing</span></div>
                <div className="flex items-center gap-2 text-sm"><CheckCircle className="h-4 w-4 text-green-500" /><span>Tamper detection alerts</span></div>
                <div className="flex items-center gap-2 text-sm"><CheckCircle className="h-4 w-4 text-green-500" /><span>Court-admissible audit trail</span></div>
              </div>
            </div>
          </div>
        </div>
      )}

      </>)}

      {/* EXPORT WIZARD */}
      {showWizard && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !wizGenerating && setShowWizard(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-6 flex items-center justify-between"><h3 className="text-lg font-semibold text-gray-900 dark:text-white">Export Evidence Package</h3>{!wizGenerating && <button onClick={() => setShowWizard(false)}><X className="h-5 w-5 text-gray-400" /></button>}</div>
            {/* Steps */}
            <div className="mb-6 flex items-center gap-1">{EXPORT_STEPS.map((s, i) => (
              <div key={i} className="flex items-center gap-1 flex-1"><div className={"flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold " + (i < wizStep ? "bg-green-600 text-white" : i === wizStep ? "bg-emerald-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400")}>{i < wizStep ? <CheckCircle className="h-3.5 w-3.5" /> : i + 1}</div>{i < EXPORT_STEPS.length - 1 && <div className={"h-0.5 flex-1 " + (i < wizStep ? "bg-green-600" : "bg-gray-200 dark:bg-gray-700")} />}</div>
            ))}</div>
            <div className="min-h-[200px]">
              {wizStep === 0 && <div className="space-y-2">{FRAMEWORKS.map(f => <button key={f.id} onClick={() => { setWizFramework(f.id); setWizSelectedControls([]); }} aria-pressed={wizFramework === f.id} className={"flex w-full items-center gap-3 rounded-xl border-2 p-4 text-left " + (wizFramework === f.id ? "border-emerald-500 bg-emerald-50 dark:bg-emerald-950/30" : "border-gray-200 dark:border-gray-700")}><f.icon className={"h-6 w-6 " + f.color} /><div className="flex-1"><p className="font-medium text-sm">{f.name}</p><p className="text-xs text-gray-400">{f.version} · {f.jurisdiction}</p></div>{wizFramework === f.id && <CheckCircle className="h-5 w-5 text-emerald-500" />}</button>)}</div>}
              {wizStep === 1 && <div className="space-y-3"><div><label className="text-sm font-medium">From Date</label><input aria-label="From date" type="date" value={wizFromDate} onChange={e => setWizFromDate(e.target.value)} className="mt-1 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div><div><label className="text-sm font-medium">To Date</label><input aria-label="To date" type="date" value={wizToDate} onChange={e => setWizToDate(e.target.value)} className="mt-1 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div></div>}
              {wizStep === 2 && <div className="space-y-2 max-h-60 overflow-y-auto">{wizControls.map(c => { const sc = statusConfig[c.status]; return <label key={c.id} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700 cursor-pointer"><input type="checkbox" checked={wizSelectedControls.includes(c.id)} onChange={() => toggleControl(c.id)} className="rounded" /><div className="flex-1"><span className="font-mono text-xs">{c.control_ref}</span> <span className="text-xs">{c.title}</span></div><span className={"px-1.5 py-0.5 rounded text-xs " + sc.color}>{sc.label}</span></label>; })}<p className="text-xs text-gray-400">{wizSelectedControls.length} selected (leave empty for all)</p></div>}
              {wizStep === 3 && <div className="space-y-2">{FORMATS.map(f => { const FIcon = f.icon; return <button key={f.id} onClick={() => setWizFormat(f.id)} aria-pressed={wizFormat === f.id} className={"flex w-full items-center gap-3 rounded-xl border-2 p-4 text-left " + (wizFormat === f.id ? "border-emerald-500 bg-emerald-50 dark:bg-emerald-950/30" : "border-gray-200 dark:border-gray-700")}><FIcon className="h-6 w-6 text-gray-400" /><div><p className="font-medium text-sm">{f.name}</p><p className="text-xs text-gray-400">{f.desc}</p></div>{wizFormat === f.id && <CheckCircle className="h-5 w-5 text-emerald-500" />}</button>; })}</div>}
              {wizStep === 4 && <div className="space-y-3">{wizResult ? <div className="rounded-xl border-2 border-green-300 bg-green-50 p-6 text-center dark:border-green-700 dark:bg-green-950/30"><CheckCircle className="h-12 w-12 mx-auto text-green-500" /><p className="mt-3 text-lg font-semibold text-green-700 dark:text-green-400">Export Ready!</p><p className="text-sm text-gray-500 mt-1">{wizResult}</p></div> : <><div className="rounded-lg border p-4 dark:border-gray-700 space-y-1 text-sm"><div><span className="text-gray-400">Framework:</span> {FRAMEWORKS.find(f => f.id === wizFramework)?.name || "—"}</div><div><span className="text-gray-400">Range:</span> {wizFromDate || "—"} to {wizToDate || "—"}</div><div><span className="text-gray-400">Controls:</span> {wizSelectedControls.length || "All"}</div><div><span className="text-gray-400">Format:</span> {FORMATS.find(f => f.id === wizFormat)?.name}</div></div><button onClick={startExport} disabled={wizGenerating} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-6 py-3 text-sm font-bold text-white hover:bg-emerald-700 disabled:opacity-50 mx-auto">{wizGenerating ? <Loader2 className="h-5 w-5 animate-spin" /> : <Download className="h-5 w-5" />} Generate Export</button></>}</div>}
            </div>
            {/* Nav */}
            {!wizResult && <div className="mt-6 flex justify-between"><button onClick={() => setWizStep(Math.max(0, wizStep - 1))} disabled={wizStep === 0 || wizGenerating} className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm disabled:opacity-30 dark:border-gray-700"><ChevronLeft className="h-4 w-4" /> Back</button>{wizStep < EXPORT_STEPS.length - 1 && <button onClick={() => setWizStep(wizStep + 1)} disabled={(wizStep === 0 && !wizFramework) || wizGenerating} className="flex items-center gap-1 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50">Next <ChevronRight className="h-4 w-4" /></button>}</div>}
          </div>
        </div>
      )}
    </div>
  );
}
