"use client";

import { useConsentManagement } from "@ggid/sdk-react";
import { CheckCircle, XCircle, FileText, Users } from "lucide-react";

export default function ConsentManagementPage() {
  const { data, loading, error, refresh, withdrawConsent } = useConsentManagement();

  if (loading) return <div className="p-8 text-gray-400">Loading consent management...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Consent Management</h1>
          <p className="text-sm text-gray-400 mt-1">Track user consent across purposes and regions</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Region Compliance */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        {(data?.per_region_compliance ?? []).map((r) => (
          <div key={r.region} className="bg-gray-900 rounded-xl p-4">
            <p className="text-xs text-gray-400 mb-1">{r.region}</p>
            <p className="text-xl font-bold text-green-400">{r.compliance_pct}%</p>
            <p className="text-xs text-gray-500 mt-0.5">{r.active_consents} active consents</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Consent Registry */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Users className="w-4 h-4 text-blue-400" />
            Consent Registry
          </h2>
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {(data?.user_consent_registry ?? []).map((c, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between mb-1">
                  <div>
                    <p className="text-sm font-medium">{c.user}</p>
                    <p className="text-xs text-gray-400">{c.purpose}</p>
                  </div>
                  {c.withdrawn_at ? (
                    <XCircle className="w-4 h-4 text-red-400" />
                  ) : (
                    <CheckCircle className="w-4 h-4 text-green-400" />
                  )}
                </div>
                <div className="flex items-center gap-3 text-xs text-gray-500">
                  <span>Granted: {c.granted_at}</span>
                  {!c.withdrawn_at && c.expires_at && <span>Expires: {c.expires_at}</span>}
                  {c.withdrawn_at && <span className="text-red-400">Withdrawn: {c.withdrawn_at}</span>}
                </div>
                {!c.withdrawn_at && (
                  <button onClick={() => withdrawConsent(c.user, c.purpose)} className="text-xs text-red-400 hover:text-red-300 mt-1">Withdraw</button>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Consent Templates */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <FileText className="w-4 h-4 text-purple-400" />
            Consent Templates
          </h2>
          <div className="space-y-2">
            {(data?.consent_templates ?? []).map((t, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-1">
                  <p className="text-sm font-medium">{t.purpose}</p>
                  <span className={"text-xs px-1.5 py-0.5 rounded " + (t.required ? "bg-red-900 text-red-300" : "bg-gray-700 text-gray-400")}>
                    {t.required ? "Required" : "Optional"}
                  </span>
                </div>
                <p className="text-xs text-gray-400 mb-1">{t.purpose_text}</p>
                <p className="text-xs text-gray-500">Legal basis: {t.legal_basis}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
