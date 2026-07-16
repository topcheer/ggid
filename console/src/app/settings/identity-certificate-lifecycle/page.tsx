"use client";

import { useIdentityCertificateLifecycle } from "@ggid/sdk-react";
import { Award, RefreshCw, AlertTriangle, FileText, Ban } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityCertificateLifecyclePage() {
  const { data, loading, error, refresh } = useIdentityCertificateLifecycle();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idCertLifecycle.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idCertLifecycle.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idCertLifecycle.subtitle")}</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("idCertLifecycle.refresh")}</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Award className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">{t("idCertLifecycle.totalCerts")}</p>
          <p className="text-xl font-bold">{data?.certificates?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <RefreshCw className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">{t("idCertLifecycle.pendingRenewal")}</p>
          <p className="text-xl font-bold text-yellow-400">{data?.renewal_queue?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">{t("idCertLifecycle.expiring30d")}</p>
          <p className="text-xl font-bold text-red-400">{data?.expiry_calendar?.filter((e) => e.days_until <= 30).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Ban className="w-5 h-5 text-gray-400 mb-1" />
          <p className="text-xs text-gray-400">{t("idCertLifecycle.revoked")}</p>
          <p className="text-xl font-bold">{data?.revocation_list?.length ?? 0}</p>
        </div>
      </div>

      {/* Certificates Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">{t("idCertLifecycle.certificates")}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">{t("idCertLifecycle.name")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("idCertLifecycle.type")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("idCertLifecycle.issuer")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("idCertLifecycle.serial")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("idCertLifecycle.validTo")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("idCertLifecycle.autoRenew")}</th>
              </tr>
            </thead>
            <tbody>
              {(data?.certificates ?? []).map((c) => (
                <tr key={c.serial} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-sm font-medium">{c.name}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs px-2 py-0.5 rounded bg-gray-800">{c.type}</span>
                  </td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{c.issuer}</td>
                  <td className="py-3 pr-3 font-mono text-xs text-gray-500">{c.serial.substring(0, 16)}...</td>
                  <td className="py-3 pr-3 text-xs">
                    <span className={c.days_to_expiry < 30 ? "text-red-400" : c.days_to_expiry < 90 ? "text-yellow-400" : "text-gray-400"}>
                      {c.valid_to} ({c.days_to_expiry}d)
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.auto_renew_days_before > 0 ? c.auto_renew_days_before + "d before" : "Manual"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Renewal Queue */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <RefreshCw className="w-4 h-4 text-yellow-400" />
            Renewal Queue
          </h2>
          <div className="space-y-2">
            {(data?.renewal_queue ?? []).map((r) => (
              <div key={r.name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{r.name}</p>
                  <p className="text-xs text-gray-400">Renews in {r.days_until_renewal} days</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (
                  r.days_until_renewal < 7 ? "bg-red-900 text-red-300" : "bg-yellow-900 text-yellow-300"
                )}>
                  {r.status}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Revocation List */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Ban className="w-4 h-4 text-red-400" />
            Revocation List
          </h2>
          <div className="space-y-2">
            {(data?.revocation_list ?? []).map((r) => (
              <div key={r.serial} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                <FileText className="w-3 h-3 text-gray-500" />
                <div className="flex-1">
                  <p className="text-xs font-mono">{r.serial.substring(0, 20)}...</p>
                  <p className="text-xs text-gray-500">Revoked: {r.revoked_at} - {r.reason}</p>
                </div>
              </div>
            ))}
            {(data?.revocation_list?.length ?? 0) === 0 && (
              <p className="text-xs text-gray-500">{t("idCertLifecycle.noRevoked")}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
