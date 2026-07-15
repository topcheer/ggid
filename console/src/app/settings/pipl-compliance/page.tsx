"use client";

import { usePiplCompliance } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Globe, Shield, FileCheck, AlertTriangle, UserCheck, Clock } from "lucide-react";

export default function PiplCompliancePage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = usePiplCompliance();

  if (loading) return <div className="p-8 text-gray-400">Loading PIPL compliance...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">PIPL Compliance</h1>
          <p className="text-sm text-gray-400 mt-1">China Personal Information Protection Law compliance management</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <FileCheck className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Compliance Status</p>
          <p className="text-sm font-bold">{data?.compliance_status ?? "Pending"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <UserCheck className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">DPO Assigned</p>
          <p className="text-sm font-bold">{data?.data_protection_officer?.name ?? "N/A"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Cross-Border Applications</p>
          <p className="text-xl font-bold">{data?.cross_border_transfer_applications?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Globe className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Consent Records</p>
          <p className="text-xl font-bold">{data?.chinese_user_consent_log?.length ?? 0}</p>
        </div>
      </div>

      {/* Data Handling Rules */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Shield className="w-5 h-5 text-blue-400" />
          Data Handling Rules
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {[
            { label: "Consent Required", value: data?.data_handling_rules?.consent_required },
            { label: "Data Minimization", value: data?.data_handling_rules?.data_minimization },
            { label: "Purpose Limitation", value: data?.data_handling_rules?.purpose_limitation },
            { label: "Cross-Border Assessment", value: data?.data_handling_rules?.cross_border_assessment },
          ].map((rule) => (
            <div key={rule.label} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm font-medium">{rule.label}</span>
              <span className={"text-xs px-2 py-0.5 rounded " + (rule.value ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                {rule.value ? "Enabled" : "Disabled"}
              </span>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Cross-Border Transfer Applications */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Globe className="w-4 h-4 text-purple-400" />
            Cross-Border Transfer Applications
          </h2>
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {(data?.cross_border_transfer_applications ?? []).map((app, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between mb-1">
                  <p className="text-sm font-medium">{app.applicant}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    app.status === "approved" ? "bg-green-900 text-green-300" :
                    app.status === "pending" ? "bg-yellow-900 text-yellow-300" :
                    "bg-red-900 text-red-300"
                  )}>
                    {app.status}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1 mb-1">
                  <span className="text-xs text-gray-400">Data: {app.data_type}</span>
                  <span className="text-xs text-gray-400">Destination: {app.recipient_country}</span>
                </div>
                <p className="text-xs text-gray-500">Assessment: {app.assessment_result}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Consent Log */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <UserCheck className="w-4 h-4 text-blue-400" />
            Chinese User Consent Log
          </h2>
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {(data?.chinese_user_consent_log ?? []).map((log, i) => (
              <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                <div className="flex-1">
                  <p className="text-xs font-medium">{log.user}</p>
                  <p className="text-xs text-gray-400">{log.purpose}</p>
                </div>
                <div className="text-right">
                  <span className={"text-xs px-1.5 py-0.5 rounded " + (log.withdrawn ? "bg-red-900 text-red-300" : "bg-green-900 text-green-300")}>
                    {log.withdrawn ? "Withdrawn" : "Active"}
                  </span>
                  <p className="text-xs text-gray-500 mt-0.5">{log.timestamp}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Data Retention Compliance */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Clock className="w-4 h-4 text-yellow-400" />
          Data Retention Compliance
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {(data?.data_retention_compliance ?? []).map((item, i) => (
            <div key={i} className="bg-gray-800 rounded-lg p-3">
              <p className="text-sm font-medium">{item.data_category}</p>
              <div className="flex items-center justify-between mt-1">
                <span className="text-xs text-gray-400">Policy: {item.policy_days} days</span>
                <span className={"text-xs px-2 py-0.5 rounded " + (item.compliant ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                  {item.compliant ? "Compliant" : "Non-Compliant"}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
