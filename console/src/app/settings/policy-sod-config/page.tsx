"use client";

import { usePolicySoDConfig } from "@ggid/sdk-react";
import { Shield, AlertTriangle, CheckCircle, Lock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicySoDConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = usePolicySoDConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading SoD config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Segregation of Duties</h1>
          <p className="text-sm text-gray-400 mt-1">Detect and prevent conflicting role assignments</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Config Toggles */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">Auto-Enforce</span>
          </div>
          <p className="text-lg font-bold">{data?.auto_enforce ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Lock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Bypass Requires C-Level</span>
          </div>
          <p className="text-lg font-bold">{data?.bypass_requires_c_level ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Violations</span>
          </div>
          <p className="text-lg font-bold">{data?.sod_violations?.length ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Conflicting Roles Pairs */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Conflicting Role Pairs</h2>
          <div className="space-y-2">
            {(data?.conflicting_roles ?? []).map((pair: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-blue-400">{pair.role_a}</span>
                    <span className="text-gray-500">{"<->"}</span>
                    <span className="text-sm font-medium text-red-400">{pair.role_b}</span>
                  </div>
                  <span className="text-xs px-2 py-0.5 rounded bg-gray-700 text-gray-300 capitalize">{pair.conflict_type}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* SoD Violations */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-5 h-5 text-red-400" />
            Active SoD Violations
          </h2>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {(data?.sod_violations ?? []).map((v: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{v.user}</p>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      v.action_required === "immediate" ? "bg-red-900 text-red-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}
                  >
                    {v.action_required}
                  </span>
                </div>
                <p className="text-xs text-gray-400">
                  {v.conflicting_roles.join(" + ")}
                </p>
                <p className="text-xs text-gray-500 mt-1">Detected: {v.detected_at}</p>
              </div>
            ))}
            {(data?.sod_violations ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No violations detected.</p>
            )}
          </div>
        </div>
      </div>

      {/* Exception Approval Flow */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <CheckCircle className="w-5 h-5 text-green-400" />
          Exception Approval Flow
        </h2>
        <div className="flex items-center gap-2 flex-wrap">
          {[
            "1. Manager Approval",
            "2. Security Officer Review",
            "3. Compliance Sign-off",
            "4. Time-limited Grant (max 30d)",
          ].map((step: any, i: number) => (
            <div key={i} className="flex items-center gap-2">
              <span className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">{step}</span>
              {i < 3 && <span className="text-gray-600">{"->"}</span>}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
