"use client";

import { usePasskeyHealth } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Fingerprint, Smartphone, CheckCircle, Clock, Shield } from "lucide-react";

export default function PasskeyHealthPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = usePasskeyHealth();

  if (loading) return <div className="p-8 text-gray-400">{t("passkeyHealth.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const platColors: Record<string, string> = {
    iOS: "#22c55e",
    Android: "#eab308",
    Windows: "#3b82f6",
    macOS: "#a855f7",
  };

  const platData = data?.platform_distribution ?? { iOS: 0, Android: 0, Windows: 0, macOS: 0 };
  const platEntries: [string, number][] = Object.entries(platData) as [string, number][];
  const totalUsers = platEntries.reduce((a, [, c]) => a + c, 0);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("passkeyHealth.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("passkeyHealth.subtitle")}</p>
        </div>
        <button aria-label="action" onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("common.refresh")}</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Fingerprint className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">{t("passkeyHealth.totalPasskeys")}</p>
          <p className="text-xl font-bold">{data?.registered_passkeys?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">{t("passkeyHealth.adoptionRate")}</p>
          <p className="text-xl font-bold text-green-400">{data?.adoption_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">{t("passkeyHealth.stalePasskeys")}</p>
          <p className="text-xl font-bold text-yellow-400">{data?.stale_passkeys?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">{t("passkeyHealth.backupEligible")}</p>
          <p className="text-xl font-bold">{data?.registered_passkeys?.filter((p) => p.backup_eligible).length ?? 0}</p>
        </div>
      </div>

      {/* Adoption Gauge + Platform Donut */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("passkeyHealth.adoptionRate")}</h2>
          <div className="relative w-32 h-32 mx-auto">
            <svg className="w-32 h-32 -rotate-90" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="40" fill="none" stroke="#374151" strokeWidth="12" />
              <circle
                cx="50" cy="50" r="40"
                fill="none"
                stroke="#22c55e"
                strokeWidth="12"
                strokeDasharray={((data?.adoption_rate_pct ?? 0) / 100 * 251.2) + " " + 251.2}
                strokeLinecap="round"
              />
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-2xl font-bold text-green-400">{data?.adoption_rate_pct ?? 0}%</span>
            </div>
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("passkeyHealth.platformDist")}</h2>
          <div className="relative w-32 h-32 mx-auto">
            <svg className="w-32 h-32 -rotate-90" viewBox="0 0 100 100">
              {(() => {
                let offset = 0;
                return platEntries.map(([platform, count]) => {
                  const pct = totalUsers > 0 ? count / totalUsers : 0;
                  const dash = pct * 251.2;
                  const el = (
                    <circle
                      key={platform}
                      cx="50" cy="50" r="40"
                      fill="none"
                      stroke={platColors[platform] ?? "#6b7280"}
                      strokeWidth="12"
                      strokeDasharray={dash + " " + (251.2 - dash)}
                      strokeDashoffset={-offset}
                    />
                  );
                  offset += dash;
                  return el;
                });
              })()}
            </svg>
          </div>
          <div className="mt-4 space-y-1">
            {platEntries.map(([platform, count]) => (
              <div key={platform} className="flex items-center gap-2 text-xs">
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: platColors[platform] ?? "#6b7280" }} />
                <span className="text-gray-400">{platform}</span>
                <span className="font-medium ml-auto">{count as number}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Recovery Options</h2>
          <div className="space-y-2">
            {(data?.recovery_options_config ?? []).map((opt) => (
              <div key={opt.method} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm">{opt.method}</span>
                <span className={"text-xs px-2 py-0.5 rounded " + (opt.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                  {opt.enabled ? t("rateLimits.on") : t("rateLimits.off")}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Registered Passkeys */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Registered Passkeys</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">User</th>
                <th scope="col" className="text-left py-2 pr-3">Device</th>
                <th scope="col" className="text-left py-2 pr-3">Platform</th>
                <th scope="col" className="text-left py-2 pr-3">Created</th>
                <th scope="col" className="text-left py-2 pr-3">Last Used</th>
                <th scope="col" className="text-left py-2 pr-3">Backup</th>
              </tr>
            </thead>
            <tbody>
              {(data?.registered_passkeys ?? []).map((p) => (
                <tr key={p.id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-sm font-medium">{p.user}</td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-1">
                      <Smartphone className="w-3 h-3 text-gray-500" />
                      <span className="text-xs">{p.device}</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-xs">{p.platform}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{p.created_at}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{p.last_used}</td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-1">
                      {p.backup_eligible ? (
                        <span className={"text-xs " + (p.backup_state === "synced" ? "text-green-400" : "text-yellow-400")}>
                          {p.backup_eligible ? t("passkeyHealth.eligible") : ""} ({p.backup_state})
                        </span>
                      ) : (
                        <span className="text-xs text-gray-500">Not eligible</span>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
