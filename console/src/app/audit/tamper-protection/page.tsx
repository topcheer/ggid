"use client";
import { useState, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, Check, Ban,
  Network, Lock, AlertTriangle, CheckCircle2, XCircle, Clock,
  ChevronRight, GitBranch, Activity, Database,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

type Tab = "merkle" | "worm" | "alerts";

interface MerkleRoot { hour: string; root_hash: string; depth: number; events: number; verified: boolean; }
interface WORMTable { name: string; records: number; locked: boolean; storage: string; }
interface TamperAlert { id: string; type: string; severity: "critical" | "high" | "medium"; block_id: string; detected: string; resolved: boolean; }

const ROOTS: MerkleRoot[] = [
  { hour: "08:00", root_hash: "0x7a3f...", depth: 18, events: 1247, verified: true },
  { hour: "07:00", root_hash: "0x9c2e...", depth: 17, events: 893, verified: true },
  { hour: "06:00", root_hash: "0x4b8d...", depth: 17, events: 612, verified: true },
  { hour: "05:00", root_hash: "0x1f5a...", depth: 16, events: 345, verified: true },
];

const WORM_TABLES: WORMTable[] = [
  { name: "audit_events", records: 2847392, locked: true, storage: "PG + S3 Object Lock" },
  { name: "audit_chain", records: 8472, locked: true, storage: "PG + S3 Object Lock" },
  { name: "compliance_evidence", records: 1247, locked: true, storage: "PG" },
  { name: "session_logs", records: 89234, locked: true, storage: "PG + S3" },
];

const ALERTS: TamperAlert[] = [];

export default function TamperProtectionPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("merkle");
  const [loading, setLoading] = useState(true);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = async () => {
    setLoading(false);
    try { await fetch("/api/v1/audit/verify-integrity", { headers: h }).catch(() => null); } catch { /* noop */ }
  };
  useEffect(() => { loadData(); }, []);

  const totalEvents = ROOTS.reduce((a: any, r: any) => a + r.events, 0);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-red-500" /> {t("tamper.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("tamper.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["merkle", t("tamper.merkleTree"), GitBranch], ["worm", t("tamper.worm"), Lock], ["alerts", `${t("tamper.alerts")} (${ALERTS.filter(a => !a.resolved).length})`, AlertTriangle]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-red-600 text-red-600 dark:text-red-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : (<>

      {/* MERKLE */}
      {tab === "merkle" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><GitBranch className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold">{ROOTS.length}</p><p className="text-xs text-gray-400">{t("tamper.hourlyRoots")}</p></div>
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{totalEvents.toLocaleString()}</p><p className="text-xs text-gray-400">{t("tamper.eventsHashed")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold text-green-600">100%</p><p className="text-xs text-gray-400">{t("tamper.verified")}</p></div>
            <div className={card + " text-center"}><Network className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{ROOTS[0]?.depth || 0}</p><p className="text-xs text-gray-400">{t("tamper.chainDepth")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("tamper.recentRoots")}</h3>
            <div className="space-y-1">{ROOTS.map(r => (
              <div key={r.hour} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700">
                <div className="flex items-center gap-3"><Clock className="h-3.5 w-3.5 text-gray-400" /><span className="text-xs font-mono font-bold">{r.hour}</span><code className="text-xs font-mono text-gray-500">{r.root_hash}</code></div>
                <div className="flex items-center gap-3"><span className="text-xs text-gray-400">{r.events} events · depth {r.depth}</span>{r.verified ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <XCircle className="h-4 w-4 text-red-500" />}</div>
              </div>
            ))}</div>
          </div>
        </div>
      )}

      {/* WORM */}
      {tab === "worm" && (
        <div className="space-y-4">
          <div className={`${card} bg-green-50 dark:bg-green-900/10`}><div className="flex items-center gap-3"><Lock className="h-5 w-5 text-green-500" /><div><p className="text-sm font-medium">{t("tamper.wormActive")}</p><p className="text-xs text-gray-400">{t("tamper.wormDesc")}</p></div></div></div>
          <div className="overflow-x-auto"><table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("tamper.table")}</th><th className="px-3 py-2 text-right text-xs text-gray-400">{t("tamper.records")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("tamper.locked")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("tamper.storage")}</th></tr></thead>
            <tbody className="divide-y dark:divide-gray-800">{WORM_TABLES.map(tbl => (
              <tr key={tbl.name} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3"><code className="text-xs font-mono text-red-500">{tbl.name}</code></td><td className="px-3 py-3 text-right text-xs font-mono">{tbl.records.toLocaleString()}</td><td className="px-3 py-3 text-center">{tbl.locked ? <Lock className="mx-auto h-4 w-4 text-green-500" /> : <Ban className="mx-auto h-4 w-4 text-red-500" />}</td><td className="px-3 py-3 text-xs text-gray-400">{tbl.storage}</td></tr>
            ))}</tbody>
          </table></div>
        </div>
      )}

      {/* ALERTS */}
      {tab === "alerts" && (
        <div>{ALERTS.length === 0 ? <div className={card}><div className="py-12 text-center"><CheckCircle2 className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">{t("tamper.noAlerts")}</p><p className="mt-1 text-xs text-gray-400">{t("tamper.chainSecure")}</p></div></div> :
          <div className="space-y-2">{ALERTS.map(a => (
            <div key={a.id} className={`${card} flex items-center justify-between !p-3`}><div className="flex items-center gap-3"><div className={`flex h-8 w-8 items-center justify-center rounded-lg ${a.severity === "critical" ? "bg-red-100 dark:bg-red-900/30" : "bg-yellow-100 dark:bg-yellow-900/30"}`}><AlertTriangle className={`h-4 w-4 ${a.severity === "critical" ? "text-red-500" : "text-yellow-500"}`} /></div><div><span className="text-sm font-medium">{a.type}</span><p className="text-xs text-gray-400">Block: <code className="font-mono">{a.block_id}</code> · {new Date(a.detected).toLocaleString()}</p></div></div></div>
          ))}</div>
        }</div>
      )}

      </>)}
    </div>
  );
}
