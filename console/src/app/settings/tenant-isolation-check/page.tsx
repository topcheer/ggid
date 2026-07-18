"use client";

import { useTenantIsolationCheck } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ShieldCheck, AlertTriangle, Bug } from "lucide-react";

export default function TenantIsolationCheckPage() {
  const { data, loading, error, refresh } = useTenantIsolationCheck();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("tenantIsolationCheck.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("tenantIsolationCheck.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("tenantIsolationCheck.subtitle")}</p>
        </div>
        <button aria-label="action" onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("tenantIsolationCheck.rerunAll")}</button>
      </div>

      {/* Compliance Status Banner */}
      <div className={"rounded-xl p-6 mb-6 " + (data?.compliance_status === "passing" ? "bg-green-950 border border-green-800" : "bg-red-950 border border-red-800")}>
        <div className="flex items-center gap-3">
          {data?.compliance_status === "passing" ? <ShieldCheck className="w-8 h-8 text-green-400" /> : <AlertTriangle className="w-8 h-8 text-red-400" />}
          <div>
            <h2 className="text-lg font-semibold">{data?.compliance_status === "passing" ? t("tenantIsolationCheck.allPassing") : t("tenantIsolationCheck.issuesDetected")}</h2>
            <p className="text-sm text-gray-400">{data?.tests?.filter((t: any) => t.status === "pass").length ?? 0}/{data?.tests?.length ?? 0} tests passed</p>
          </div>
        </div>
      </div>

      {/* Isolation Tests */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">{t("tenantIsolationCheck.tests")}</h2>
        <div className="space-y-3">
          {(data?.tests ?? []).map((test: any) => (
            <div key={test.test_name} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-center gap-3 mb-2">
                {test.status === "pass" ? <ShieldCheck className="w-4 h-4 text-green-400" /> : <AlertTriangle className="w-4 h-4 text-red-400" />}
                <h3 className="text-sm font-medium">{test.test_name}</h3>
                <span className={"text-xs px-2 py-0.5 rounded ml-auto " + (test.status === "pass" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>{test.status}</span>
              </div>
              <p className="text-xs text-gray-400 mb-2">{test.evidence}</p>
              {test.status !== "pass" && test.remediation && (
                <div className="bg-gray-900 rounded p-2 mt-2">
                  <p className="text-xs text-yellow-400">{t("tenantIsolationCheck.remediation")} {test.remediation}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Cross-Tenant Access Log */}
      {data?.cross_tenant_access_log && data.cross_tenant_access_log.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6 mb-6">
          <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Bug className="w-4 h-4 text-red-400" /> Cross-Tenant Access Attempts</h2>
          <div className="space-y-1">
            {data.cross_tenant_access_log.map((log: any) => (
              <div key={log.id} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
                <span className="text-gray-500">{log.timestamp}</span>
                <span className="text-gray-300">User {log.user_id}</span>
                <span className="text-gray-600">attempted access to tenant</span>
                <span className="font-mono text-red-400">{log.target_tenant}</span>
                <span className="ml-auto px-1.5 py-0.5 rounded bg-red-900 text-red-300">{log.action}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* RLS Validation */}
      {data?.rls_validation && data.rls_validation.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">{t("tenantIsolationCheck.rlsValidation")}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {data.rls_validation.map((r: any) => (
              <div key={r.table} className="bg-gray-800 rounded-lg p-3 flex items-center justify-between">
                <span className="text-sm font-mono">{r.table}</span>
                <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                  RLS {r.enabled ? t("tenantIsolationCheck.on") : t("tenantIsolationCheck.off")}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
