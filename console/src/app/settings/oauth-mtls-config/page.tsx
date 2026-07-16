"use client";
import { useTranslations } from "@/lib/i18n";

import { useOAuthMtlsConfig } from "@ggid/sdk-react";
import { Lock, Shield, Award, AlertTriangle, CheckCircle } from "lucide-react";

export default function OAuthMtlsConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useOAuthMtlsConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading mTLS config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">mTLS Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Mutual TLS for OAuth client authentication</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Config Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Lock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Require mTLS</span>
          </div>
          <p className="text-lg font-bold">{data?.require_mtls ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Award className="w-4 h-4" />
            <span className="text-xs text-gray-400">Trusted CAs</span>
          </div>
          <p className="text-lg font-bold">{data?.trusted_ca_certs?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">Revocation Check</span>
          </div>
          <p className="text-sm font-bold uppercase">{data?.certificate_revocation_check ?? "CRL"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">mTLS Adoption</span>
          </div>
          <p className="text-lg font-bold text-green-400">{data?.mtls_adoption_pct ?? 0}%</p>
        </div>
      </div>

      {/* Allow Self-Signed Banner */}
      <div className={"rounded-xl p-4 mb-6 flex items-center gap-3 " + (
        data?.allow_self_signed ? "bg-red-900/30 border border-red-800" : "bg-green-900/30 border border-green-800"
      )}>
        <AlertTriangle className={"w-5 h-5 " + (data?.allow_self_signed ? "text-red-400" : "text-green-400")} />
        <span className="text-sm">Self-signed certificates are {data?.allow_self_signed ? "allowed (NOT RECOMMENDED for production)" : "not allowed"}</span>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Trusted CA Certs */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Award className="w-5 h-5 text-blue-400" />
            Trusted CA Certificates
          </h2>
          <div className="space-y-2">
            {(data?.trusted_ca_certs ?? []).map((ca) => (
              <div key={ca.name} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{ca.name}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    ca.status === "valid" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300"
                  )}>
                    {ca.status}
                  </span>
                </div>
                <p className="text-xs text-gray-400 font-mono">{ca.fingerprint}</p>
                <p className="text-xs text-gray-500 mt-1">Expires: {ca.expiry}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Per-Client mTLS */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Per-Client mTLS</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th scope="col" className="text-left py-2 pr-3">Client</th>
                  <th scope="col" className="text-left py-2 pr-3">Required</th>
                  <th scope="col" className="text-left py-2 pr-3">Thumbprint Binding</th>
                </tr>
              </thead>
              <tbody>
                {(data?.per_client_mtls ?? []).map((c) => (
                  <tr key={c.client} className="border-b border-gray-800">
                    <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.client}</td>
                    <td className="py-3 pr-3">
                      <span className={"text-xs px-2 py-0.5 rounded " + (c.required ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                        {c.required ? "Yes" : "No"}
                      </span>
                    </td>
                    <td className="py-3 pr-3 font-mono text-xs text-gray-300">{c.cert_thumbprint_binding ?? "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
