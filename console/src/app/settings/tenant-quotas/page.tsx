"use client";

import { useTenantQuotas } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Users, Phone, Database, MonitorSmartphone, KeyRound, AlertTriangle, TrendingUp, ArrowUpCircle } from "lucide-react";

export default function TenantQuotasPage() {
  const { data, loading, error, refresh } = useTenantQuotas();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("tenantQuotas.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const usageIcons: Record<string, React.ReactNode> = {
    users: <Users className="w-4 h-4" />,
    api_calls: <Phone className="w-4 h-4" />,
    storage_mb: <Database className="w-4 h-4" />,
    sessions: <MonitorSmartphone className="w-4 h-4" />,
    tokens_issued: <KeyRound className="w-4 h-4" />,
  };

  const maxTrend = Math.max(...(data?.usage_trend_30d ?? [1]), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("tenantQuotas.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("tenantQuotas.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
            <ArrowUpCircle className="w-4 h-4" />
            Upgrade Plan
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Current Plan Banner */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6 flex items-center justify-between">
        <div>
          <span className="text-sm text-gray-400">{t("tenantQuotas.currentPlan")} </span>
          <span className="text-lg font-bold text-blue-400 capitalize">{data?.current_plan ?? "free"}</span>
        </div>
        <div className="flex items-center gap-4 text-sm text-gray-400">
          <span>{t("tenantQuotas.billingReset")} {data?.days_until_reset ?? 0} {t("tenantQuotas.days")}</span>
        </div>
      </div>

      {/* Usage Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">{t("tenantQuotas.resourceUsage")}</h2>
        <div className="space-y-4">
          {(data?.usage ?? []).map((u) => {
            const pct = u.limit > 0 ? (u.used / u.limit) * 100 : 0;
            const barColor = pct >= 90 ? "bg-red-500" : pct >= 70 ? "bg-yellow-500" : "bg-green-500";
            return (
              <div key={u.resource}>
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2 text-gray-300">
                    {usageIcons[u.resource] ?? <Database className="w-4 h-4" />}
                    <span className="text-sm font-medium capitalize">{u.resource.replace(/_/g, " ")}</span>
                  </div>
                  <span className="text-sm text-gray-400">
                    {u.used.toLocaleString()} / {u.limit.toLocaleString()}
                    <span className={`ml-2 font-medium ${pct >= 90 ? "text-red-400" : pct >= 70 ? "text-yellow-400" : "text-green-400"}`}>
                      ({Math.round(pct)}%)
                    </span>
                  </span>
                </div>
                <div className="w-full bg-gray-700 rounded-full h-2">
                  <div className={barColor + " rounded-full h-2 transition-all"} style={{ width: `${Math.min(pct, 100)}%` }} />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-Plan Limits */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("tenantQuotas.planLimitsComparison")}</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th scope="col" className="text-left py-2 pr-4">{t("tenantQuotas.resource")}</th>
                  {(data?.per_plan_limits ?? []).map((p) => (
                    <th scope="col" key={p.plan} className="text-right py-2 px-2 capitalize">{p.plan}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {(["users", "api_calls", "storage_mb", "sessions", "tokens_issued"] as const).map((res) => (
                  <tr key={res} className="border-b border-gray-800">
                    <td className="py-2 pr-4 text-gray-300 capitalize">{res.replace(/_/g, " ")}</td>
                    {(data?.per_plan_limits ?? []).map((p) => (
                      <td key={p.plan} className="text-right py-2 px-2 text-gray-300">
                        {p.limits[res] === -1 ? "∞" : (p.limits[res] ?? 0).toLocaleString()}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div className="space-y-6">
          {/* Overage Alerts */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-yellow-400" />
              Overage Alerts
            </h2>
            <div className="space-y-2">
              {(data?.overage_alerts ?? []).map((alert: any, i: number) => (
                <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  <AlertTriangle className={"w-4 h-4 flex-shrink-0 " + (alert.severity === "critical" ? "text-red-400" : "text-yellow-400")} />
                  <div className="flex-1">
                    <p className="text-sm font-medium">{alert.resource.replace(/_/g, " ")}</p>
                    <p className="text-xs text-gray-400">{alert.message}</p>
                  </div>
                  <span className={"text-xs px-2 py-0.5 rounded " + (alert.severity === "critical" ? "bg-red-900 text-red-300" : "bg-yellow-900 text-yellow-300")}>
                    {alert.severity}
                  </span>
                </div>
              ))}
              {(data?.overage_alerts ?? []).length === 0 && (
                <p className="text-sm text-gray-500 text-center py-4">{t("tenantQuotas.noOverage")}</p>
              )}
            </div>
          </div>

          {/* Usage Trend */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <TrendingUp className="w-5 h-5 text-blue-400" />
              Usage Trend (30d)
            </h2>
            <div className="flex items-end gap-1 h-32">
              {(data?.usage_trend_30d ?? []).map((v: any, i: number) => (
                <div key={i} className="flex-1 flex flex-col items-center gap-1">
                  <div
                    className="w-full rounded-t bg-blue-500 hover:bg-blue-400 transition-all"
                    style={{ height: `${(v / maxTrend) * 100}%`, minHeight: "2px" }}
                    title={`Day ${i + 1}: ${v}`}
                  />
                  {i % 7 === 0 && <span className="text-xs text-gray-500">{i + 1}</span>}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
