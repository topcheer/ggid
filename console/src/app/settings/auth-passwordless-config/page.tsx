"use client";

import { useAuthPasswordlessConfig } from "@ggid/sdk-react";
import { Mail, Key, Fingerprint, ScanFace, ShieldCheck, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthPasswordlessConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAuthPasswordlessConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading passwordless config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const methodIcons: Record<string, React.ReactNode> = {
    magic_link: <Mail className="w-5 h-5" />,
    passkey: <Key className="w-5 h-5" />,
    webauthn: <ShieldCheck className="w-5 h-5" />,
    biometric: <ScanFace className="w-5 h-5" />,
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Passwordless Authentication</h1>
          <p className="text-sm text-gray-400 mt-1">Configure passwordless login methods and fallback policies</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Enabled Methods */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Enabled Methods</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {(data?.enabled_methods ?? []).map((m: any) => (
            <div key={m.method} className="bg-gray-800 rounded-lg p-4 flex items-center gap-3">
              <div className={"w-10 h-10 rounded-lg flex items-center justify-center " + (
                m.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-500"
              )}>
                {methodIcons[m.method] ?? <Key className="w-5 h-5" />}
              </div>
              <div className="flex-1">
                <p className="text-sm font-semibold capitalize">{m.method.replace(/_/g, " ")}</p>
                <p className="text-xs text-gray-400">{m.description}</p>
              </div>
              <span
                className={"text-xs px-2 py-0.5 rounded " + (
                  m.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                )}
              >
                {m.enabled ? "ON" : "OFF"}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Configuration Parameters */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Mail className="w-4 h-4" />
            <span className="text-xs text-gray-400">Magic Link Expiry</span>
          </div>
          <p className="text-xl font-bold">{data?.magic_link_expiry_minutes ?? 0} min</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Key className="w-4 h-4" />
            <span className="text-xs text-gray-400">Passkey RP ID</span>
          </div>
          <p className="text-sm font-mono text-purple-300">{data?.passkey_rp_id ?? "N/A"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <ShieldCheck className="w-4 h-4" />
            <span className="text-xs text-gray-400">WebAuthn Timeout</span>
          </div>
          <p className="text-xl font-bold">{data?.webauthn_timeout_seconds ?? 0}s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Fallback to Password</span>
          </div>
          <p className="text-xl font-bold">{data?.fallback_to_password ? "Yes" : "No"}</p>
        </div>
      </div>

      {/* Per-Role Requirement Matrix */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Per-Role Passwordless Requirements</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-4">Role</th>
                <th scope="col" className="text-left py-2 pr-4">Required Method</th>
                <th scope="col" className="text-left py-2 pr-4">Enforcement</th>
                <th scope="col" className="text-left py-2 pr-4">Grace Period</th>
              </tr>
            </thead>
            <tbody>
              {(data?.per_role_requirement ?? []).map((req: any) => (
                <tr key={req.role} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-medium">{req.role}</td>
                  <td className="py-3 pr-4">
                    <span className="text-xs px-2 py-0.5 rounded bg-blue-900 text-blue-300">{req.required_method}</span>
                  </td>
                  <td className="py-3 pr-4">
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        req.enforcement === "required" ? "bg-red-900 text-red-300" :
                        req.enforcement === "recommended" ? "bg-yellow-900 text-yellow-300" :
                        "bg-green-900 text-green-300"
                      )}
                    >
                      {req.enforcement}
                    </span>
                  </td>
                  <td className="py-3 pr-4 text-gray-300">{req.grace_period_days > 0 ? `${req.grace_period_days} days` : "Immediate"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
