"use client";

import { useOAuthJarConfig } from "@ggid/sdk-react";
import { Shield, FileJson, Clock, BarChart3, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthJarConfigPage() {

  const { data, loading, error, refresh } = useOAuthJarConfig();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("oauthJarConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("oauthJarConfig.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("oauthJarConfig.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Config Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("oauthJarConfig.requireJAR")}</span>
          </div>
          <p className="text-lg font-bold">{data?.require_jar ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("oauthJarConfig.jarLifetime")}</span>
          </div>
          <p className="text-lg font-bold">{data?.jar_lifetime_seconds ?? 0}s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <FileJson className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("oauthJarConfig.signingAlgorithm")}</span>
          </div>
          <p className="text-lg font-bold font-mono">{data?.signing_alg ?? "RS256"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("oauthJarConfig.encryptionOptional")}</span>
          </div>
          <p className="text-lg font-bold">{data?.encryption_optional ? "Yes" : "No"}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-Client Override Table */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("oauthJarConfig.perClientOverride")}</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th scope="col" className="text-left py-2 pr-3">{t("oauthJarConfig.clientId")}</th>
                  <th scope="col" className="text-left py-2 pr-3">{t("oauthJarConfig.jarRequired")}</th>
                  <th scope="col" className="text-left py-2 pr-3">{t("oauthJarConfig.signingAlg")}</th>
                  <th scope="col" className="text-left py-2 pr-3">{t("oauthJarConfig.lifetime")}</th>
                </tr>
              </thead>
              <tbody>
                {(data?.per_client_override ?? []).map((c: any) => (
                  <tr key={c.client_id} className="border-b border-gray-800">
                    <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.client_id}</td>
                    <td className="py-3 pr-3">
                      <span className={"text-xs px-2 py-0.5 rounded " + (c.jar_required ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                        {c.jar_required ? "Yes" : "No"}
                      </span>
                    </td>
                    <td className="py-3 pr-3 font-mono text-gray-300">{c.signing_alg}</td>
                    <td className="py-3 pr-3 text-gray-300">{c.lifetime_seconds}s</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* JAR Usage Stats */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <BarChart3 className="w-5 h-5 text-blue-400" />
            JAR Usage Stats
          </h2>
          <div className="space-y-2">
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">{t("oauthJarConfig.totalRequests24h")}</span>
              <span className="text-sm font-bold">{data?.jar_usage_stats.total_requests_24h.toLocaleString() ?? 0}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">{t("oauthJarConfig.jarRequests24h")}</span>
              <span className="text-sm font-bold text-green-400">{data?.jar_usage_stats.jar_requests_24h.toLocaleString() ?? 0}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">{t("oauthJarConfig.adoptionRate")}</span>
              <span className="text-sm font-bold text-blue-400">{data?.jar_usage_stats.adoption_rate_pct ?? 0}%</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">{t("oauthJarConfig.validationFailures")}</span>
              <span className="text-sm font-bold text-red-400">{data?.jar_usage_stats.validation_failures_24h ?? 0}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Request Object Preview */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <FileJson className="w-5 h-5 text-purple-400" />
          Request Object Preview
        </h2>
        <div className="bg-gray-800 rounded-lg p-4 overflow-x-auto">
          <pre className="text-xs font-mono text-gray-300 whitespace-pre-wrap">{JSON.stringify(data?.request_object_preview ?? {}, null, 2)}</pre>
        </div>
      </div>
    </div>
  );
}
