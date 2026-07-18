"use client";
import { useTranslations } from "@/lib/i18n";

import { useCertExpiryTracker } from "@ggid/sdk-react";
import { Shield, AlertTriangle, CheckCircle, Calendar } from "lucide-react";

export default function CertExpiryTrackerPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useCertExpiryTracker();

  if (loading) return <div className="p-8 text-gray-400">Loading cert expiry tracker...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("backend.certExpiry.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor certificate expiration and auto-renewal status</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Healthy (&gt;90d)</p>
          <p className="text-xl font-bold text-green-400">{data?.certs?.filter((c: any) => c.days_remaining > 90).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Calendar className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Expiring (&lt;90d)</p>
          <p className="text-xl font-bold text-yellow-400">{data?.certs?.filter((c: any) => c.days_remaining <= 90 && c.days_remaining > 30).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-orange-400 mb-1" />
          <p className="text-xs text-gray-400">Critical (&lt;30d)</p>
          <p className="text-xl font-bold text-orange-400">{data?.certs?.filter((c: any) => c.days_remaining <= 30 && c.days_remaining > 0).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">{t("backend.certExpiry.expired")}</p>
          <p className="text-xl font-bold text-red-400">{data?.certs?.filter((c: any) => c.days_remaining <= 0).length ?? 0}</p>
        </div>
      </div>

      {/* Cert Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">{t("backend.certExpiry.certificates")}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Name</th>
                <th scope="col" className="text-left py-2 pr-3">Type</th>
                <th scope="col" className="text-left py-2 pr-3">{t("backend.certExpiry.issuer")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("backend.certExpiry.expiry")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("backend.certExpiry.daysLeft")}</th>
                <th scope="col" className="text-left py-2 pr-3">Auto-Renew</th>
              </tr>
            </thead>
            <tbody>
              {(data?.certs ?? []).map((c: any) => (
                <tr key={c.name} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-medium">{c.name}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.type}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.issuer}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.expiry_date}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs font-bold " + (
                      c.days_remaining <= 0 ? "text-red-400" :
                      c.days_remaining <= 30 ? "text-orange-400" :
                      c.days_remaining <= 90 ? "text-yellow-400" : "text-green-400"
                    )}>{c.days_remaining > 0 ? c.days_remaining + "d" : "EXPIRED"}</span>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (c.auto_renewal_enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                      {c.auto_renewal_enabled ? "Enabled" : "Disabled"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Alert Config */}
      {data?.alert_config && (
        <div className="bg-gray-900 rounded-xl p-6 mt-6">
          <h2 className="text-sm font-semibold mb-3">{t("backend.certExpiry.alertConfig")}</h2>
          <div className="flex items-center gap-4">
            <div>
              <p className="text-xs text-gray-500">{t("backend.certExpiry.firstAlert")}</p>
              <p className="text-sm">{data.alert_config.first_alert_days}d before expiry</p>
            </div>
            <div>
              <p className="text-xs text-gray-500">{t("backend.certExpiry.escalation")}</p>
              <p className="text-sm">{data.alert_config.escalation_days}d before expiry</p>
            </div>
            <div>
              <p className="text-xs text-gray-500">{t("backend.certExpiry.channel")}</p>
              <p className="text-sm">{data.alert_config.channels.join(", ")}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
