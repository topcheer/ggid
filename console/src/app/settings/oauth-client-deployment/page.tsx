"use client";

import { useOAuthClientDeployment } from "@ggid/sdk-react";
import { Rocket, RotateCcw, GitCompare, Download, HeartPulse, CheckCircle, Clock, ArrowUp } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthClientDeploymentPage() {

  const { data, loading, error, refresh, promote, rollback } = useOAuthClientDeployment();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("oauthClientDeploy.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("oauthClientDeploy.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("oauthClientDeploy.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" />
            Export Config
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Environments */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        {(data?.environments ?? []).map((env: any) => (
          <div key={env.name} className="bg-gray-900 rounded-xl p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                {env.name === "prod" ? (
                  <Rocket className="w-5 h-5 text-green-400" />
                ) : env.name === "staging" ? (
                  <GitCompare className="w-5 h-5 text-yellow-400" />
                ) : (
                  <Clock className="w-5 h-5 text-blue-400" />
                )}
                <h3 className="font-semibold capitalize">{env.name}</h3>
              </div>
              <span
                className={"text-xs px-2 py-0.5 rounded " + (
                  env.health === "healthy" ? "bg-green-900 text-green-300" :
                  env.health === "degraded" ? "bg-yellow-900 text-yellow-300" :
                  "bg-red-900 text-red-300"
                )}
              >
                {env.health}
              </span>
            </div>
            <div className="space-y-1 text-xs">
              <div className="flex justify-between">
                <span className="text-gray-400">{t("oauthClientDeploy.version")}</span>
                <span className="font-medium">{env.version}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">{t("oauthClientDeploy.lastDeploy")}</span>
                <span>{env.last_deploy}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-400">{t("oauthClientDeploy.activeClients")}</span>
                <span className="font-medium">{env.active_clients}</span>
              </div>
            </div>
            {env.name !== "prod" && (
              <button
                onClick={() => promote(env.name)}
                className="w-full mt-3 flex items-center justify-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-md text-xs font-medium transition"
              >
                <ArrowUp className="w-3 h-3" />
                Promote
              </button>
            )}
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Config Diff */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <GitCompare className="w-5 h-5 text-blue-400" />
            Config Diff (Staging → Prod)
          </h2>
          <div className="space-y-2">
            {(data?.config_diff ?? []).map((d: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-mono text-blue-400">{d.field}</span>
                  <span
                    className={"text-xs px-1.5 py-0.5 rounded " + (
                      d.change_type === "added" ? "bg-green-900 text-green-300" :
                      d.change_type === "removed" ? "bg-red-900 text-red-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}
                  >
                    {d.change_type}
                  </span>
                </div>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <span className="text-gray-500">{t("oauthClientDeploy.from")}</span>
                    <span className="text-red-400">{d.old_value || "-"}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">{t("oauthClientDeploy.to")}</span>
                    <span className="text-green-400">{d.new_value || "-"}</span>
                  </div>
                </div>
              </div>
            ))}
            {(data?.config_diff ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">{t("oauthClientDeploy.noDiff")}</p>
            )}
          </div>
        </div>

        <div className="space-y-6">
          {/* Deployment History */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">{t("oauthClientDeploy.deploymentHistory")}</h2>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {(data?.deployment_history ?? []).map((dep: any, i: number) => (
                <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  <div className={"w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 " + (
                    dep.status === "success" ? "bg-green-900 text-green-300" :
                    dep.status === "failed" ? "bg-red-900 text-red-300" :
                    "bg-yellow-900 text-yellow-300"
                  )}>
                    {dep.status === "success" ? <CheckCircle className="w-4 h-4" /> :
                     dep.status === "failed" ? <RotateCcw className="w-4 h-4" /> :
                     <Clock className="w-4 h-4" />}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium">{dep.environment} - v{dep.version}</p>
                    <p className="text-xs text-gray-400">{dep.deployed_by} - {dep.timestamp}</p>
                  </div>
                  {dep.rollback_available && (
                    <button
                      onClick={() => rollback(dep.id)}
                      className="text-xs px-2 py-1 bg-gray-700 hover:bg-red-700 rounded transition"
                    >
                      Rollback
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Health Check */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <HeartPulse className="w-5 h-5 text-red-400" />
              Health Check
            </h2>
            <div className="space-y-2">
              {(data?.health_checks ?? []).map((hc: any, i: number) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                  <span className="text-sm text-gray-300">{hc.check}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-400">{hc.latency_ms}ms</span>
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        hc.status === "pass" ? "bg-green-900 text-green-300" :
                        hc.status === "warn" ? "bg-yellow-900 text-yellow-300" :
                        "bg-red-900 text-red-300"
                      )}
                    >
                      {hc.status}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
