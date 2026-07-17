"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Link2, Loader2, AlertCircle, X, RefreshCw, Play,
  CheckCircle2, XCircle, AlertTriangle, Clock, Shield,
  ChevronRight, Hash, FileText, Activity,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ChainStatus { head_hash: string; chain_length: number; last_verified: string; gaps: number; integrity_pct: number; }
interface VerificationRun { id: string; blocks_checked: number; anomalies: number; status: string; duration_ms: number; timestamp: string; }
interface TamperEvent { id: string; block_id: string; type: string; description: string; severity: string; detected_at: string; resolved: boolean; }

type Tab = "status" | "verification" | "tamper";

export default function IntegrityPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("status");
  const [status, setStatus] = useState<ChainStatus | null>(null);
  const [verifications, setVerifications] = useState<VerificationRun[]>([]);
  const [tamperEvents, setTamperEvents] = useState<TamperEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [verifying, setVerifying] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [sRes, vRes, tRes] = await Promise.all([
        fetch("/api/v1/audit/verify-integrity", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/tamper-check", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/verify-integrity?detail=tamper", { headers: h }).catch(() => null),
      ]);
      if (sRes?.ok) { const d = await sRes.json(); setStatus({ head_hash: d.head_hash || "sha256:0x...", chain_length: d.chain_length || d.total_blocks || 0, last_verified: d.last_verified || new Date().toISOString(), gaps: d.gaps || 0, integrity_pct: d.integrity_pct ?? 100 }); setVerifications(d.verification_log || []); }
      if (tRes?.ok) { const d = await tRes.json(); setTamperEvents(d.tamper_events || d.events || []); }
    } catch { setError(t("integrity.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const runVerification = async () => {
    setVerifying(true);
    try { await fetch("/api/v1/audit/verify-integrity", { method: "POST", headers: h }); loadData(); }
    catch { setError(t("integrity.verifyError")); }
    finally { setVerifying(false); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Link2 className="h-6 w-6 text-blue-500" /> {t("integrity.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("integrity.subtitle")}</p></div>
        <button onClick={runVerification} disabled={verifying} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{verifying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} {t("integrity.runVerification")}</button>
      </div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "status" as Tab, label: t("integrity.chainStatus"), icon: Link2 },
          { id: "verification" as Tab, label: t("integrity.verificationLog"), icon: Clock },
          { id: "tamper" as Tab, label: `${t("integrity.tamperDetection")} (${tamperEvents.filter(e => !e.resolved).length})`, icon: AlertTriangle },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {tb.label}</button>
        );})}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div> : (<>

      {/* STATUS */}
      {tab === "status" && status && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Link2 className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{status.chain_length.toLocaleString()}</p><p className="text-xs text-gray-400">{t("integrity.chainLength")}</p></div>
            <div className={card + " text-center"}><Shield className="mx-auto h-5 w-5 text-green-400" /><p className={`mt-2 text-2xl font-bold ${status.integrity_pct === 100 ? "text-green-600" : "text-red-600"}`}>{status.integrity_pct}%</p><p className="text-xs text-gray-400">{t("integrity.integrity")}</p></div>
            <div className={card + " text-center"}><AlertTriangle className="mx-auto h-5 w-5 text-yellow-400" /><p className="mt-2 text-2xl font-bold">{status.gaps}</p><p className="text-xs text-gray-400">{t("integrity.gaps")}</p></div>
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-gray-400" /><p className="mt-2 text-sm font-mono">{status.last_verified ? new Date(status.last_verified).toLocaleDateString() : "—"}</p><p className="text-xs text-gray-400">{t("integrity.lastVerified")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Hash className="h-4 w-4" /> {t("integrity.headHash")}</h3>
            <code className="text-xs font-mono break-all text-gray-500">{status.head_hash}</code>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("integrity.integrityScore")}</h3>
            <div className="h-4 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={`h-full rounded-full ${status.integrity_pct === 100 ? "bg-green-500" : "bg-red-500"}`} style={{ width: `${status.integrity_pct}%` }} /></div>
            <p className="mt-2 text-xs text-gray-400">{status.integrity_pct === 100 ? t("integrity.allBlocksValid") : t("integrity.blocksFailed")}</p>
          </div>
        </div>
      )}

      {/* VERIFICATION LOG */}
      {tab === "verification" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("integrity.time")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("integrity.blocksChecked")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("integrity.anomalies")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("integrity.duration")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("integrity.status")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{verifications.map(v => (
            <tr key={v.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs">{new Date(v.timestamp).toLocaleString()}</td><td className="px-3 py-3 text-center text-xs font-mono">{v.blocks_checked.toLocaleString()}</td><td className="px-3 py-3 text-center"><span className={`text-xs font-mono ${v.anomalies > 0 ? "text-red-600 font-bold" : "text-gray-400"}`}>{v.anomalies}</span></td><td className="px-3 py-3 text-center text-xs font-mono">{v.duration_ms}ms</td><td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${v.status === "pass" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{v.status}</span></td></tr>
          ))}</tbody>
        </table></div>
      )}

      {/* TAMPER */}
      {tab === "tamper" && (
        <div>
          {tamperEvents.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><CheckCircle2 className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">{t("integrity.noTamper")}</p></div></div>
          ) : (
            <div className="space-y-2">{tamperEvents.map(ev => (
              <div key={ev.id} className={`${card} flex items-center justify-between !p-3 ${!ev.resolved ? "border-red-200 dark:border-red-800" : "opacity-60"}`}>
                <div className="flex items-center gap-3"><div className={`flex h-9 w-9 items-center justify-center rounded-lg ${ev.severity === "critical" ? "bg-red-100 dark:bg-red-900/30" : "bg-yellow-100 dark:bg-yellow-900/30"}`}><AlertTriangle className={`h-4 w-4 ${ev.severity === "critical" ? "text-red-500" : "text-yellow-500"}`} /></div><div><div className="flex items-center gap-2"><span className="text-sm font-medium">{ev.type}</span><code className="text-xs font-mono text-gray-500">{ev.block_id}</code></div><p className="text-xs text-gray-400">{ev.description}</p><p className="text-xs text-gray-400">{new Date(ev.detected_at).toLocaleString()}</p></div></div>
                <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${ev.resolved ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{ev.resolved ? t("integrity.resolved") : t("integrity.unresolved")}</span>
              </div>
            ))}</div>
          )}
        </div>
      )}

      </>)}
    </div>
  );
}
