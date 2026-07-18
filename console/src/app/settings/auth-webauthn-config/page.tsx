"use client";

import { useAuthWebauthnConfig } from "@ggid/sdk-react";
import { Fingerprint, Key, Shield, CheckCircle, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthWebauthnConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAuthWebauthnConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading WebAuthn config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">WebAuthn Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Configure passkey and WebAuthn authenticator settings</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* RP Settings */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Fingerprint className="w-5 h-5 text-blue-400" />
          Relying Party Settings
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label className="text-xs text-gray-400 mb-1 block">RP ID</label>
            <p className="text-sm font-mono bg-gray-800 rounded-lg px-3 py-2 border border-gray-700">{data?.rp_id}</p>
          </div>
          <div>
            <label className="text-xs text-gray-400 mb-1 block">RP Name</label>
            <p className="text-sm font-mono bg-gray-800 rounded-lg px-3 py-2 border border-gray-700">{data?.rp_name}</p>
          </div>
          <div>
            <label className="text-xs text-gray-400 mb-1 block">Origin</label>
            <p className="text-sm font-mono bg-gray-800 rounded-lg px-3 py-2 border border-gray-700">{data?.origin}</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Attestation & Verification */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Shield className="w-5 h-5 text-green-400" />
            Security Policy
          </h2>
          <div className="space-y-3">
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">Attestation Requirement</span>
              <span className="text-sm font-medium text-blue-400 capitalize">{data?.attestation_requirement}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">User Verification</span>
              <span className="text-sm font-medium text-blue-400 capitalize">{data?.user_verification}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300 flex items-center gap-1">
                <Clock className="w-3 h-3" />
                Timeout
              </span>
              <span className="text-sm font-medium">{data?.timeout_seconds ?? 0}s</span>
            </div>
          </div>
        </div>

        {/* Supported Algorithms */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Key className="w-5 h-5 text-purple-400" />
            Supported Algorithms
          </h2>
          <div className="space-y-2">
            {(data?.supported_algs ?? []).map((alg: any) => (
              <div key={alg.name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div className="flex items-center gap-2">
                  <CheckCircle className="w-4 h-4 text-green-400" />
                  <span className="text-sm font-mono font-medium">{alg.name}</span>
                </div>
                <span className="text-xs text-gray-400">COSE ID: {alg.cose_id}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Per-Platform Config */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">Per-Platform Configuration</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-4">Platform</th>
                <th scope="col" className="text-left py-2 pr-4">Authenticator Type</th>
                <th scope="col" className="text-left py-2 pr-4">Attachment</th>
                <th scope="col" className="text-left py-2 pr-4">Discoverable</th>
              </tr>
            </thead>
            <tbody>
              {(data?.per_platform_config ?? []).map((p: any) => (
                <tr key={p.platform} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-medium capitalize">{p.platform}</td>
                  <td className="py-3 pr-4 text-gray-300 capitalize">{p.authenticator_type.replace(/_/g, " ")}</td>
                  <td className="py-3 pr-4 text-gray-300 capitalize">{p.attachment}</td>
                  <td className="py-3 pr-4">
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        p.discoverable_credentials ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                      )}
                    >
                      {p.discoverable_credentials ? "Yes" : "No"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
