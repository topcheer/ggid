"use client";

import { useOAuthCrossTenant } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ArrowLeftRight, Plus, Shield, Clock, AlertTriangle } from "lucide-react";

export default function OAuthCrossTenantPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useOAuthCrossTenant();

  if (loading) return <div className="p-8 text-gray-400">Loading cross-tenant config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Cross-Tenant Trust</h1>
          <p className="text-sm text-gray-400 mt-1">Manage trust relationships between tenants</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
            <Plus className="w-4 h-4" />
            Add Trust
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <ArrowLeftRight className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Trusts</span>
          </div>
          <p className="text-2xl font-bold">{data?.trusted_tenants?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">Revocation Policy</span>
          </div>
          <p className="text-sm font-bold capitalize">{data?.revocation_policy?.type ?? "immediate"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Audit Entries (30d)</span>
          </div>
          <p className="text-2xl font-bold">{data?.audit_trail?.length ?? 0}</p>
        </div>
      </div>

      {/* Trusted Tenants */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Trusted Tenants</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Tenant ID</th>
                <th scope="col" className="text-left py-2 pr-3">Direction</th>
                <th scope="col" className="text-left py-2 pr-3">Scopes Allowed</th>
                <th scope="col" className="text-left py-2 pr-3">Expires</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.trusted_tenants ?? []).map((t) => (
                <tr key={t.tenant_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{t.tenant_id}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      t.trust_direction === "bidirectional" ? "bg-purple-900 text-purple-300" :
                      t.trust_direction === "inbound" ? "bg-blue-900 text-blue-300" :
                      "bg-green-900 text-green-300"
                    )}>
                      {t.trust_direction}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex flex-wrap gap-1">
                      {t.scopes_allowed.map((s) => (
                        <span key={s} className="text-xs px-1.5 py-0.5 bg-gray-800 rounded">{s}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{t.expires_at}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs px-2 py-0.5 rounded bg-green-900 text-green-300">Active</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-App Sharing */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Per-App Sharing</h2>
          <div className="space-y-2">
            {(data?.per_app_sharing ?? []).map((app) => (
              <div key={app.app_id} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{app.app_name}</p>
                  <p className="text-xs text-gray-400">{app.shared_with_tenant_ids.length} tenants</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (app.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                  {app.enabled ? "Shared" : "Private"}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Audit Trail */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Audit Trail</h2>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {(data?.audit_trail ?? []).map((a, i) => (
              <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-2">
                {a.action === "trust_created" ? <Plus className="w-3 h-3 text-green-400" /> :
                 a.action === "trust_revoked" ? <AlertTriangle className="w-3 h-3 text-red-400" /> :
                 <Shield className="w-3 h-3 text-blue-400" />}
                <div className="flex-1">
                  <p className="text-xs font-medium">{a.action}</p>
                  <p className="text-xs text-gray-500">{a.actor}</p>
                </div>
                <span className="text-xs text-gray-500">{a.timestamp}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
