"use client";
import { useState } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, Download, Trash2, Check,
  Database, FileText, Activity, Lock, Clock, AlertTriangle,
  ChevronRight, Eye, Ban,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

type Tab = "myData" | "export" | "delete";

const DATA_CATEGORIES = [
  { category: "Identity", fields: ["email", "display_name", "phone", "avatar_url"], retained: "Account lifetime", encrypted: true },
  { category: "Authentication", fields: ["password_hash", "mfa_secrets", "passkey_credentials", "oauth_tokens"], retained: "Account lifetime", encrypted: true },
  { category: "Sessions", fields: ["active_sessions", "device_fingerprints", "login_history"], retained: "90 days", encrypted: true },
  { category: "Audit Log", fields: ["action_logs", "policy_decisions", "risk_scores"], retained: "7 years (compliance)", encrypted: true },
  { category: "Organizational", fields: ["department", "manager", "title", "roles", "group_memberships"], retained: "Account lifetime", encrypted: false },
];

export default function PrivacyPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("myData");
  const [exporting, setExporting] = useState(false);
  const [exportRequested, setExportRequested] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState("");
  const [deleting, setDeleting] = useState(false);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const requestExport = async () => {
    setExporting(true);
    try { await fetch("/api/v1/audit/dsr", { method: "POST", headers: H, body: JSON.stringify({ type: "portability", user_id: "me" }) }); setExportRequested(true); }
    catch { /* noop */ }
    finally { setExporting(false); }
  };

  const deleteAccount = async () => {
    if (deleteConfirm !== "DELETE") return;
    setDeleting(true);
    try { await fetch("/api/v1/audit/dsr", { method: "POST", headers: H, body: JSON.stringify({ type: "erasure", user_id: "me" }) }); setConfirmDelete(false); }
    catch { /* noop */ }
    finally { setDeleting(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-purple-500" /> {t("privacy.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("privacy.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["myData", t("privacy.myData"), Database], ["export", t("privacy.dataExport"), Download], ["delete", t("privacy.deleteAccount"), Trash2]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-purple-600 text-purple-600 dark:text-purple-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* MY DATA */}
      {tab === "myData" && (
        <div className="space-y-4">
          <div className={`${card} bg-purple-50 dark:bg-purple-900/10`}><div className="flex items-center gap-3"><Eye className="h-5 w-5 text-purple-500" /><div><p className="text-sm font-medium">{t("privacy.transparency")}</p><p className="text-xs text-gray-400">{t("privacy.transparencyDesc")}</p></div></div></div>
          {DATA_CATEGORIES.map(cat => (
            <div key={cat.category} className={card}>
              <div className="flex items-center justify-between mb-3"><div className="flex items-center gap-2"><Database className="h-4 w-4 text-purple-400" /><h3 className="text-sm font-semibold">{cat.category}</h3></div><div className="flex items-center gap-2">{cat.encrypted && <span className="flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 text-green-600"><Lock className="h-2.5 w-2.5" /> {t("privacy.encrypted")}</span>}</div></div>
              <div className="flex flex-wrap gap-1 mb-2">{cat.fields.map(f => <code key={f} className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{f}</code>)}</div>
              <p className="text-xs text-gray-400">{t("privacy.retention")}: <span className="font-medium">{cat.retained}</span></p>
            </div>
          ))}
        </div>
      )}

      {/* EXPORT */}
      {tab === "export" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Download className="h-4 w-4" /> {t("privacy.requestExport")}</h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{t("privacy.exportDesc")}</p>
            <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3 text-xs text-blue-600 dark:text-blue-400 mb-4"><Clock className="inline h-3 w-3" /> {t("privacy.exportSla")}</div>
            {exportRequested ? (
              <div className="rounded-lg border-2 border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30 p-4"><div className="flex items-center gap-2"><Check className="h-5 w-5 text-green-500" /><span className="text-sm font-medium text-green-700 dark:text-green-400">{t("privacy.exportRequested")}</span></div><p className="mt-1 text-xs text-green-600 dark:text-green-500">{t("privacy.exportReadySoon")}</p></div>
            ) : (
              <button onClick={requestExport} disabled={exporting} className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />} {t("privacy.downloadData")}</button>
            )}
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("privacy.exportFormat")}</h3>
            <div className="space-y-2 text-xs text-gray-500"><p className="flex items-center gap-2"><Check className="h-3 w-3 text-green-500" /> {t("privacy.format1")}</p><p className="flex items-center gap-2"><Check className="h-3 w-3 text-green-500" /> {t("privacy.format2")}</p><p className="flex items-center gap-2"><Check className="h-3 w-3 text-green-500" /> {t("privacy.format3")}</p><p className="flex items-center gap-2"><Check className="h-3 w-3 text-green-500" /> {t("privacy.format4")}</p></div>
          </div>
        </div>
      )}

      {/* DELETE */}
      {tab === "delete" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Trash2 className="h-4 w-4 text-red-500" /> {t("privacy.deleteTitle")}</h3>
            <div className="rounded-lg bg-red-50 dark:bg-red-900/20 p-3 mb-4"><AlertTriangle className="inline h-4 w-4 text-red-500" /> <span className="text-xs text-red-600 dark:text-red-400">{t("privacy.deleteWarning")}</span></div>
            <div className="space-y-2 text-xs text-gray-500 mb-4"><p className="flex items-center gap-2"><Ban className="h-3 w-3 text-red-400" /> {t("privacy.deleteConsequence1")}</p><p className="flex items-center gap-2"><Ban className="h-3 w-3 text-red-400" /> {t("privacy.deleteConsequence2")}</p><p className="flex items-center gap-2"><Ban className="h-3 w-3 text-red-400" /> {t("privacy.deleteConsequence3")}</p></div>
            <button onClick={() => setConfirmDelete(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Trash2 className="h-4 w-4" /> {t("privacy.requestDeletion")}</button>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("privacy.auditRetention")}</h3>
            <p className="text-xs text-gray-500 dark:text-gray-400">{t("privacy.auditRetentionDesc")}</p>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-red-500" /><h3 className="text-lg font-semibold text-red-600">{t("privacy.confirmDelete")}</h3></div>
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{t("privacy.confirmDeleteDesc")}</p>
            <div className="mt-3"><label className="text-sm font-medium">{t("privacy.typeDelete")}</label><input type="text" value={deleteConfirm} onChange={e => setDeleteConfirm(e.target.value)} placeholder="DELETE" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setConfirmDelete(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={deleteAccount} disabled={deleteConfirm !== "DELETE" || deleting} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{deleting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />} {t("privacy.confirm")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
