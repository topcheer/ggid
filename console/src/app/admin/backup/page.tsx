"use client";
import { useState, useEffect } from "react";
import {
  Database, Loader2, AlertCircle, X, RefreshCw, Check, Ban,
  CheckCircle2, XCircle, Clock, HardDrive, RotateCcw, Activity,
  Shield, Zap, ChevronRight, AlertTriangle,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "";

interface RestorePoint { id: string; timestamp: string; type: "full" | "WAL" | "snapshot"; size_gb: number; verified: boolean; }

type Tab = "status" | "restore" | "dr";

const RESTORE_POINTS: RestorePoint[] = [
  { id: "rp-001", timestamp: new Date(Date.now() - 3600000).toISOString(), type: "WAL", size_gb: 0.3, verified: true },
  { id: "rp-002", timestamp: new Date(Date.now() - 7200000).toISOString(), type: "WAL", size_gb: 0.4, verified: true },
  { id: "rp-003", timestamp: new Date(Date.now() - 86400000).toISOString(), type: "full", size_gb: 12.4, verified: true },
  { id: "rp-004", timestamp: new Date(Date.now() - 172800000).toISOString(), type: "full", size_gb: 11.8, verified: true },
  { id: "rp-005", timestamp: new Date(Date.now() - 259200000).toISOString(), type: "snapshot", size_gb: 10.2, verified: false },
];

export default function BackupPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("status");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmRestore, setConfirmRestore] = useState<RestorePoint | null>(null);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => { setLoading(false); }, []);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Database className="h-6 w-6 text-indigo-500" /> {t("backup.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("backup.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["status", t("backup.backupStatus"), Database], ["restore", t("backup.restorePoints"), RotateCcw], ["dr", t("backup.drStatus"), Shield]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* STATUS */}
      {tab === "status" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-sm font-bold">1h ago</p><p className="text-xs text-gray-400">{t("backup.lastBackup")}</p></div>
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-sm font-bold">in 1h</p><p className="text-xs text-gray-400">{t("backup.nextBackup")}</p></div>
            <div className={card + " text-center"}><HardDrive className="mx-auto h-5 w-5 text-purple-400" /><p className="mt-2 text-sm font-bold">12.4 GB</p><p className="text-xs text-gray-400">{t("backup.lastSize")}</p></div>
            <div className={card + " text-center"}><Database className="mx-auto h-5 w-5 text-indigo-400" /><p className="mt-2 text-sm font-bold">s3://ggid</p><p className="text-xs text-gray-400">{t("backup.location")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("backup.componentStatus")}</h3>
            <div className="space-y-2">
              {[["PostgreSQL", true, "Streaming WAL active"], ["Redis", true, "AOF + RDB snapshots"], ["Audit Log Chain", true, "Hash chain intact"], ["Config Store", true, "Synced to S3"]].map(([name, ok, detail]) => (
                <div key={name as string} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">{ok ? <CheckCircle2 className="h-5 w-5 text-green-500" /> : <XCircle className="h-5 w-5 text-red-500" />}<div><span className="text-sm font-medium">{name}</span><p className="text-xs text-gray-400">{detail}</p></div></div>
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${ok ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{ok ? t("backup.healthy") : t("backup.failed")}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* RESTORE */}
      {tab === "restore" && (
        <div>
          {RESTORE_POINTS.map(rp => (
            <div key={rp.id} className={`${card} mb-2 flex items-center justify-between !p-3`}>
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/30"><RotateCcw className="h-4 w-4 text-indigo-500" /></div>
                <div>
                  <div className="flex items-center gap-2"><span className="text-sm font-medium">{new Date(rp.timestamp).toLocaleString()}</span><span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700 font-mono">{rp.type}</span>{rp.verified ? <CheckCircle2 className="h-3.5 w-3.5 text-green-500" /> : <AlertTriangle className="h-3.5 w-3.5 text-yellow-500" />}</div>
                  <p className="text-xs text-gray-400">{rp.size_gb} GB · {rp.verified ? t("backup.verified") : t("backup.unverified")}</p>
                </div>
              </div>
              <button onClick={() => setConfirmRestore(rp)} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><RotateCcw className="h-3 w-3" /> {t("backup.restore")}</button>
            </div>
          ))}
        </div>
      )}

      {/* DR */}
      {tab === "dr" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">&lt;5m</p><p className="text-xs text-gray-400">{t("backup.rpo")}</p></div>
            <div className={card + " text-center"}><Zap className="mx-auto h-5 w-5 text-yellow-400" /><p className="mt-2 text-2xl font-bold">&lt;15m</p><p className="text-xs text-gray-400">{t("backup.rto")}</p></div>
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">0.3s</p><p className="text-xs text-gray-400">{t("backup.replLag")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold text-green-600">Ready</p><p className="text-xs text-gray-400">{t("backup.failover")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("backup.drChecks")}</h3>
            <div className="space-y-2">
              {[["Primary → Standby replication", true], ["DNS failover configured", true], ["Health check endpoints", true], ["Runbook documented", true], ["Last DR drill: 2025-01-10", true]].map(([check, ok]: any[]) => (
                <div key={check as string} className="flex items-center gap-3 rounded-lg border p-2 dark:border-gray-700">{ok ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <XCircle className="h-4 w-4 text-red-500" />}<span className="text-sm">{check}</span></div>
              ))}
            </div>
          </div>
        </div>
      )}

      </>)}

      {confirmRestore && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRestore(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-red-500" /><h3 className="text-lg font-semibold">{t("backup.restoreTitle")}</h3></div>
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{t("backup.restoreConfirm")} — {confirmRestore.type} from {new Date(confirmRestore.timestamp).toLocaleString()}?</p>
            {!confirmRestore.verified && <div className="mt-2 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 p-2 text-xs text-yellow-600"><AlertTriangle className="inline h-3 w-3" /> {t("backup.unverifiedWarning")}</div>}
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setConfirmRestore(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={() => setConfirmRestore(null)} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><RotateCcw className="h-4 w-4" /> {t("backup.confirmRestore")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
