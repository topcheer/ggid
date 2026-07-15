"use client";

import { useDeviceFingerprintAnalytics } from "@ggid/sdk-react";
import { Fingerprint, Smartphone, Globe, AlertTriangle, CheckCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function DeviceFingerprintAnalyticsPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useDeviceFingerprintAnalytics();

  if (loading) return <div className="p-8 text-gray-400">{t("big1.deviceFingerprintAnalytics.loadingFingerprintAnalytics")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("big1.deviceFingerprintAnalytics.error")}{error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("big1.deviceFingerprintAnalytics.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("big1.deviceFingerprintAnalytics.analyzeDeviceFingerprintsForFraudDetectionAndAuthentication")}</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("big1.deviceFingerprintAnalytics.refresh")}</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Fingerprint className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">{t("big1.deviceFingerprintAnalytics.uniqueFingerprints")}</p>
          <p className="text-xl font-bold">{data?.unique_fingerprints?.toLocaleString() ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">{t("big1.deviceFingerprintAnalytics.matchRate")}</p>
          <p className="text-xl font-bold text-green-400">{data?.fingerprint_match_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">{t("big1.deviceFingerprintAnalytics.suspicious")}</p>
          <p className="text-xl font-bold text-red-400">{data?.suspicious_fingerprints?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Smartphone className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">{t("big1.deviceFingerprintAnalytics.clusters")}</p>
          <p className="text-xl font-bold">{data?.fingerprint_clusters?.length ?? 0}</p>
        </div>
      </div>

      {/* Hash Distribution */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">{t("big1.deviceFingerprintAnalytics.canvasHashVsWebglHashDistribution")}</h2>
        <div className="grid grid-cols-2 gap-6">
          <div>
            <p className="text-xs text-gray-400 mb-2">{t("big1.deviceFingerprintAnalytics.canvasHashDiversity")}</p>
            <div className="flex items-end gap-1 h-24">
              {(data?.canvas_hash_distribution ?? []).map((v, i) => {
                const max = Math.max(...(data?.canvas_hash_distribution ?? [1]));
                return <div key={i} className="flex-1 bg-blue-500 rounded-t" style={{ height: max > 0 ? (v / max) * 100 + "%" : "0" }} />;
              })}
            </div>
          </div>
          <div>
            <p className="text-xs text-gray-400 mb-2">{t("big1.deviceFingerprintAnalytics.webglHashDiversity")}</p>
            <div className="flex items-end gap-1 h-24">
              {(data?.webgl_hash_distribution ?? []).map((v, i) => {
                const max = Math.max(...(data?.webgl_hash_distribution ?? [1]));
                return <div key={i} className="flex-1 bg-purple-500 rounded-t" style={{ height: max > 0 ? (v / max) * 100 + "%" : "0" }} />;
              })}
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Fingerprint Clusters */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("big1.deviceFingerprintAnalytics.fingerprintClusters")}</h2>
          <div className="space-y-2">
            {(data?.fingerprint_clusters ?? []).map((c) => (
              <div key={c.browser} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{c.browser} / {c.os}</p>
                  <p className="text-xs text-gray-400">{c.device_count}{t("big1.deviceFingerprintAnalytics.devices")}</p>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-16 h-1.5 bg-gray-700 rounded-full">
                    <div className="h-full bg-blue-500 rounded-full" style={{ width: c.pct + "%" }} />
                  </div>
                  <span className="text-xs text-gray-400">{c.pct}%</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Suspicious Fingerprints */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-4 h-4 text-red-400" />{t("big1.deviceFingerprintAnalytics.suspiciousFingerprints")}</h2>
          <div className="space-y-2 max-h-72 overflow-y-auto">
            {(data?.suspicious_fingerprints ?? []).map((s) => (
              <div key={s.fingerprint_hash} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between mb-1">
                  <p className="text-xs font-mono text-gray-400">{s.fingerprint_hash.substring(0, 24)}...</p>
                  <span className={"text-xs px-1.5 py-0.5 rounded " + (
                    s.reason === "headless_browser" ? "bg-red-900 text-red-300" :
                    s.reason === "spoofed" ? "bg-orange-900 text-orange-300" :
                    "bg-yellow-900 text-yellow-300"
                  )}>
                    {s.reason}
                  </span>
                </div>
                <p className="text-xs text-gray-500">{t("big1.deviceFingerprintAnalytics.user")}{s.associated_user} - {s.timestamp}</p>
              </div>
            ))}
            {(data?.suspicious_fingerprints?.length ?? 0) === 0 && (
              <p className="text-sm text-gray-500">{t("big1.deviceFingerprintAnalytics.noSuspiciousFingerprintsDetected")}</p>
            )}
          </div>
        </div>
      </div>

      {/* Known Good List */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <CheckCircle className="w-4 h-4 text-green-400" />{t("big1.deviceFingerprintAnalytics.knownGoodFingerprintsSample")}</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
          {(data?.known_good_list ?? []).map((f, i) => (
            <div key={i} className="flex items-center gap-2 bg-gray-800 rounded-lg p-2">
              <Fingerprint className="w-3 h-3 text-green-400" />
              <span className="text-xs font-mono text-gray-400">{f.hash.substring(0, 20)}...</span>
              <span className="text-xs text-gray-500 ml-auto">{f.last_seen}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
