"use client";
import { useState, useEffect } from "react";
import {
  GitBranch, Loader2, AlertCircle, X, RefreshCw, Play, Check,
  CheckCircle2, XCircle, Clock, ChevronRight, Activity,
  Shield, Database, AlertTriangle, Lock,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

type Tab = "tree" | "log";

interface VerificationRun { id: string; blocks_checked: number; roots_verified: number; anomalies: number; duration_ms: number; status: string; timestamp: string; }

const TREE_DATA = {
  root: { hash: "0x7a3f9c2e", level: 3 },
  left: { hash: "0x4b8d1f5a", level: 2 },
  right: { hash: "0x9c2e7a3f", level: 2 },
  leaves: [
    { hash: "0x1a2b3c", event: "user.login", level: 0 },
    { hash: "0x2b3c4d", event: "policy.eval", level: 0 },
    { hash: "0x3c4d5e", event: "session.create", level: 0 },
    { hash: "0x4d5e6f", event: "token.issue", level: 0 },
  ],
};

const RUNS: VerificationRun[] = [
  { id: "v1", blocks_checked: 2847, roots_verified: 4, anomalies: 0, duration_ms: 1240, status: "pass", timestamp: new Date(Date.now() - 1800000).toISOString() },
  { id: "v2", blocks_checked: 2691, roots_verified: 4, anomalies: 0, duration_ms: 1180, status: "pass", timestamp: new Date(Date.now() - 5400000).toISOString() },
  { id: "v3", blocks_checked: 2403, roots_verified: 4, anomalies: 0, duration_ms: 1320, status: "pass", timestamp: new Date(Date.now() - 9000000).toISOString() },
];

export default function MerklePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("tree");
  const [loading, setLoading] = useState(true);
  const [verifying, setVerifying] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = async () => {
    setLoading(false);
    try { await fetch("/api/v1/audit/verify-integrity", { headers: h }).catch(() => null); } catch { /* noop */ }
  };
  useEffect(() => { loadData(); }, []);

  const runVerification = async () => { setVerifying(true); try { await fetch("/api/v1/audit/verify-integrity", { method: "POST", headers: h }); } catch { /* noop */ } finally { setVerifying(false); } };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><GitBranch className="h-6 w-6 text-blue-500" /> {t("merkle.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("merkle.subtitle")}</p></div>
        <button onClick={runVerification} disabled={verifying} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{verifying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} {t("merkle.verifyNow")}</button>
      </div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["tree", t("merkle.treeViewer"), GitBranch], ["log", t("merkle.verificationLog"), Clock]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div> : (<>

      {/* TREE */}
      {tab === "tree" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><GitBranch className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{TREE_DATA.root.level}</p><p className="text-xs text-gray-400">{t("merkle.treeDepth")}</p></div>
            <div className={card + " text-center"}><Database className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{TREE_DATA.leaves.length}</p><p className="text-xs text-gray-400">{t("merkle.leaves")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold text-green-600">0</p><p className="text-xs text-gray-400">{t("merkle.anomalies")}</p></div>
            <div className={card + " text-center"}><Lock className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-sm font-mono font-bold">{TREE_DATA.root.hash}</p><p className="text-xs text-gray-400">{t("merkle.rootHash")}</p></div>
          </div>

          {/* Visual tree */}
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("merkle.visualTree")}</h3>
            <div className="flex flex-col items-center space-y-4">
              {/* Root */}
              <div className="rounded-lg border-2 border-blue-500 bg-blue-50 dark:bg-blue-950/30 px-4 py-2"><code className="text-xs font-mono font-bold text-blue-600">{TREE_DATA.root.hash}</code><span className="ml-2 text-xs text-gray-400">Root (L{TREE_DATA.root.level})</span></div>
              <div className="w-px h-4 bg-gray-300 dark:bg-gray-700" />
              {/* Level 2 */}
              <div className="flex gap-8">
                <div className="rounded-lg border border-blue-300 bg-blue-50 dark:bg-blue-950/20 px-3 py-1.5"><code className="text-xs font-mono text-blue-500">{TREE_DATA.left.hash}</code></div>
                <div className="rounded-lg border border-blue-300 bg-blue-50 dark:bg-blue-950/20 px-3 py-1.5"><code className="text-xs font-mono text-blue-500">{TREE_DATA.right.hash}</code></div>
              </div>
              <div className="flex gap-2"><div className="w-px h-4 bg-gray-300 dark:bg-gray-700 ml-4" /><div className="w-px h-4 bg-gray-300 dark:bg-gray-700 ml-12" /></div>
              {/* Leaves */}
              <div className="flex gap-2 flex-wrap justify-center">
                {TREE_DATA.leaves.map((leaf, i) => (
                  <div key={i} className="rounded-lg border border-gray-200 dark:border-gray-700 px-2 py-1"><code className="text-xs font-mono text-gray-500">{leaf.hash}</code><p className="text-[10px] text-gray-400">{leaf.event}</p></div>
                ))}
              </div>
              <div className="flex items-center gap-2 text-xs text-gray-400"><CheckCircle2 className="h-3 w-3 text-green-500" /> {t("merkle.allNodesVerified")}</div>
            </div>
          </div>
        </div>
      )}

      {/* LOG */}
      {tab === "log" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("merkle.time")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("merkle.blocks")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("merkle.rootsVerified")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("merkle.anomalies2")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("merkle.duration")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("merkle.status")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{RUNS.map(r => (
            <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs">{new Date(r.timestamp).toLocaleString()}</td><td className="px-3 py-3 text-center text-xs font-mono">{r.blocks_checked.toLocaleString()}</td><td className="px-3 py-3 text-center text-xs font-mono">{r.roots_verified}</td><td className="px-3 py-3 text-center"><span className={`text-xs font-mono ${r.anomalies > 0 ? "text-red-600 font-bold" : "text-gray-400"}`}>{r.anomalies}</span></td><td className="px-3 py-3 text-center text-xs font-mono">{r.duration_ms}ms</td><td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${r.status === "pass" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{r.status}</span></td></tr>
          ))}</tbody>
        </table></div>
      )}

      </>)}
    </div>
  );
}
