"use client";

import { useState } from "react";
import { useIdentityUserAttestation } from "@ggid/sdk-react";
import { CheckCircle, XCircle, Clock, Users, AlertTriangle, ShieldCheck } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityUserAttestationPage() {
  const { data, loading, error, refresh, bulkAttest } = useIdentityUserAttestation();
  const [selectedCampaign, setSelectedCampaign] = useState("");
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idUserAttestation.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const campaign = selectedCampaign
    ? (data?.campaigns ?? []).find((c: any) => c.id === selectedCampaign)
    : (data?.campaigns ?? [])[0];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idUserAttestation.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idUserAttestation.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idUserAttestation.pending")}</span>
          </div>
          <p className="text-2xl font-bold text-yellow-400">{campaign?.pending_count ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idUserAttestation.attested")}</span>
          </div>
          <p className="text-2xl font-bold text-green-400">{campaign?.attested_count ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idUserAttestation.overdue")}</span>
          </div>
          <p className="text-2xl font-bold text-red-400">{data?.overdue_attestations ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <ShieldCheck className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idUserAttestation.autoRevokeAfter")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.auto_revoke_unattested_days ?? 0}d</p>
        </div>
      </div>

      {/* Campaign Selector */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center gap-3">
          <label className="text-sm text-gray-400">{t("idUserAttestation.campaign")}</label>
          <select
            value={selectedCampaign || campaign?.id || ""}
            onChange={(e) => setSelectedCampaign(e.target.value)}
            className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          >
            {(data?.campaigns ?? []).map((c: any) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          <button
            onClick={() => bulkAttest(campaign?.id ?? "")}
            className="flex items-center gap-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <Users className="w-4 h-4" />
            Bulk Attest All
          </button>
        </div>
      </div>

      {/* User List */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">{t("idUserAttestation.status")}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-4">{t("idUserAttestation.user")}</th>
                <th scope="col" className="text-left py-2 pr-4">{t("idUserAttestation.statusLabel")}</th>
                <th scope="col" className="text-left py-2 pr-4">{t("idUserAttestation.lastAttested")}</th>
                <th scope="col" className="text-left py-2 pr-4">{t("idUserAttestation.attestedBy")}</th>
                <th scope="col" className="text-left py-2 pr-4">{t("idUserAttestation.permissions")}</th>
              </tr>
            </thead>
            <tbody>
              {(campaign?.user_list ?? []).map((u: any) => (
                <tr key={u.user_id} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-medium">{u.user_name}</td>
                  <td className="py-3 pr-4">
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        u.attestation_status === "attested" ? "bg-green-900 text-green-300" :
                        u.attestation_status === "revoked" ? "bg-red-900 text-red-300" :
                        "bg-yellow-900 text-yellow-300"
                      )}
                    >
                      {u.attestation_status}
                    </span>
                  </td>
                  <td className="py-3 pr-4 text-gray-300">{u.last_attested_at ?? "Never"}</td>
                  <td className="py-3 pr-4 text-gray-300">{u.attested_by ?? "-"}</td>
                  <td className="py-3 pr-4 text-gray-300">{u.permissions_at_time}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
