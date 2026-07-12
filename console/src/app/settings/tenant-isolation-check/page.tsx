"use client";

import { useTenantIsolationCheck } from "@ggid/sdk-react";
import { ShieldCheck, AlertTriangle, Bug } from "lucide-react";

export default function TenantIsolationCheckPage() {
  const { data, loading, error, refresh } = useTenantIsolationCheck();

  if (loading) return <div className="p-8 text-gray-400">Loading tenant isolation checks...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Tenant Isolation Check</h1>
          <p className="text-sm text-gray-400 mt-1">Verify multi-tenant data isolation and access controls</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Re-run All Tests</button>
      </div>

      {/* Compliance Status Banner */}
      <div className={"rounded-xl p-6 mb-6 " + (data?.compliance_status === "passing" ? "bg-green-950 border border-green-800" : "bg-red-950 border border-red-800")}>
        <div className="flex items-center gap-3">
          {data?.compliance_status === "passing" ? <ShieldCheck className="w-8 h-8 text-green-400" /> : <AlertTriangle className="w-8 h-8 text-red-400" />}
          <div>
            <h2 className="text-lg font-semibold">{data?.compliance_status === "passing" ? "All Isolation Tests Passing" : "Isolation Issues Detected"}</h2>
            <p className="text-sm text-gray-400">{data?.tests?.filter((t) => t.status === "pass").length ?? 0}/{data?.tests?.length ?? 0} tests passed</p>
          </div>
        </div>
      </div>

      {/* Isolation Tests */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Isolation Tests</h2>
        <div className="space-y-3">
          {(data?.tests ?? []).map((t) => (
            <div key={t.test_name} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-center gap-3 mb-2">
                {t.status === "pass" ? <ShieldCheck className="w-4 h-4 text-green-400" /> : <AlertTriangle className="w-4 h-4 text-red-400" />}
                <h3 className="text-sm font-medium">{t.test_name}</h3>
                <span className={"text-xs px-2 py-0.5 rounded ml-auto " + (t.status === "pass" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>{t.status}</span>
              </div>
              <p className="text-xs text-gray-400 mb-2">{t.evidence}</p>
              {t.status !== "pass" && t.remediation && (
                <div className="bg-gray-900 rounded p-2 mt-2">
                  <p className="text-xs text-yellow-400">Remediation: {t.remediation}</p>
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
            {data.cross_tenant_access_log.map((log) => (
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
          <h2 className="text-sm font-semibold mb-3">Row-Level Security Validation</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {data.rls_validation.map((r) => (
              <div key={r.table} className="bg-gray-800 rounded-lg p-3 flex items-center justify-between">
                <span className="text-sm font-mono">{r.table}</span>
                <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                  RLS {r.enabled ? "ON" : "OFF"}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
