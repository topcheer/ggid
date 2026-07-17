"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, CheckCircle, XCircle,
  Globe, Database, Lock, FileText, AlertTriangle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ComplianceItem {
  id: string;
  law: string;
  article: string;
  requirement: string;
  status: "compliant" | "partial" | "non_compliant";
  data_residency: string;
  encryption: boolean;
  audit_log: boolean;
  user_consent: boolean;
  right_to_delete: boolean;
  last_audit: string;
}

const statusConfig: Record<string, { color: string; icon: typeof CheckCircle; label: string }> = {
  compliant: { color: "text-green-600", icon: CheckCircle, label: "Compliant" },
  partial: { color: "text-yellow-600", icon: AlertTriangle, label: "Partial" },
  non_compliant: { color: "text-red-600", icon: XCircle, label: "Non-Compliant" },
};

export default function DataSecurityDashboard() {
  const t = useTranslations();
  const [items, setItems] = useState<ComplianceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/audit/compliance/data-security", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setItems(d.items || d.compliance || []);
      }
    } catch { setError("Failed to load compliance data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const compliant = items.filter(i => i.status === "compliant").length;
  const partial = items.filter(i => i.status === "partial").length;
  const nonCompliant = items.filter(i => i.status === "non_compliant").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="h-6 w-6 text-emerald-500" />
            Data Security Law Compliance
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Multi-jurisdiction data protection compliance: GDPR, PIPL, CCPA, ISO 27001.
          </p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh compliance" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Overview stats */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <div className={cardCls}><div className="flex items-center gap-2"><CheckCircle className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Compliant</span></div><p className="mt-2 text-2xl font-bold text-green-600">{compliant}</p></div>
        <div className={cardCls}><div className="flex items-center gap-2"><AlertTriangle className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">Partial</span></div><p className="mt-2 text-2xl font-bold text-yellow-600">{partial}</p></div>
        <div className={cardCls}><div className="flex items-center gap-2"><XCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Non-Compliant</span></div><p className="mt-2 text-2xl font-bold text-red-600">{nonCompliant}</p></div>
        <div className={cardCls}><div className="flex items-center gap-2"><Globe className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Jurisdictions</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{new Set(items.map(i => i.law)).size}</p></div>
      </div>

      {/* Compliance items */}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-500" /></div> : items.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Shield className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No compliance data available.</p><p className="mt-1 text-xs text-gray-400">Backend endpoint /api/v1/audit/compliance/data-security may not be implemented yet.</p></div></div>
      ) : (
        <div className="space-y-3">
          {items.map(item => {
            const cfg = statusConfig[item.status] || statusConfig.partial;
            const StatusIcon = cfg.icon;
            return (
              <div key={item.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-gray-900 dark:text-white">{item.law}</span>
                      <span className="px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400">{item.article}</span>
                      <span className={"flex items-center gap-1 text-xs font-medium " + cfg.color}><StatusIcon className="h-3.5 w-3.5" /> {cfg.label}</span>
                    </div>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{item.requirement}</p>
                    <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-5">
                      <div className="flex items-center gap-1.5 text-xs"><Globe className="h-3.5 w-3.5 text-gray-400" /><span className="text-gray-500">Residency:</span><span className="font-medium">{item.data_residency || "—"}</span></div>
                      <div className="flex items-center gap-1.5 text-xs"><Lock className={"h-3.5 w-3.5 " + (item.encryption ? "text-green-500" : "text-red-500")} /><span className={item.encryption ? "text-green-600" : "text-red-600"}>Encryption</span></div>
                      <div className="flex items-center gap-1.5 text-xs"><FileText className={"h-3.5 w-3.5 " + (item.audit_log ? "text-green-500" : "text-red-500")} /><span className={item.audit_log ? "text-green-600" : "text-red-600"}>Audit Log</span></div>
                      <div className="flex items-center gap-1.5 text-xs"><CheckCircle className={"h-3.5 w-3.5 " + (item.user_consent ? "text-green-500" : "text-red-500")} /><span className={item.user_consent ? "text-green-600" : "text-red-600"}>Consent</span></div>
                      <div className="flex items-center gap-1.5 text-xs"><XCircle className={"h-3.5 w-3.5 " + (item.right_to_delete ? "text-green-500" : "text-red-500")} /><span className={item.right_to_delete ? "text-green-600" : "text-red-600"}>Delete Right</span></div>
                    </div>
                    {item.last_audit && <p className="mt-2 text-xs text-gray-400">Last audit: {new Date(item.last_audit).toLocaleDateString()}</p>}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
