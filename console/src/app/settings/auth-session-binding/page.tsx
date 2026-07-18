"use client";

import { useAuthSessionBinding } from "@ggid/sdk-react";
import { KeyRound, Shield, RefreshCw, Smartphone, Cookie } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthSessionBindingPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAuthSessionBinding();

  if (loading) return <div className="p-8 text-gray-400">Loading session binding...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Session Binding</h1>
          <p className="text-sm text-gray-400 mt-1">Configure how sessions are bound to clients and devices</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Binding Method & Policy */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <KeyRound className="w-4 h-4" />
            <span className="text-xs text-gray-400">Binding Method</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.binding_method}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">Hijack Protection</span>
          </div>
          <p className="text-lg font-bold">{data?.session_hijack_protection ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Cookie className="w-4 h-4" />
            <span className="text-xs text-gray-400">Fallback Method</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.fallback_method ?? "none"}</p>
        </div>
      </div>

      {/* Per-Application Binding Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Per-Application Binding</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-4">Application</th>
                <th scope="col" className="text-left py-2 pr-4">Method</th>
                <th scope="col" className="text-left py-2 pr-4">Enforced</th>
                <th scope="col" className="text-left py-2 pr-4">Rotation Policy</th>
              </tr>
            </thead>
            <tbody>
              {(data?.per_application_binding ?? []).map((app: any) => (
                <tr key={app.app} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-medium">{app.app}</td>
                  <td className="py-3 pr-4 capitalize">{app.method}</td>
                  <td className="py-3 pr-4">
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        app.enforce ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                      )}
                    >
                      {app.enforce ? "Yes" : "No"}
                    </span>
                  </td>
                  <td className="py-3 pr-4 text-gray-300 text-xs">{data?.binding_rotation_policy ?? "every 90 days"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Cross-Device Session Transfer */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Smartphone className="w-5 h-5 text-blue-400" />
          Cross-Device Session Transfer
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
            <span className="text-sm text-gray-300">Enabled</span>
            <span className={"text-sm font-medium " + (data?.cross_device_session_transfer?.enabled ? "text-green-400" : "text-red-400")}>
              {data?.cross_device_session_transfer?.enabled ? "Yes" : "No"}
            </span>
          </div>
          <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
            <span className="text-sm text-gray-300">Transfer Window</span>
            <span className="text-sm font-medium">{data?.cross_device_session_transfer?.transfer_window_seconds ?? 0}s</span>
          </div>
          <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
            <span className="text-sm text-gray-300">Verification Required</span>
            <span className="text-sm font-medium">{data?.cross_device_session_transfer?.verification_required ? "Yes" : "No"}</span>
          </div>
          <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
            <span className="text-sm text-gray-300">Max Transfers / Day</span>
            <span className="text-sm font-medium">{data?.cross_device_session_transfer?.max_per_day ?? 0}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
